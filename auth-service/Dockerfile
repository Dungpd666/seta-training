# build 
FROM golang:1.26-alpine AS builder 
WORKDIR /src 
COPY go.mod go.sum ./ 
RUN go mod download
COPY . .
RUN go build -o /app ./cmd 

# run 
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app /app/server
COPY --from=builder /src/migrations ./migrations
EXPOSE 8081
CMD ["/app/server"]


