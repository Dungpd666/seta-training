package user

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailInUse         = errors.New("email already in use")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

var expectedHeader = []string{"username", "email", "password", "role"}

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindAll(ctx context.Context) ([]User, error)
}

type Service interface {
	Register(ctx context.Context, username, email, password, role string) (*User, error)
	Login(ctx context.Context, email, password string) (*User, error)
	ListAll(ctx context.Context) ([]User, error)
	ImportFromCSV(ctx context.Context, r io.Reader, workers int) (*ImportResult, error)
}

type CSVRow struct {
	LineNo   int
	Username string
	Email    string
	Password string
	Role     string
}

type ImportError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

type ImportResult struct {
	Succeeded int           `json:"succeeded"`
	Failed    int           `json:"failed"`
	Errors    []ImportError `json:"errors"`
}

type rowResult struct {
	lineNo int
	err    error
}

type service struct {
	repo           Repository
	defaultWorkers int
}

type Option func(*service)

func NewService(repo Repository, opts ...Option) Service {
	s := &service{
		repo:           repo,
		defaultWorkers: 5,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *service) ListAll(ctx context.Context) ([]User, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) Register(ctx context.Context, username, email, password, role string) (*User, error) {
	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailInUse
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *service) Login(ctx context.Context, email, password string) (*User, error) {
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

func (s *service) ImportFromCSV(ctx context.Context, r io.Reader, workers int) (*ImportResult, error) {
	rows, err := parseCSV(r)
	if err != nil {
		return nil, err
	}

	if workers <= 0 {
		workers = s.defaultWorkers
	}
	if workers > len(rows) && len(rows) > 0 {
		workers = len(rows)
	}

	jobs := make(chan CSVRow, len(rows))
	results := make(chan rowResult, len(rows))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go s.importWorker(ctx, &wg, jobs, results)
	}

	for _, row := range rows {
		jobs <- row
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	result := &ImportResult{Errors: []ImportError{}}
	for r := range results {
		if r.err != nil {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:    r.lineNo,
				Reason: r.err.Error(),
			})
		} else {
			result.Succeeded++
		}
	}

	sort.Slice(result.Errors, func(i, j int) bool {
		return result.Errors[i].Row < result.Errors[j].Row
	})

	return result, nil
}

func parseCSV(r io.Reader) ([]CSVRow, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = len(expectedHeader)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if err := validateHeader(header); err != nil {
		return nil, err
	}

	var rows []CSVRow
	lineNo := 1
	for {
		lineNo++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		rows = append(rows, CSVRow{
			LineNo:   lineNo,
			Username: strings.TrimSpace(record[0]),
			Email:    strings.TrimSpace(record[1]),
			Password: record[2],
			Role:     strings.TrimSpace(record[3]),
		})
	}
	return rows, nil
}

func validateHeader(header []string) error {
	if len(header) != len(expectedHeader) {
		return fmt.Errorf("expected %d columns, got %d", len(expectedHeader), len(header))
	}
	for i, col := range header {
		if !strings.EqualFold(strings.TrimSpace(col), expectedHeader[i]) {
			return fmt.Errorf("column %d: expected %q, got %q", i+1, expectedHeader[i], col)
		}
	}
	return nil
}

func (s *service) importWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs <-chan CSVRow,
	results chan<- rowResult,
) {
	defer wg.Done()

	for row := range jobs {
		select {
		case <-ctx.Done():
			results <- rowResult{lineNo: row.LineNo, err: ctx.Err()}
			return
		default:
		}

		_, err := s.Register(ctx, row.Username, row.Email, row.Password, row.Role)
		results <- rowResult{lineNo: row.LineNo, err: err}
	}
}

func WithWorkers(n int) Option {
	return func(s *service) {
		if n > 0 {
			s.defaultWorkers = n
		}
	}
}
