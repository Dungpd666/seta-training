# Giải thích Chi tiết Code Auth-Service

Tài liệu này mô tả cấu trúc và logic của mã nguồn `auth-service`.

---

## 1) Cấu trúc dự án

```
auth-service/
├── cmd/
│   └── main.go              # Điểm vào chính: khởi tạo, thiết lập Gin, load keys
├── internal/
│   ├── handler/
│   │   └── user_handler.go  # HTTP handlers cho các endpoint
│   ├── service/
│   │   ├── auth_service.go  # Logic token (JWT, refresh, revoke)
│   │   ├── user_service.go  # Logic user (register, login)
│   │   └── interfaces.go    # Interface definitions
│   ├── repository/
│   │   ├── user_repository.go          # DB operations cho users
│   │   └── refresh_token_repository.go # DB operations cho refresh tokens
│   └── model/
│       ├── user.go          # Struct User
│       └── refresh_token.go # Struct RefreshToken
├── migrations/              # SQL migration files
├── go.mod                   # Dependency management
├── jwt_rs256.pem           # Private key (để ký token)
├── jwt_rs256.pub           # Public key (để verify token)
└── main                     # Compiled binary
```

---

## 2) Clean Architecture Pattern

Code tuân theo **Clean Architecture** với 3 tầng chính:

```
┌─────────────┐
│  handler    │  HTTP endpoints, request/response
├─────────────┤
│  service    │  Business logic, validation, token generation
├─────────────┤
│ repository  │  Database operations (CRUD)
├─────────────┤
│  model      │  Struct definition
└─────────────┘
```

Hướng phụ thuộc: từ handler → service → repository → model (1 chiều)

---

## 3) Models (Dữ liệu)

### File: `internal/model/user.go`

```go
type User struct {
    UserID       string    `gorm:"column:user_id;primaryKey"`
    Username     string    `gorm:"column:username"`
    Email        string    `gorm:"column:email"`
    PasswordHash string    `gorm:"column:password_hash"`
    Role         string    `gorm:"column:role"`
    CreatedAt    time.Time `gorm:"column:created_at"`
}
```

- `UserID`: UUID sinh ngẫu nhiên trong DB
- `PasswordHash`: Hash bcrypt, không lưu plaintext
- `Role`: Giới hạn `manager` hoặc `member`
- Tương ứng bảng `users` trong PostgreSQL

### File: `internal/model/refresh_token.go`

```go
type RefreshToken struct {
    JTI       string    `gorm:"column:jti;primaryKey"`
    UserID    string    `gorm:"column:user_id"`
    ExpiresAt time.Time `gorm:"column:expires_at"`
    Revoked   bool      `gorm:"column:revoked"`
}
```

- `JTI`: Định danh token (UUID)
- `UserID`: FK tới user
- `Revoked`: Đánh dấu token đã bị hủy (logout, reuse detected)
- Tương ứng bảng `refresh_tokens` trong PostgreSQL

---

## 4) Repository Layer (Database Access)

### File: `internal/repository/user_repository.go`

Chức năng: Đọc/ghi user vào DB qua GORM

**Phương thức chính:**

1. **`Create(user *model.User) error`**
   - Tạo record user mới
   - GORM tự sinh UUID cho `user_id`

2. **`FindByEmail(email string) (*model.User, error)`**
   - Tìm user theo email
   - Trả `nil` nếu không tìm thấy
   - Các endpoint login dùng hàm này

3. **`FindAll() ([]model.User, error)`**
   - Lấy danh sách tất cả user
   - Không dùng trong auth-service hiện tại (dành cho API công khai)

### File: `internal/repository/refresh_token_repository.go`

Chức năng: Quản lý vòng đời refresh token

**Phương thức chính:**

1. **`Insert(rt *model.RefreshToken) error`**
   - Lưu refresh token mới sau khi login
   - TTL 7 ngày

