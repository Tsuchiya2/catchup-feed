# Security Evaluation Report: GET /articles/{id} Endpoint

**Evaluator**: Code Security Evaluator v1 (Self-Adapting)
**Version**: 2.0
**Timestamp**: 2025-12-06
**Evaluated Files**:
- `/internal/handler/http/article/get.go`
- `/internal/infra/adapter/persistence/postgres/article_repo.go`
- `/internal/usecase/article/service.go`
- `/internal/handler/http/auth/middleware.go`
- `/internal/handler/http/respond/respond.go`
- `/internal/handler/http/respond/sanitize.go`

---

## Executive Summary

| Security Category | Score | Status |
|-------------------|-------|--------|
| **Overall Security** | **4.2/5.0** | **PASS** |
| SQL Injection Prevention | 5.0/5.0 | ✅ PASS |
| Input Validation | 4.5/5.0 | ✅ PASS |
| Authentication/Authorization | 3.5/5.0 | ⚠️ PASS WITH CONCERNS |
| Information Leakage Prevention | 4.5/5.0 | ✅ PASS |

**Result**: **PASS** (4.2/5.0 >= 4.0 threshold)

**Critical Findings**: 0
**High Findings**: 1
**Medium Findings**: 2
**Low Findings**: 1

---

## 1. SQL Injection Prevention Assessment

### Score: 5.0/5.0 ✅

**Status**: EXCELLENT - No SQL injection vulnerabilities detected

### Analysis

#### Repository Layer (`article_repo.go`)

```go
func (repo *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
    const query = `
SELECT a.id, a.source_id, a.title, a.url, a.summary, a.published_at, a.created_at, s.name AS source_name
FROM articles a
INNER JOIN sources s ON a.source_id = s.id
WHERE a.id = $1
LIMIT 1`
    var article entity.Article
    var sourceName string
    err := repo.db.QueryRowContext(ctx, query, id).
        Scan(&article.ID, &article.SourceID, &article.Title, &article.URL,
            &article.Summary, &article.PublishedAt, &article.CreatedAt, &sourceName)
    // ...
}
```

**Strengths**:
1. ✅ **Parameterized Queries**: Uses PostgreSQL placeholder `$1` for ID parameter
2. ✅ **No String Concatenation**: Query is defined as constant with no dynamic string building
3. ✅ **Type Safety**: `id` parameter is strongly typed as `int64`
4. ✅ **QueryRowContext Usage**: Properly uses context-aware database method

**Verification**:
- All SQL queries in the repository use parameterized queries (`$1`, `$2`, etc.)
- No usage of `fmt.Sprintf()` or string concatenation in queries
- Uses `database/sql` with PostgreSQL driver (`pgx/v5`), which prevents SQL injection by default

**Recommendation**: None - Implementation follows best practices.

---

## 2. Input Validation Assessment

### Score: 4.5/5.0 ✅

**Status**: GOOD - Input validation is implemented with minor improvements possible

### Analysis

#### Path Parameter Validation (`get.go`)

```go
func (h GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    id, err := pathutil.ExtractID(r.URL.Path, "/articles/")
    if err != nil {
        respond.SafeError(w, http.StatusBadRequest, err)
        return
    }
    // ...
}
```

#### ID Extraction Implementation (`pathutil/id.go`)

```go
func ExtractID(path, prefix string) (int64, error) {
    idStr := strings.TrimPrefix(path, prefix)
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil || id <= 0 {
        return 0, ErrInvalidID
    }
    return id, nil
}
```

**Strengths**:
1. ✅ **Integer Parsing Validation**: Validates that ID is a valid integer
2. ✅ **Positive Number Check**: Rejects negative and zero IDs (`id <= 0`)
3. ✅ **Type Safety**: Uses `int64` to prevent integer overflow
4. ✅ **Centralized Validation**: Validation logic is centralized in `pathutil` package

#### Use Case Layer Validation (`service.go`)

```go
func (s *Service) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
    if id <= 0 {
        return nil, "", ErrInvalidArticleID
    }
    // ...
}
```

**Strengths**:
1. ✅ **Defense in Depth**: Validates ID again at use case layer
2. ✅ **Clear Error Messages**: Returns `ErrInvalidArticleID` sentinel error

### Issues Found

#### Medium: Missing Maximum ID Validation

