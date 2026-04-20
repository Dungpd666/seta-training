# Xác thực (Authentication) đang được triển khai trong dự án

Tài liệu này mô tả **đúng theo phần đã implement trong code hiện tại** của `auth-service`.

## 1) Tổng quan kiến trúc xác thực

`auth-service` chịu trách nhiệm:

- Đăng ký tài khoản (`/register`)
- Đăng nhập và cấp token (`/login`)
- Làm mới token theo cơ chế rotation (`/refresh`)
- Đăng xuất và thu hồi phiên (`/logout`)
- Cung cấp public key qua JWKS (`/.well-known/jwks.json`)

Cơ chế token dùng **JWT RS256** (khóa bất đối xứng):

- Ký token bằng private key (`jwt_rs256.pem`)
- Verify token bằng public key (`jwt_rs256.pub`)

Ngoài PostgreSQL, service dùng Redis để lưu blacklist cho access token sau khi logout.

## 2) Mô hình dữ liệu liên quan đến auth

### Bảng `users`

Thông tin chính:

- `user_id` (UUID, PK)
- `username`
- `email` (unique)
- `password_hash`
- `role` (`manager` hoặc `member`)
- `created_at`

### Bảng `refresh_tokens`

Dùng để quản lý vòng đời refresh token:

- `jti` (UUID, PK)
- `user_id` (FK sang `users`)
- `expires_at`
- `revoked` (boolean)

## 3) Luồng đăng ký / đăng nhập

### Đăng ký (`POST /register`)

- Validate input:
  - `username` bắt buộc
  - `email` đúng format
  - `password` tối thiểu 6 ký tự
  - `role` chỉ nhận `manager` hoặc `member`
- Kiểm tra email đã tồn tại chưa
- Băm mật khẩu bằng `bcrypt`
- Lưu user vào DB
- Trả về thông tin user (không trả `password_hash`)

### Đăng nhập (`POST /login`)

- Tìm user theo email
- So sánh password plaintext với `password_hash` bằng `bcrypt.CompareHashAndPassword`
- Nếu đúng, cấp cặp token:
  - `access_token`
  - `refresh_token`

## 4) Cấu trúc và thời hạn token

Service dùng custom claims gồm:

- `sub` = `user_id`
- `jti` = mã định danh token
- `iss` = `auth-service`
- `aud` = `seta`
- `exp`, `iat`
- `role` (cho access token)
- `typ=refresh` (cho refresh token)

TTL hiện tại trong code:

- `access_token`: **15 phút**
- `refresh_token`: **7 ngày**

## 5) Refresh token rotation + phát hiện reuse

Khi gọi `POST /refresh` với `refresh_token`:

1. Parse + verify chữ ký và claims (`iss`, `aud`, `exp`) 
2. Bắt buộc `typ` phải là `refresh`
3. Kiểm tra `jti` còn hợp lệ trong DB (`revoked=false`, chưa hết hạn)
4. Nếu không hợp lệ: coi như có khả năng reuse token cũ, revoke toàn bộ refresh token của user
5. Nếu hợp lệ: revoke refresh token hiện tại (`mark revoked`)
6. Cấp cặp token mới

Cách làm này giúp giảm rủi ro khi refresh token bị lộ và bị dùng lại.

## 6) Logout / thu hồi phiên

`POST /logout` yêu cầu:

- Header `Authorization: Bearer <access_token>`
- Body có `refresh_token`

Xử lý:

1. Verify access token
2. Verify refresh token
3. Revoke refresh token trong DB
4. Blacklist `jti` của access token trong Redis theo key:
   - `jwt:blacklist:<access_jti>`
   - TTL bằng thời gian còn lại của access token

## 7) JWKS endpoint

`GET /.well-known/jwks.json` trả public key theo định dạng JWKS (`kty`, `alg`, `kid`, `n`, `e`).

Mục đích: service khác có thể lấy key để verify JWT do `auth-service` ký.

## 8) Một số lưu ý đúng theo trạng thái code hiện tại

- Tại `auth-service`, các endpoint auth chưa có middleware bảo vệ vì đây là các endpoint public để đăng ký/đăng nhập/refresh/logout.
- Cơ chế blacklist access token đã được ghi vào Redis khi logout.
- Trong luồng refresh, role không được set trong refresh token claims. Vì vậy access token mới sau refresh có thể không mang `role` như kỳ vọng (đây là hành vi hiện tại của code, có thể cần chỉnh trong bước cải tiến tiếp theo).

---

Nếu bạn muốn, mình có thể viết thêm một bản thứ hai theo kiểu “dành cho người mới” (ít thuật ngữ hơn, có sơ đồ luồng request/response minh họa).

---

## 8) Luồng Request/Response chi tiết

### `POST /register` — Đăng ký tài khoản

**Request:**
```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "securepass123",
  "role": "manager"
}
```

**Response (201 Created):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "alice",
  "email": "alice@example.com",
  "role": "manager"
}
```

**Lỗi (409 Conflict):**
```json
{
  "error": "email already in use"
}
```

---

### `POST /login` — Đăng nhập

**Request:**
```json
{
  "email": "alice@example.com",
  "password": "securepass123"
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJyb2xlIjoibWFuYWdlciIsImp0aSI6IjEyMzQ1Njc4LWFiY2QtZWZnaCIsImlzcyI6ImF1dGgtc2VydmljZSIsImF1ZCI6InNldGEiLCJleHAiOjE3MTM2MDA5MDB9...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJ0eXAiOiJyZWZyZXNoIiwianRpIjoiYWJjZGVmZ2gtMTIzNDU2Nzg4IiwiaXNzIjoiYXV0aC1zZXJ2aWNlIiwiYXVkIjoic2V0YSIsImV4cCI6MTcxNDIwNTA5MH0..."
}
```

**Lỗi (401 Unauthorized):**
```json
{
  "error": "invalid credentials"
}
```

---

### `POST /refresh` — Làm mới token

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Lỗi (401 Unauthorized):**
```json
{
  "error": "invalid refresh token"
}
```

Hoặc nếu phát hiện reuse:
```json
{
  "error": "refresh token reuse detected"
}
```

---

### `POST /logout` — Đăng xuất

**Request:**
```
Headers:
  Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...

Body:
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (204 No Content):**
```
(không có body)
```

**Lỗi (401 Unauthorized):**
```json
{
  "error": "invalid access token"
}
```

---

### `GET /.well-known/jwks.json` — JWKS endpoint

**Request:**
```
GET /.well-known/jwks.json
```

**Response (200 OK):**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "auth-service-key-1",
      "n": "xGOr-H0A-_qzUaVG8GxLJ...",
      "e": "AQAB"
    }
  ]
}
```

---

## 9) Cấu trúc JWT token minh họa

**Access token decoded (ví dụ):**
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "role": "manager",
  "jti": "12345678-abcd-efgh",
  "iss": "auth-service",
  "aud": "seta",
  "exp": 1713600900,
  "iat": 1713599000
}
```

**Refresh token decoded (ví dụ):**
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "typ": "refresh",
  "jti": "abcdefgh-12345678",
  "iss": "auth-service",
  "aud": "seta",
  "exp": 1714205090,
  "iat": 1713600690
}
```