2. **`MarkRevoked(jti string) error`**
   - Đánh dấu token là revoked (logout, reuse, refresh)
   - Cập nhật cột `revoked = true`

3. **`IsValid(jti string) (bool, error)`**
   - Kiểm tra token còn hợp lệ không
   - Điều kiện: `revoked = false` AND `expires_at > NOW()`

4. **`RevokeAllForUser(userID string) error`**
   - Hủy toàn bộ token của 1 user
   - Dùng khi phát hiện refresh token reuse → session bị compromise

---

## 5) Service Layer (Business Logic)

### File: `internal/service/user_service.go`

Chức năng: Quản lý user registration & login

```go
type UserService struct {
    repo UserRepo
}
```

**Phương thức chính:**

#### `Register(username, email, password, role string) (*model.User, error)`

**Logic:**

```
1. FindByEmail(email) → kiểm tra email đã dùng chưa
2. Nếu tồn tại → return error "email already in use"
3. bcrypt.GenerateFromPassword(password) → hash mật khẩu
4. Tạo struct User mới với password_hash
5. repo.Create(user) → lưu vào DB
6. Return user (không return password_hash)
```

**Validation:**
- Email: không duplicate
- Password: tối thiểu 6 ký tự (được check ở handler via binding)
- Role: `manager` hoặc `member` (được check ở handler via binding)

#### `Login(email, password string) (*model.User, error)`

**Logic:**

```
1. FindByEmail(email) → lấy user từ DB
2. Nếu user = nil → return error "invalid credentials"
3. bcrypt.CompareHashAndPassword(hash, password) → verify password
4. Nếu không match → return error "invalid credentials"
5. Return user (không return password_hash)
```

**Lưu ý:**
- Trả cùng một thông báo lỗi cho cả email không tồn tại và password sai → chống brute force (user enumeration)
- Ngoài password_hash, user object chứa đủ `UserID` và `Role` để tạo JWT

### File: `internal/service/auth_service.go`

Chức năng: JWT token generation, verification, rotation, revocation

```go
type AuthService struct {
    refreshRepo RefreshTokenRepo
    privateKey  *rsa.PrivateKey
    publicKey   *rsa.PublicKey
    redis       *redis.Client
}

type Claims struct {
    Role string `json:"role"`
    Type string `json:"typ,omitempty"`
    jwt.RegisteredClaims
}
```

**Hằng số:**

```go
const (
    Issuer          = "auth-service"
    Audience        = "seta"
    AccessTokenTTL  = 15 * time.Minute
    refreshTokenTTL = 7 * 24 * time.Hour
)
```

#### `GenerateTokenPair(userID, role string) (accessToken, refreshToken string, err error)`

**Logic tạo access token:**

```go
accessClaims := Claims{
    Role: role,
    RegisteredClaims: jwt.RegisteredClaims{
        Subject:   userID,                      // sub
        ID:        uuid.NewString(),            // jti
        IssuedAt:  jwt.NewNumericDate(now),     // iat
        ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),  // exp (15 min)
        Issuer:    Issuer,                      // iss = "auth-service"
        Audience:  jwt.ClaimStrings{Audience},  // aud = "seta"
    },
}
accessToken = jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
```

**Logic tạo refresh token:**

```go
refreshJTI := uuid.NewString()  // Mỗi refresh token có jti duy nhất
refreshClaims := Claims{
    Type: "refresh",  // Đánh dấu loại token
    RegisteredClaims: jwt.RegisteredClaims{
        Subject:   userID,
        ID:        refreshJTI,
        IssuedAt:  jwt.NewNumericDate(now),
        ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL)),  // exp (7 ngày)
        Issuer:    Issuer,
        Audience:  jwt.ClaimStrings{Audience},
    },
}
refreshToken = jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims).SignedString(privateKey)
```

**Sau đó lưu refresh token vào DB:**

```go
err = s.refreshRepo.Insert(&model.RefreshToken{
    JTI:       refreshJTI,
    UserID:    userID,
    ExpiresAt: now.Add(refreshTokenTTL),
})
```