**Finding**:
```go
// Current implementation
id, err := strconv.ParseInt(idStr, 10, 64)
if err != nil || id <= 0 {
    return 0, ErrInvalidID
}

// No upper bound check
// Risk: Very large IDs (e.g., 9223372036854775807) could cause:
// - Database performance issues
// - Potential DoS by forcing full table scans
```

**Impact**: Medium
**Likelihood**: Low
**Severity**: Medium

**Recommendation**:
```go
// Add maximum ID validation
const MaxArticleID = 2147483647 // 2^31 - 1 (reasonable upper limit)

if err != nil || id <= 0 || id > MaxArticleID {
    return 0, ErrInvalidID
}
```

**Rationale**: While PostgreSQL can handle large IDs, adding an upper bound:
- Prevents potential abuse with extremely large IDs
- Adds defense-in-depth
- Documents expected ID range

---

## 3. Authentication/Authorization Assessment

### Score: 3.5/5.0 ⚠️

**Status**: PASS WITH CONCERNS - Authentication is implemented but has architectural inconsistencies

### Analysis

#### Route Registration (`register.go`)

```go
func Register(mux *http.ServeMux, svc artUC.Service) {
    mux.Handle("GET    /articles", ListHandler{svc})
    mux.Handle("GET    /articles/search", SearchHandler{svc})
    mux.Handle("GET    /articles/", auth.Authz(GetHandler{svc}))  // ⚠️ Auth required

    mux.Handle("POST   /articles", auth.Authz(CreateHandler{svc}))
    mux.Handle("PUT    /articles/", auth.Authz(UpdateHandler{svc}))
    mux.Handle("DELETE /articles/", auth.Authz(DeleteHandler{svc}))
}
```

#### Authentication Middleware (`middleware.go`)

```go
func Authz(next http.Handler) http.Handler {
    secret := []byte(os.Getenv("JWT_SECRET"))
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Step 1: Check if endpoint is public
        if IsPublicEndpoint(r.URL.Path) {
            next.ServeHTTP(w, r)
            return
        }

        // Step 2: Protected endpoint - require JWT for ALL methods
        user, role, err := validateJWT(r.Header.Get("Authorization"), secret)
        if err != nil {
            respond.SafeError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized: %w", err))
            return
        }

        // Step 3: Check role-based permissions
        hasPermission := checkRolePermission(role, r.Method, r.URL.Path)
        if !hasPermission {
            respond.SafeError(w, http.StatusForbidden, fmt.Errorf("forbidden: %s role cannot perform %s operations", role, r.Method))
            return
        }

        ctx := context.WithValue(r.Context(), ctxUser, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Strengths**:
1. ✅ **JWT Authentication**: Uses industry-standard JWT tokens
2. ✅ **Bearer Token Validation**: Properly validates `Bearer` prefix
3. ✅ **Algorithm Validation**: Enforces HS256 signing method
4. ✅ **Token Expiration Check**: Validates `exp` claim
5. ✅ **Role-Based Access Control (RBAC)**: Implements role-based permissions
6. ✅ **Secure JWT Secret**: Enforces minimum 32-character secret
7. ✅ **Context Propagation**: Adds user to request context for downstream handlers

### Issues Found

#### High: Inconsistent Authentication Requirements Across Similar Endpoints

**Finding**:
```go
// Article endpoints
mux.Handle("GET    /articles", ListHandler{svc})              // ❌ NO AUTH
mux.Handle("GET    /articles/search", SearchHandler{svc})     // ❌ NO AUTH
mux.Handle("GET    /articles/", auth.Authz(GetHandler{svc})) // ✅ AUTH REQUIRED

// Inconsistency:
// - List all articles: No authentication required
// - Search articles: No authentication required
// - Get single article: Authentication required ⚠️
```

**Impact**: High
**Likelihood**: High
**Severity**: High

**Security Implications**:
1. **Broken Access Control (OWASP A01:2021)**:
   - Unauthenticated users can list ALL articles via `/articles`
   - Unauthenticated users can search ALL articles via `/articles/search`
   - But require authentication to view a SINGLE article via `/articles/{id}`

2. **Information Disclosure**:
   - Sensitive article metadata (titles, URLs, summaries) exposed without authentication
   - Article IDs exposed, making enumeration trivial
   - Source names and relationships exposed

3. **Inconsistent Security Model**:
   - Creates confusion about which data is public vs protected
   - Violates principle of least privilege
   - Documentation shows `@Security BearerAuth` but other endpoints don't require it

**Recommendation**:

**Option 1: Protect All Endpoints (Recommended)**
```go
func Register(mux *http.ServeMux, svc artUC.Service) {
    // Require authentication for ALL article endpoints
    mux.Handle("GET    /articles", auth.Authz(ListHandler{svc}))
    mux.Handle("GET    /articles/search", auth.Authz(SearchHandler{svc}))
    mux.Handle("GET    /articles/", auth.Authz(GetHandler{svc}))

    mux.Handle("POST   /articles", auth.Authz(CreateHandler{svc}))
    mux.Handle("PUT    /articles/", auth.Authz(UpdateHandler{svc}))
    mux.Handle("DELETE /articles/", auth.Authz(DeleteHandler{svc}))
}
```

**Option 2: Make All Endpoints Public (If Intended)**
```go
func Register(mux *http.ServeMux, svc artUC.Service) {
    // All endpoints are public (requires business justification)
    mux.Handle("GET    /articles", ListHandler{svc})
    mux.Handle("GET    /articles/search", SearchHandler{svc})
    mux.Handle("GET    /articles/", GetHandler{svc}) // Remove auth.Authz

    // Only write operations require auth
    mux.Handle("POST   /articles", auth.Authz(CreateHandler{svc}))
    mux.Handle("PUT    /articles/", auth.Authz(UpdateHandler{svc}))
    mux.Handle("DELETE /articles/", auth.Authz(DeleteHandler{svc}))
}
```

**Recommendation**: **Option 1** is strongly recommended for a secure RSS feed system unless there's a specific business requirement for public read access.

#### Medium: Handler-Level Authorization Not Implemented

**Finding**:
```go
func (h GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // No authorization check in handler itself
    // Relies entirely on middleware

    id, err := pathutil.ExtractID(r.URL.Path, "/articles/")
    if err != nil {
        respond.SafeError(w, http.StatusBadRequest, err)
        return
    }

    // No check if user has permission to view this specific article
    article, sourceName, err := h.Svc.GetWithSource(r.Context(), id)
    // ...
}
```

**Impact**: Medium
**Likelihood**: Low (currently mitigated by route-level middleware)
**Severity**: Medium

**Risk**:
- If route is registered without middleware by mistake, handler has no defense
- No resource-level authorization (e.g., user can only view their own articles)

**Recommendation**:
```go
func (h GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Defense in depth: Validate authentication even if middleware is bypassed
    user := auth.GetUserFromContext(r.Context())
    if user == "" {
        respond.SafeError(w, http.StatusUnauthorized, errors.New("authentication required"))
        return
    }

    id, err := pathutil.ExtractID(r.URL.Path, "/articles/")
    if err != nil {
        respond.SafeError(w, http.StatusBadRequest, err)
        return
    }

    article, sourceName, err := h.Svc.GetWithSource(r.Context(), id)
    if err != nil {
        // ...
    }

    // Optional: Resource-level authorization
    // if !canUserAccessArticle(user, article) {
    //     respond.SafeError(w, http.StatusForbidden, errors.New("access denied"))
    //     return
    // }

    // ...
}
```

---

## 4. Information Leakage Prevention Assessment

### Score: 4.5/5.0 ✅

**Status**: GOOD - Error handling properly prevents information leakage

### Analysis

#### Error Sanitization (`respond/sanitize.go`)

```go
var (
    anthropicKeyPattern = regexp.MustCompile(`sk-ant-[a-zA-Z0-9-_]+`)
    openaiKeyPattern = regexp.MustCompile(`sk-[a-zA-Z0-9]{10,}`)
    dbPasswordPattern = regexp.MustCompile(`://([^:]+):([^@]+)@`)
)