#### `RotateRefreshToken(tokenStr string) (accessToken, refreshToken string, err error)`

**Logic (cơ chế refresh token rotation):**

```
1. ParseToken(tokenStr) → verify chữ ký + claims
2. Kiểm tra type = "refresh" (nếu không → error)
3. refreshRepo.IsValid(jti) → kiểm tra trong DB:
   - revoked = false?
   - expires_at > now()?
4. Nếu không hợp lệ:
   → refreshRepo.RevokeAllForUser(userID) → hủy toàn bộ token (session compromise!)
   → return error "refresh token reuse detected"
5. Nếu hợp lệ:
   → refreshRepo.MarkRevoked(jti) → revoke token cũ (rotation)
   → GenerateTokenPair(userID, role) → cấp cặp token mới
```

**Mục đích:**
- Phát hiện khi token cũ bị dùng lại (hacker có refresh token lộ)
- Buộc ngắt toàn bộ phiên của user (RevokeAllForUser)

#### `RevokeSession(accessTokenStr, refreshTokenStr string) error`

**Logic (logout):**

```
1. ParseToken(accessTokenStr) → verify access token
2. ParseToken(refreshTokenStr) → verify refresh token
3. refreshRepo.MarkRevoked(refreshClaims.ID) → revoke refresh token trong DB
4. Tính TTL còn lại của access token:
   ttl = expiresAt - now()
5. Nếu ttl > 0:
   → redis.Set("jwt:blacklist:" + accessJTI, "1", ttl)
   → Blacklist access token trong Redis theo TTL
```

**Mục đích:**
- Logout ngay lập tức, không cần đợi access token hết hạn
- Redis được dùng (thay vì DB) vì:
  - In-memory, nhanh
  - Tự expire theo TTL
  - Không cần cột mới trong DB

#### `ParseToken(tokenStr string, opts ...jwt.ParserOption) (*Claims, error)`

**Logic:**

```go
claims := &Claims{}
_, err := jwt.ParseWithClaims(tokenStr, claims, s.keyFunc(), opts...)
return claims, err
```

**Phụ thuộc:**
- `s.keyFunc()` → trả public key
- JWT library tự verify chữ ký + claims

#### `keyFunc() jwt.Keyfunc`

```go
return func(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return s.publicKey, nil
}
```

**Mục đích:**
- Callback được gọi bởi jwt.Parse
- Kiểm tra method là RS256 (chống downgrade attack)
- Trả public key để verify

---

## 6) Handler Layer (HTTP Endpoints)

### File: `internal/handler/user_handler.go`

Chức năng: Xử lý HTTP request/response

```go
type UserHandler struct {
    userSvc *service.UserService
    authSvc *service.AuthService
}
```

#### `Register(c *gin.Context)`

**Flow:**

```
1. ShouldBindJSON → parse + validate request
   - username required
   - email required, valid format
   - password required, min 6 chars
   - role required, oneof="manager member"

2. userSvc.Register(username, email, password, role)

3. Nếu error:
   - "email already in use" → 409 Conflict
   - Lỗi khác → 400 Bad Request

4. Success → 201 Created + JSON response:
   {
     "user_id": "...",
     "username": "...",
     "email": "...",
     "role": "..."
   }
```

#### `Login(c *gin.Context)`

**Flow:**

```
1. ShouldBindJSON → parse + validate
   - email required, valid format
   - password required

2. userSvc.Login(email, password)

3. Nếu error → 401 Unauthorized + {"error": "invalid credentials"}

4. authSvc.GenerateTokenPair(user.UserID, user.Role)

5. Success → 200 OK + JSON:
   {
     "access_token": "...",
     "refresh_token": "..."
   }
```

#### `Refresh(c *gin.Context)`

**Flow:**