func SanitizeError(err error) string {
    if err == nil {
        return ""
    }

    msg := err.Error()

    // APIキーのマスク
    msg = anthropicKeyPattern.ReplaceAllString(msg, "sk-ant-****")
    msg = openaiKeyPattern.ReplaceAllString(msg, "sk-****")

    // DBパスワードのマスク
    msg = dbPasswordPattern.ReplaceAllString(msg, "://$1:****@")

    return msg
}
```

**Strengths**:
1. ✅ **API Key Masking**: Masks Anthropic and OpenAI API keys
2. ✅ **Database Password Masking**: Masks passwords in DSN strings
3. ✅ **Regex-Based Detection**: Uses patterns to detect sensitive data

#### Safe Error Response (`respond/respond.go`)

```go
func SafeError(w http.ResponseWriter, code int, err error) {
    if err == nil {
        return
    }

    msg := err.Error()

    // Safe errors can be returned to user
    safeErrors := []string{
        "required", "invalid", "not found", "already exists",
        "must be", "cannot be", "too long", "too short",
    }

    isSafe := false
    lowerMsg := strings.ToLower(msg)
    for _, safe := range safeErrors {
        if strings.Contains(lowerMsg, safe) {
            isSafe = true
            break
        }
    }

    // 500 errors are always internal
    if code >= 500 {
        isSafe = false
    }

    if isSafe {
        JSON(w, code, map[string]string{"error": msg})
    } else {
        // Log detailed error, return generic message
        logger.Error("internal server error",
            slog.String("status", http.StatusText(code)),
            slog.Int("code", code),
            slog.Any("error", SanitizeError(err)))
        JSON(w, code, map[string]string{"error": "internal server error"})
    }
}
```

**Strengths**:
1. ✅ **Whitelisting Approach**: Only allows known-safe error messages
2. ✅ **Generic Error Messages**: Returns "internal server error" for unsafe errors
3. ✅ **Structured Logging**: Logs detailed errors for debugging
4. ✅ **Automatic 5xx Handling**: Always hides 500-level errors

#### Handler Error Handling (`get.go`)

```go
func (h GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    id, err := pathutil.ExtractID(r.URL.Path, "/articles/")
    if err != nil {
        respond.SafeError(w, http.StatusBadRequest, err)
        return
    }

    article, sourceName, err := h.Svc.GetWithSource(r.Context(), id)
    if err != nil {
        code := http.StatusInternalServerError
        if errors.Is(err, artUC.ErrInvalidArticleID) {
            code = http.StatusBadRequest
        } else if errors.Is(err, artUC.ErrArticleNotFound) {
            code = http.StatusNotFound
        }
        respond.SafeError(w, code, err)
        return
    }

    // ...
}
```

**Strengths**:
1. ✅ **Sentinel Error Matching**: Uses `errors.Is()` for type-safe error handling
2. ✅ **Appropriate HTTP Status Codes**:
   - 400 for invalid input
   - 404 for not found
   - 500 for internal errors
3. ✅ **SafeError Usage**: All errors go through `SafeError()` sanitization

### Issues Found

#### Low: Potential Database Schema Leakage in Error Messages

**Finding**:
```go
// In article_repo.go
func (repo *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
    const query = `
SELECT a.id, a.source_id, a.title, a.url, a.summary, a.published_at, a.created_at, s.name AS source_name
FROM articles a
INNER JOIN sources s ON a.source_id = s.id
WHERE a.id = $1
LIMIT 1`
    var article entity.Article
    var sourceName string
    err := repo.db.QueryRowContext(ctx, query, id).Scan(...)
    if err == sql.ErrNoRows {
        return nil, "", nil  // ✅ Properly handled
    }
    if err != nil {
        return nil, "", fmt.Errorf("GetWithSource: %w", err)  // ⚠️ May leak schema info
    }
    return &article, sourceName, nil
}
```

**Impact**: Low
**Likelihood**: Low
**Severity**: Low

**Risk**:
- Database errors (e.g., column type mismatch, schema changes) might leak table/column names
- Currently mitigated by `SafeError()` which hides 5xx errors
- Defense-in-depth: Should wrap repository errors more carefully

**Recommendation**:
```go
if err != nil {
    // Don't wrap error directly - return generic repository error
    logger := slog.Default()
    logger.Error("database query failed",
        slog.String("operation", "GetWithSource"),
        slog.Int64("article_id", id),
        slog.Any("error", err))
    return nil, "", errors.New("failed to retrieve article")
}
```

---

## 5. Additional Security Considerations

### Context Timeout Implementation

**Finding**: Database queries don't enforce timeouts

```go
func (repo *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
    // Uses context but no explicit timeout
    err := repo.db.QueryRowContext(ctx, query, id).Scan(...)
}
```

**Recommendation**:
```go
func (s *Service) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
    if id <= 0 {
        return nil, "", ErrInvalidArticleID
    }

    // Add timeout to prevent long-running queries
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    article, sourceName, err := s.Repo.GetWithSource(ctx, id)
    // ...
}
```

### Rate Limiting

**Finding**: No rate limiting on GET endpoints

**Current State**:
```go
// In main.go
authRateLimiter := middleware.NewRateLimiter(5, 1*time.Minute, ipExtractor)
publicMux.Handle("/auth/token", authRateLimiter.Middleware(hauth.TokenHandler(authService)))