```
1. ShouldBindJSON → parse
   - refresh_token required

2. authSvc.RotateRefreshToken(req.RefreshToken)

3. Nếu error (invalid, expired, revoked, reuse) → 401 Unauthorized

4. Success → 200 OK + cặp token mới
```

#### `Logout(c *gin.Context)`

**Flow:**

```
1. GetHeader("Authorization") → kiểm tra "Bearer <token>"
   - Nếu không có hoặc format sai → 400 Bad Request

2. ShouldBindJSON → parse
   - refresh_token required

3. authSvc.RevokeSession(accessToken, refreshToken)

4. Nếu error → 401 Unauthorized

5. Success → 204 No Content (không có body)
```

#### `JWKS(c *gin.Context)`

**Flow:**

```
1. authSvc.PublicKey() → lấy public key

2. Extract thành phần RSA:
   - n (modulus): pub.N.Bytes() → Base64URL encode
   - e (exponent): pub.E → convert uint32 → Base64URL encode

3. JSON response:
   {
     "keys": [
       {
         "kty": "RSA",
         "use": "sig",
         "alg": "RS256",
         "kid": "auth-service-key-1",
         "n": "...",
         "e": "..."
       }
     ]
   }
```

**Dùng cho:** Service khác verify JWT mà auth-service ký

---

## 7) main.go - Khởi tạo và Wiring

### `func main()`

**Flow:**

```
1. log.Logger = zerolog.ConsoleWriter
   → Setup logging

2. run()
   → Delegate to main logic
```

### `func run() error`

**Flow:**

```
1. Load config từ env:
   - DB_URL (PostgreSQL DSN)
   - REDIS_URL
   - JWT_PRIVATE_KEY_PATH (mặc định: jwt_rs256.pem)
   - JWT_PUBLIC_KEY_PATH (mặc định: jwt_rs256.pub)
   - PORT (mặc định: 8081)

2. runMigrations(dbURL)
   → Chạy SQL migration (tạo tables)

3. gorm.Open(postgres.Open(dbURL))
   → Kết nối PostgreSQL

4. loadPrivateKey() / loadPublicKey()
   → Đọc .pem files, parse RSA keys

5. connectRedis()
   → Kết nối Redis (dùng cho access token blacklist)

6. Tạo repositories:
   - userRepo := NewUserRepository(db)
   - refreshRepo := NewRefreshTokenRepository(db)

7. Tạo services:
   - userSvc := NewUserService(userRepo)
   - authSvc := NewAuthService(refreshRepo, privateKey, publicKey, rdb)

8. Tạo handler:
   - h := NewUserHandler(userSvc, authSvc)

9. Setup Gin router:
   r := gin.New()
   - GET  /health
   - GET  /.well-known/jwks.json
   - POST /register
   - POST /login
   - POST /refresh
   - POST /logout

10. r.Run(":" + port)
    → Start HTTP server
```

---

## 8) Các hàm phụ trong main.go

### `loadPrivateKey() (*rsa.PrivateKey, error)`

**Logic:**

```go
data := os.ReadFile(path)          // Đọc .pem file
block, _ := pem.Decode(data)       // Decode PEM format
key := x509.ParsePKCS8PrivateKey(block.Bytes)  // Parse PKCS8
rsaKey := key.(*rsa.PrivateKey)    // Type assert thành RSA
return rsaKey
```

**Lỗi nếu:**
- File không tồn tại
- Format không hợp lệ
- Không phải RSA key

### `loadPublicKey() (*rsa.PublicKey, error)`

**Tương tự loadPrivateKey nhưng:**
- Parse PKIX format (public key standard)
- Type assert thành `*rsa.PublicKey`

### `runMigrations(dbURL string) error`

```go
m, err := migrate.New("file://migrations", dbURL)
err := m.Up()  // Chạy tất cả migration chưa chạy
```

**Dùng `golang-migrate`:**
- Tìm .sql files trong `migrations/`
- Áp dụng theo thứ tự (000001, 000002, ...)
- Idempotent (chạy lại không error)

### `connectRedis() (*redis.Client, error)`

```go
opt := redis.ParseURL(url)
rdb := redis.NewClient(opt)
rdb.Ping(context.Background())  // Test connection
return rdb
```

**Mục đích:** Connect Redis cho access token blacklist

---

## 9) Dependency Injection Flow

```
main()
  ↓
db ← PostgreSQL
redisClient ← Redis
privateKey, publicKey ← File .pem
  ↓
UserRepository(db)
RefreshTokenRepository(db)
  ↓
UserService(userRepo)
AuthService(refreshRepo, privateKey, publicKey, redis)
  ↓
UserHandler(userSvc, authSvc)
  ↓
Gin routes → handler methods
```

Cách làm này giúp:
- **Testability**: Mock repositories dễ dàng
- **Separation of Concerns**: Mỗi layer có trách nhiệm riêng
- **Flexibility**: Thay đổi implementation không ảnh hưởng đến layer khác

---

## 10) Error Handling Pattern

**Ở handler:**

```go
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
}
```

**Ở service:**

```go
if user == nil {
    return nil, errors.New("invalid credentials")
}
```

**Ở repository:**

```go
return r.db.Where("...").First(&user).Error
```

**Lưu ý:**
- Không panic (trừ startup)
- Trả error message phù hợp
- HTTP status code thích hợp (400, 401, 409, 500)

---

## 11) Security Practices trong Code

1. **Password Hashing:**
   - Dùng bcrypt, không hash thô
   - Mặc định cost factor đủ cao

2. **Token Signing:**
   - RS256 (asymmetric), không HS256
   - Private key chỉ ở auth-service
   - Public key công khai

3. **Refresh Token Rotation:**
   - Mỗi refresh phải revoke cái cũ
   - Phát hiện reuse → revoke toàn phiên

4. **Access Token Blacklist:**
   - Logout blacklist token trong Redis
   - TTL = token expiry
   - Không cần DB overhead

5. **Error Messages:**
   - Login: "invalid credentials" (chung) → chống user enumeration
   - Không lộ thông tin chi tiết (DB error, etc)

6. **Validation:**
   - Ở handler: format validation (email, password length, role)
   - Ở service: business logic validation (unique email)

---

## 12) Testing gợi ý

```bash
# Run tất cả tests
go test ./...

# Run tests cho service layer
go test ./internal/service -v

# Run test cụ thể
go test ./internal/service -run TestLogin -v

# Coverage
go test ./... -cover
```

**Mock patterns:**

```go
type mockUserRepo struct{}

func (m *mockUserRepo) FindByEmail(email string) (*model.User, error) {
    // Mock logic
}

// Test
repo := &mockUserRepo{}
svc := service.NewUserService(repo)
```

---

## 13) Tóm tắt Data Flow

### Login Flow:

```
Client POST /login
  ↓
Handler.Login()
  ↓
UserService.Login() → UserRepository.FindByEmail() + bcrypt.Compare
  ↓
AuthService.GenerateTokenPair() → JWT + RefreshTokenRepository.Insert()
  ↓
Handler response: access_token + refresh_token
```

### Refresh Flow:

```
Client POST /refresh + refresh_token
  ↓
Handler.Refresh()
  ↓
AuthService.RotateRefreshToken() → ParseToken + RefreshTokenRepository.IsValid()
  ↓
Nếu invalid → RefreshTokenRepository.RevokeAllForUser()
Nếu valid → RefreshTokenRepository.MarkRevoked() + GenerateTokenPair()
  ↓
Handler response: new access_token + new refresh_token
```

### Logout Flow:

```
Client POST /logout + Authorization header + refresh_token
  ↓
Handler.Logout()
  ↓
AuthService.RevokeSession() → ParseToken + RefreshTokenRepository.MarkRevoked()
  ↓
Redis.Set(jwt:blacklist:<jti>, 1, ttl)
  ↓
Handler response: 204 No Content
```