// But no rate limiting on article endpoints
privateMux := http.NewServeMux()
hsrc.Register(privateMux, srcSvc)
harticle.Register(privateMux, artSvc)  // No rate limiter
```

**Recommendation**: Add rate limiting to prevent abuse
```go
// Create read endpoint rate limiter (more permissive than auth)
readRateLimiter := middleware.NewRateLimiter(100, 1*time.Minute, ipExtractor)

// Apply to article endpoints
mux.Handle("GET /articles", readRateLimiter.Middleware(ListHandler{svc}))
mux.Handle("GET /articles/search", readRateLimiter.Middleware(SearchHandler{svc}))
mux.Handle("GET /articles/", readRateLimiter.Middleware(auth.Authz(GetHandler{svc})))
```

---

## 6. Compliance with OWASP Top 10 (2021)

| OWASP Category | Status | Notes |
|---------------|--------|-------|
| A01:2021 - Broken Access Control | ⚠️ CONCERN | Inconsistent auth between list/search/get endpoints |
| A02:2021 - Cryptographic Failures | ✅ PASS | JWT secret properly validated (32+ chars) |
| A03:2021 - Injection | ✅ PASS | Parameterized queries prevent SQL injection |
| A04:2021 - Insecure Design | ⚠️ CONCERN | Auth inconsistency suggests design gap |
| A05:2021 - Security Misconfiguration | ✅ PASS | Proper error handling, sanitization |
| A06:2021 - Vulnerable Components | N/A | Not evaluated (dependency scan needed) |
| A07:2021 - Auth Failures | ⚠️ CONCERN | Some endpoints bypass authentication |
| A08:2021 - Data Integrity Failures | ✅ PASS | JWT signature validation enforced |
| A09:2021 - Logging Failures | ✅ PASS | Proper logging with sanitization |
| A10:2021 - SSRF | N/A | Not applicable to this endpoint |

---

## 7. Summary of Findings

### Critical Findings (0)
None

### High Findings (1)

| ID | Severity | Title | Impact | Recommendation Priority |
|----|----------|-------|--------|------------------------|
| SEC-001 | High | Inconsistent Authentication Requirements | Unauthenticated users can list/search all articles but not view single article | **CRITICAL** - Fix immediately |

### Medium Findings (2)

| ID | Severity | Title | Impact | Recommendation Priority |
|----|----------|-------|--------|------------------------|
| SEC-002 | Medium | Missing Maximum ID Validation | Potential DoS with extremely large IDs | **HIGH** - Add upper bound |
| SEC-003 | Medium | Handler-Level Authorization Not Implemented | Lack of defense-in-depth | **MEDIUM** - Consider for future |

### Low Findings (1)

| ID | Severity | Title | Impact | Recommendation Priority |
|----|----------|-------|--------|------------------------|
| SEC-004 | Low | Potential Database Schema Leakage | Minor information disclosure risk | **LOW** - Consider for hardening |

---

## 8. Detailed Scoring Breakdown

### SQL Injection Prevention: 5.0/5.0
- ✅ Parameterized queries throughout
- ✅ No string concatenation in SQL
- ✅ Type-safe parameter binding
- ✅ PostgreSQL driver with built-in protection

**Deductions**: None

### Input Validation: 4.5/5.0
- ✅ Integer parsing validation
- ✅ Positive number check
- ✅ Defense in depth (handler + use case)
- ⚠️ Missing maximum ID validation (-0.5)

**Deductions**: -0.5 for missing upper bound

### Authentication/Authorization: 3.5/5.0
- ✅ JWT authentication implemented
- ✅ Role-based access control
- ✅ Secure JWT secret validation
- ⚠️ Inconsistent auth across similar endpoints (-1.0)
- ⚠️ No handler-level auth checks (-0.5)

**Deductions**: -1.5 total

### Information Leakage Prevention: 4.5/5.0
- ✅ Error message sanitization
- ✅ API key/password masking
- ✅ Safe error handling
- ✅ Generic error messages for internal errors
- ⚠️ Potential schema leakage (low risk) (-0.5)

**Deductions**: -0.5 for minor leakage risk

### Overall Score Calculation
```
Overall = (
  SQL_Injection * 0.30 +      # 5.0 * 0.30 = 1.50
  Input_Validation * 0.20 +   # 4.5 * 0.20 = 0.90
  Auth_Authz * 0.35 +         # 3.5 * 0.35 = 1.225
  Info_Leakage * 0.15         # 4.5 * 0.15 = 0.675
) = 4.2/5.0
```

**Weighted Rationale**:
- Auth/Authz (35%): Most critical for endpoint security
- SQL Injection (30%): High impact if present
- Input Validation (20%): Important for data integrity
- Info Leakage (15%): Important but lower immediate risk

---

## 9. Recommendations

### Immediate Actions (Critical Priority)

1. **Fix Authentication Inconsistency (SEC-001)**
   - Decision required: Should `/articles` and `/articles/search` require authentication?
   - If YES: Add `auth.Authz()` middleware to both endpoints
   - If NO: Remove `auth.Authz()` from `/articles/{id}` endpoint
   - Update Swagger documentation to reflect actual auth requirements
   - **Estimated Effort**: 30 minutes

2. **Add Input Validation Upper Bound (SEC-002)**
   - Add maximum ID constant (e.g., `2^31 - 1`)
   - Validate in `pathutil.ExtractID()`
   - **Estimated Effort**: 15 minutes

### Short-Term Actions (High Priority)

3. **Add Rate Limiting to Read Endpoints**
   - Prevent enumeration attacks
   - Protect against DoS
   - **Estimated Effort**: 1 hour

4. **Implement Query Timeouts**
   - Add context timeouts to database operations
   - Prevent resource exhaustion
   - **Estimated Effort**: 30 minutes

### Long-Term Actions (Medium Priority)

5. **Handler-Level Authorization (SEC-003)**
   - Add defense-in-depth auth checks in handlers
   - Consider resource-level authorization
   - **Estimated Effort**: 2 hours

6. **Improve Error Wrapping (SEC-004)**
   - Wrap repository errors more carefully
   - Prevent any potential schema leakage
   - **Estimated Effort**: 1 hour

---

## 10. Conclusion

The GET /articles/{id} endpoint demonstrates **good security practices** overall:
- SQL injection prevention is excellent
- Input validation is solid with minor improvements needed
- Error handling properly prevents information leakage

However, **one critical architectural issue** must be addressed:
- **Inconsistent authentication requirements** across similar endpoints create a broken access control vulnerability

**Overall Assessment**: **PASS** (4.2/5.0)

**Recommended Action**: Fix authentication inconsistency before production deployment.

---

## 11. Testing Recommendations

### Security Test Cases to Add

1. **SQL Injection Tests**
   ```go
   // Test: SQL injection attempts
   GET /articles/1' OR '1'='1
   GET /articles/1; DROP TABLE articles--

   // Expected: 400 Bad Request (invalid ID format)
   ```

2. **Authentication Tests**
   ```go
   // Test: Access without token
   GET /articles/1
   // Expected: 401 Unauthorized

   // Test: Access with expired token
   GET /articles/1
   Authorization: Bearer <expired_token>
   // Expected: 401 Unauthorized

   // Test: Access with invalid signature
   GET /articles/1
   Authorization: Bearer <tampered_token>
   // Expected: 401 Unauthorized
   ```

3. **Input Validation Tests**
   ```go
   // Test: Negative ID
   GET /articles/-1
   // Expected: 400 Bad Request

   // Test: Zero ID
   GET /articles/0
   // Expected: 400 Bad Request

   // Test: Non-integer ID
   GET /articles/abc
   // Expected: 400 Bad Request

   // Test: Very large ID
   GET /articles/999999999999999
   // Expected: 400 Bad Request (after adding upper bound)
   ```

4. **Authorization Tests**
   ```go
   // Test: Viewer role access
   GET /articles/1
   Authorization: Bearer <viewer_token>
   // Expected: 200 OK (if RBAC allows)

   // Test: Admin role access
   GET /articles/1
   Authorization: Bearer <admin_token>
   // Expected: 200 OK
   ```

5. **Error Handling Tests**
   ```go
   // Test: Non-existent article
   GET /articles/999999
   Authorization: Bearer <valid_token>
   // Expected: 404 Not Found

   // Test: Database error simulation (via mock)
   // Expected: 500 Internal Server Error with generic message
   ```

---

**Evaluation Complete**

**Next Steps**:
1. Address SEC-001 (authentication inconsistency) immediately
2. Add input validation upper bound (SEC-002)
3. Implement recommended security tests
4. Re-evaluate after fixes are applied

**Evaluator**: Claude Code Security Evaluator v1
**Contact**: Generated via EDAF Framework
