# Design Reliability Evaluation - Article Detail Endpoint

**Evaluator**: design-reliability-evaluator
**Design Document**: /Users/yujitsuchiya/catchup-feed/docs/designs/article-detail-endpoint.md
**Evaluated**: 2025-12-06T00:00:00Z

---

## Overall Judgment

**Status**: Request Changes
**Overall Score**: 3.4 / 5.0

---

## Detailed Scores

### 1. Error Handling Strategy: 3.5 / 5.0 (Weight: 35%)

**Findings**:
The design document provides a comprehensive list of error scenarios and appropriate HTTP status codes. The error handling strategy leverages the existing `respond.SafeError` utility to sanitize error messages, preventing information disclosure. However, there are several gaps in the error handling specification:

**Failure Scenarios Checked**:
- Database unavailable: **Partially Handled** - 500 error mentioned, but no retry/fallback strategy
- S3 upload fails: **Not Applicable** - This endpoint is read-only
- Validation errors: **Handled** - Invalid ID format returns 400
- Network timeouts: **Not Handled** - No timeout configuration mentioned
- Article exists but source deleted: **Partially Handled** - Mentioned as "JOIN fails (no result)" but unclear if returns 404 or 500

**Issues**:

1. **Missing Error Type Definitions**: The design references `ErrArticleNotFound` and `ErrInvalidArticleID` but these errors don't exist in the codebase yet. The pathutil package only defines `ErrInvalidID`, not article-specific errors.

2. **Database Error Handling Incomplete**: While the design mentions "Database Error → 500 Internal Server Error", it doesn't specify:
   - How to distinguish between connection failures vs query execution errors
   - Whether errors should be logged before being sanitized
   - Whether database connection pool exhaustion is handled

3. **NULL Value Handling Unclear**: Section 9 mentions "Article with NULL summary → Return empty string" but doesn't specify:
   - How NULL values are detected (sql.NullString?)
   - Whether zero values are validated
   - What happens if critical fields (title, source_id) are NULL (database constraints should prevent this, but design should confirm)

4. **Source Deleted During Request**: The design acknowledges "Article exists, source deleted → JOIN fails (no result)" as an edge case, but the error handling is ambiguous. Should this return:
   - 404 Not Found (article effectively doesn't exist without source)?
   - 500 Internal Server Error (data integrity issue)?
   - The design needs to clarify this scenario.

5. **No Structured Error Logging**: While the design mentions "Error details logged via `slog`", there's no specification of:
   - What context should be logged (user ID, article ID, request ID, trace ID)
   - Log level (Error vs Warn for different scenarios)
   - Log format (structured fields)

**Recommendation**:

1. **Define Article-Specific Error Types**:
   ```go
   // In internal/usecase/article/errors.go
   var (
       ErrArticleNotFound   = errors.New("article not found")
       ErrInvalidArticleID  = errors.New("invalid article id")
       ErrSourceNotFound    = errors.New("source not found for article") // For orphaned articles
   )
   ```

2. **Enhance Repository Error Handling**:
   ```go
   func (r *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
       var article entity.Article
       var sourceName string

       err := r.db.QueryRowContext(ctx, query, id).Scan(...)
       if err != nil {
           if errors.Is(err, sql.ErrNoRows) {
               return nil, "", ErrArticleNotFound
           }
           // Log database error with context before returning
           slog.ErrorContext(ctx, "database query failed",
               slog.Int64("article_id", id),
               slog.Any("error", err))
           return nil, "", fmt.Errorf("failed to get article: %w", err)
       }
       return &article, sourceName, nil
   }
   ```

3. **Add Structured Logging to Handler**:
   ```go
   func (h GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
       id, err := pathutil.ExtractID(r.URL.Path, "/articles/")
       if err != nil {
           slog.WarnContext(r.Context(), "invalid article id in request",
               slog.String("path", r.URL.Path),
               slog.Any("error", err))
           respond.SafeError(w, http.StatusBadRequest, err)
           return
       }

       article, sourceName, err := h.Svc.GetWithSource(r.Context(), id)
       if err != nil {
           if errors.Is(err, article.ErrArticleNotFound) {
               respond.SafeError(w, http.StatusNotFound, err)
               return
           }
           respond.SafeError(w, http.StatusInternalServerError, err)
           return
       }
       // ... success path
   }
   ```

4. **Clarify Edge Case Handling**: Document that if an article exists but its source is deleted (orphaned article), the JOIN will fail and return 404 Not Found, treating it as if the article doesn't exist (since it's in an invalid state).

5. **Add NULL Value Handling Specification**:
   ```go
   // Use sql.NullString for nullable fields
   var summary sql.NullString
   err := row.Scan(&article.ID, &article.SourceID, &article.Title, &summary, ...)
   if summary.Valid {
       article.Summary = summary.String
   } else {
       article.Summary = "" // Explicit empty string for NULL
   }
   ```

**Reliability Benefit**:
- Clear error types enable proper error handling at each layer
- Structured logging enables debugging and monitoring
- Consistent NULL handling prevents unexpected behavior
- Edge case clarification prevents ambiguous error responses

---

### 2. Fault Tolerance: 2.5 / 5.0 (Weight: 30%)

**Findings**:
The design lacks comprehensive fault tolerance mechanisms. While it follows clean architecture patterns (separation of concerns), it doesn't address how the system should behave when dependencies fail or become unavailable.

**Fallback Mechanisms**:
- **None specified** - No fallback strategy for database failures
- No caching layer to serve stale data if database is unavailable
- No circuit breaker to prevent cascading failures
- No graceful degradation strategy

**Retry Policies**:
- **None specified** - No retry logic for transient database failures
- No exponential backoff configuration
- No idempotency guarantees (though GET is naturally idempotent)
- Section 7 mentions "Retry with exponential backoff" as a client-side recovery strategy, but no server-side retry logic

**Circuit Breakers**:
- **None specified** - No circuit breaker pattern implementation
- No health check endpoint to verify database connectivity
- No fail-fast mechanism to prevent resource exhaustion during database outages

**Issues**:

1. **No Database Connection Pool Configuration**: The design doesn't specify:
   - Maximum connection pool size
   - Connection timeout
   - Query timeout
   - Idle connection timeout
   - Connection retry behavior

2. **No Timeout Configuration**: The design mentions "Database connection failure, query timeout" as an error scenario but doesn't specify:
   - Context timeout for database queries
   - HTTP request timeout
   - Whether timeouts are configurable

3. **Single Point of Failure**: The database is a single point of failure. If PostgreSQL is unavailable:
   - All requests fail with 500 errors
   - No fallback mechanism to serve cached data
   - No circuit breaker to fail fast

4. **No Health Check Mechanism**: The design doesn't include:
   - Database health check endpoint
   - Periodic connection validation
   - Readiness probe for Kubernetes/orchestration

5. **No Rate Limiting**: While Section 12 mentions rate limiting as a "Future Enhancement", the design lacks:
   - Per-user rate limiting to prevent abuse
   - Database query rate limiting
   - Connection pool throttling

6. **No Graceful Degradation**: If the database is slow or overloaded:
   - No queue mechanism to handle backpressure
   - No load shedding strategy
   - No priority-based request handling

**Recommendation**:

1. **Add Database Connection Pool Configuration**:
   ```go
   // In database initialization
   db.SetMaxOpenConns(25)          // Maximum open connections
   db.SetMaxIdleConns(5)           // Maximum idle connections
   db.SetConnMaxLifetime(5 * time.Minute)  // Connection lifetime
   db.SetConnMaxIdleTime(2 * time.Minute)  // Idle connection timeout
   ```

2. **Add Context Timeout to Database Queries**:
   ```go
   func (r *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
       // Set query timeout
       ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
       defer cancel()

       err := r.db.QueryRowContext(ctx, query, id).Scan(...)
       if err != nil {
           if errors.Is(err, context.DeadlineExceeded) {
               slog.ErrorContext(ctx, "database query timeout",
                   slog.Int64("article_id", id))
               return nil, "", fmt.Errorf("database query timeout")
           }
           // ... other error handling
       }
       return &article, sourceName, nil
   }
   ```

3. **Consider Adding Response Caching** (if appropriate):
   ```go
   // Optional: Add in-memory cache for frequently accessed articles
   type CachedArticleRepo struct {
       repo  ArticleRepository
       cache *sync.Map // or use redis/memcached
   }

   func (r *CachedArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
       // Try cache first
       if cached, ok := r.cache.Load(id); ok {
           if entry, ok := cached.(CacheEntry); ok && !entry.IsExpired() {
               return entry.Article, entry.SourceName, nil
           }
       }

       // Cache miss - fetch from database
       article, sourceName, err := r.repo.GetWithSource(ctx, id)
       if err == nil {
           r.cache.Store(id, CacheEntry{
               Article:    article,
               SourceName: sourceName,
               ExpiresAt:  time.Now().Add(5 * time.Minute),
           })
       }
       return article, sourceName, err
   }
   ```

4. **Add Health Check Endpoint** (separate feature):
   ```go
   // GET /health/db
   func (h HealthHandler) CheckDatabase(w http.ResponseWriter, r *http.Request) {
       ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
       defer cancel()

       if err := h.db.PingContext(ctx); err != nil {
           respond.JSON(w, http.StatusServiceUnavailable, map[string]string{
               "status": "unhealthy",
               "error":  "database unreachable",
           })
           return
       }
       respond.JSON(w, http.StatusOK, map[string]string{"status": "healthy"})
   }
   ```

5. **Document Retry Strategy for Clients**:
   - Clients should implement exponential backoff for 500 errors
   - Maximum 3 retries with 1s, 2s, 4s delays
   - Use jitter to prevent thundering herd

6. **Consider Circuit Breaker Pattern** (future enhancement):
   - Implement using library like `github.com/sony/gobreaker`
   - Open circuit after 5 consecutive failures
   - Half-open state after 30 seconds
   - Close circuit after 2 successful requests

**Reliability Benefit**:
- Timeouts prevent resource exhaustion during database outages
- Connection pool limits prevent overloading the database
- Health checks enable proactive monitoring
- Caching reduces database load and provides fallback during failures
- Circuit breakers prevent cascading failures

---

### 3. Transaction Management: 4.0 / 5.0 (Weight: 20%)

**Findings**:
This endpoint is a read-only operation (GET), so transaction management concerns are minimal. The design uses a single SQL query with JOIN, which is atomic and doesn't require explicit transaction management. However, there are still some considerations for data consistency.

**Multi-Step Operations**:
- Single database query (INNER JOIN) - **Atomicity Guaranteed** by database
- No write operations - **No rollback needed**
- No distributed transactions - **No coordination needed**

**Rollback Strategy**:
- **Not Applicable** - Read-only operation doesn't require rollback
- Query is atomic (single SELECT with JOIN)

**Issues**:

1. **Dirty Read Concern**: While the query is atomic, if an article or source is being updated concurrently:
   - PostgreSQL's default isolation level (Read Committed) may return uncommitted data
   - The design doesn't specify isolation level requirements
   - For most use cases, Read Committed is sufficient, but design should document this

2. **Orphaned Article Handling**: If an article's source is deleted concurrently:
   - INNER JOIN will exclude the article from results
   - Returns 404 (article not found) even though article exists
   - This is acceptable behavior, but should be explicitly documented

3. **No Discussion of Read Consistency**: The design doesn't address:
   - Whether stale reads are acceptable
   - Whether read replicas could be used (if available)
   - Whether eventual consistency is acceptable

4. **NULL Handling in JOIN**: If `source_id` is NULL (should be prevented by database constraints):
   - INNER JOIN will exclude the article
   - Design should confirm that `source_id` has NOT NULL constraint

**Recommendation**:

1. **Document Isolation Level**:
   ```markdown
   ### Read Isolation

   - Uses PostgreSQL default isolation level: READ COMMITTED
   - Queries see a consistent snapshot of committed data
   - Concurrent updates to articles/sources are handled gracefully:
     - If article is deleted during query → 404 Not Found
     - If source is deleted during query → 404 Not Found (INNER JOIN excludes orphaned articles)
     - If article/source is updated during query → Returns committed version
   ```

2. **Confirm Database Constraints**:
   ```sql
   -- Verify in migration
   ALTER TABLE articles
   ADD CONSTRAINT fk_articles_source_id
   FOREIGN KEY (source_id)
   REFERENCES sources(id)
   ON DELETE CASCADE; -- Or ON DELETE SET NULL, depending on requirements

   -- Ensure source_id is NOT NULL
   ALTER TABLE articles ALTER COLUMN source_id SET NOT NULL;
   ```

3. **Consider Read Replica Support** (future enhancement):
   ```go
   // If read replicas are available, use them for GET requests
   func (r *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
       // Use read replica for query (reduces load on primary)
       err := r.replicaDB.QueryRowContext(ctx, query, id).Scan(...)
       // ...
   }
   ```

4. **Document Concurrent Update Behavior**:
   ```markdown
   ### Concurrent Modification Handling

   This endpoint is read-only and doesn't modify data, so race conditions are limited:

   1. **Article Deleted During Query**: Returns 404 Not Found
   2. **Source Deleted During Query**: INNER JOIN excludes article → 404 Not Found
   3. **Article Updated During Query**: Returns committed version (before or after update)
   4. **Source Updated During Query**: Returns committed version (before or after update)

   All scenarios are handled gracefully without data corruption.
   ```

**Reliability Benefit**:
- Clear documentation of isolation behavior
- Database constraints prevent orphaned articles
- Graceful handling of concurrent modifications
- No risk of data corruption (read-only operation)

---

### 4. Logging & Observability: 3.5 / 5.0 (Weight: 15%)

**Findings**:
The design mentions using `slog` for error logging and the `respond.SafeError` utility already logs internal errors. However, the logging strategy is incomplete and lacks comprehensive observability.

**Logging Strategy**:
- **Partially Specified** - Section 7 mentions "Error details logged via `slog`"
- Existing `respond.SafeError` implementation logs errors with status code and error details
- No specification for request tracing, performance metrics, or access logs

**Structured Logging**:
- **Yes** - Uses `slog` (Go's structured logging library)
- Existing implementation in `respond.go` uses structured fields:
  ```go
  slog.Default().Error("internal server error",
      slog.String("status", http.StatusText(code)),
      slog.Int("code", code),
      slog.Any("error", SanitizeError(err)))
  ```

**Log Context**:
- **Limited** - Current implementation logs:
  - HTTP status code
  - Error details (sanitized)
- **Missing**:
  - Request ID (for tracing)
  - User ID (from JWT context)
  - Article ID (for debugging)
  - Response time (for performance monitoring)
  - Request path and method

**Distributed Tracing**:
- **No** - No mention of distributed tracing
- No trace ID propagation
- No integration with OpenTelemetry or similar

**Issues**:

1. **No Request ID Tracking**: Without a request ID, it's difficult to:
   - Trace a request through the system
   - Correlate logs from different components
   - Debug issues reported by users

2. **No Access Logging**: The design doesn't specify:
   - Whether successful requests are logged
   - What information is logged for each request
   - Log retention policy

3. **No Performance Metrics**: The design doesn't include:
   - Request duration logging
   - Database query duration
   - Percentile tracking (p50, p95, p99)
   - Error rate monitoring

4. **No Context Propagation**: The design doesn't specify:
   - How to pass request ID through layers
   - How to extract user ID from JWT context
   - How to add contextual information to logs

5. **No Log Level Configuration**: The design doesn't specify:
   - When to use Error vs Warn vs Info
   - Whether debug logs are needed
   - How log levels are configured

6. **No Sensitive Data Handling**: While `respond.SafeError` calls `SanitizeError`, the design doesn't specify:
   - What data should be sanitized (passwords, tokens, PII)
   - How to prevent accidental logging of sensitive data

**Recommendation**:

1. **Add Request ID Middleware**:
   ```go
   // In internal/handler/http/middleware/request_id.go
   func RequestID(next http.Handler) http.Handler {
       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           requestID := r.Header.Get("X-Request-ID")
           if requestID == "" {
               requestID = uuid.New().String()
           }

           // Add to response header
           w.Header().Set("X-Request-ID", requestID)

           // Add to context
           ctx := context.WithValue(r.Context(), "request_id", requestID)
           next.ServeHTTP(w, r.WithContext(ctx))
       })
   }
   ```

2. **Add Access Logging Middleware**:
   ```go
   // In internal/handler/http/middleware/access_log.go
   func AccessLog(next http.Handler) http.Handler {
       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           start := time.Now()

           // Wrap response writer to capture status code
           wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

           next.ServeHTTP(wrapped, r)

           duration := time.Since(start)

           slog.InfoContext(r.Context(), "http request",
               slog.String("method", r.Method),
               slog.String("path", r.URL.Path),
               slog.Int("status", wrapped.statusCode),
               slog.Duration("duration", duration),
               slog.String("request_id", r.Context().Value("request_id").(string)),
               slog.String("user_agent", r.Header.Get("User-Agent")),
               slog.String("remote_addr", r.RemoteAddr))
       })
   }
   ```

3. **Enhance Handler Logging**:
   ```go
   func (h GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
       ctx := r.Context()
       requestID := ctx.Value("request_id").(string)

       id, err := pathutil.ExtractID(r.URL.Path, "/articles/")
       if err != nil {
           slog.WarnContext(ctx, "invalid article id",
               slog.String("request_id", requestID),
               slog.String("path", r.URL.Path),
               slog.Any("error", err))
           respond.SafeError(w, http.StatusBadRequest, err)
           return
       }

       start := time.Now()
       article, sourceName, err := h.Svc.GetWithSource(ctx, id)
       dbDuration := time.Since(start)

       if err != nil {
           if errors.Is(err, article.ErrArticleNotFound) {
               slog.InfoContext(ctx, "article not found",
                   slog.String("request_id", requestID),
                   slog.Int64("article_id", id))
               respond.SafeError(w, http.StatusNotFound, err)
               return
           }
           slog.ErrorContext(ctx, "failed to get article",
               slog.String("request_id", requestID),
               slog.Int64("article_id", id),
               slog.Duration("db_duration", dbDuration),
               slog.Any("error", err))
           respond.SafeError(w, http.StatusInternalServerError, err)
           return
       }

       slog.DebugContext(ctx, "article retrieved successfully",
           slog.String("request_id", requestID),
           slog.Int64("article_id", id),
           slog.Duration("db_duration", dbDuration))

       // ... success response
   }
   ```

4. **Add Repository Layer Logging**:
   ```go
   func (r *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
       start := time.Now()

       var article entity.Article
       var sourceName string

       err := r.db.QueryRowContext(ctx, query, id).Scan(...)

       queryDuration := time.Since(start)

       if err != nil {
           if errors.Is(err, sql.ErrNoRows) {
               slog.DebugContext(ctx, "article not found in database",
                   slog.Int64("article_id", id),
                   slog.Duration("query_duration", queryDuration))
               return nil, "", ErrArticleNotFound
           }
           slog.ErrorContext(ctx, "database query failed",
               slog.Int64("article_id", id),
               slog.Duration("query_duration", queryDuration),
               slog.Any("error", err))
           return nil, "", fmt.Errorf("failed to get article: %w", err)
       }

       slog.DebugContext(ctx, "article query successful",
           slog.Int64("article_id", id),
           slog.Duration("query_duration", queryDuration))

       return &article, sourceName, nil
   }
   ```

5. **Define Log Level Strategy**:
   ```markdown
   ### Logging Levels

   - **DEBUG**: Successful operations with detailed context (disabled in production)
     - Article retrieved successfully
     - Database query successful

   - **INFO**: Important state changes and access logs
     - HTTP request completed
     - User authentication successful

   - **WARN**: Recoverable errors and validation failures
     - Invalid article ID format
     - Article not found (user error)

   - **ERROR**: System errors requiring investigation
     - Database connection failure
     - Internal server errors
     - Unexpected errors
   ```

6. **Add Metrics Specification** (future enhancement):
   ```markdown
   ### Metrics to Track

   - `http_requests_total{method, path, status}` - Total requests
   - `http_request_duration_seconds{method, path}` - Request duration histogram
   - `db_query_duration_seconds{operation}` - Database query duration
   - `article_not_found_total` - Article not found count
   - `db_errors_total` - Database error count
   ```

**Reliability Benefit**:
- Request ID enables end-to-end tracing
- Structured logging enables efficient log searching
- Performance metrics enable proactive monitoring
- Debug logs enable troubleshooting in development
- Access logs provide audit trail

---

## Reliability Risk Assessment

### High Risk Areas

1. **Database Single Point of Failure**
   - **Description**: The database is a single point of failure. If PostgreSQL becomes unavailable, all requests fail with 500 errors.
   - **Impact**: Complete service outage
   - **Likelihood**: Medium (depends on database infrastructure)
   - **Mitigation**:
     - Add database health checks
     - Implement connection pooling with timeouts
     - Consider read replicas for redundancy
     - Add response caching for frequently accessed articles

2. **Missing Error Type Definitions**
   - **Description**: The design references error types that don't exist in the codebase yet (`ErrArticleNotFound`, `ErrInvalidArticleID`).
   - **Impact**: Inconsistent error handling, unclear error propagation
   - **Likelihood**: High (will cause issues during implementation)
   - **Mitigation**:
     - Define error types before implementation
     - Document error propagation strategy
     - Ensure errors are exported from appropriate packages

### Medium Risk Areas

1. **No Timeout Configuration**
   - **Description**: Database queries don't have timeout configuration, which can lead to resource exhaustion during database slowdowns.
   - **Impact**: Handler goroutines hang indefinitely, memory leak, service degradation
   - **Likelihood**: Medium (depends on database performance)
   - **Mitigation**:
     - Add context timeout to database queries (5 seconds)
     - Add HTTP server timeouts (ReadTimeout, WriteTimeout)
     - Add connection pool timeout configuration

2. **Limited Observability**
   - **Description**: No request tracing, limited logging context, no performance metrics.
   - **Impact**: Difficult to debug issues, slow incident response, poor visibility into system health
   - **Likelihood**: High (will impact operations)
   - **Mitigation**:
     - Add request ID middleware
     - Add access logging middleware
     - Add structured logging with context
     - Consider adding distributed tracing (OpenTelemetry)

3. **Orphaned Article Handling Ambiguity**
   - **Description**: If an article's source is deleted, the INNER JOIN will exclude it, returning 404. This behavior is not explicitly documented.
   - **Impact**: Confusing error responses, potential data integrity concerns
   - **Likelihood**: Low (if foreign key constraints are properly configured)
   - **Mitigation**:
     - Document INNER JOIN behavior for orphaned articles
     - Add ON DELETE CASCADE or ON DELETE SET NULL constraint
     - Consider using LEFT JOIN if orphaned articles should be returned

### Mitigation Strategies

1. **Implement Database Resilience**:
   - Add connection pool configuration (max connections, timeouts)
   - Add context timeouts to database queries
   - Add database health check endpoint
   - Consider read replicas for redundancy

2. **Enhance Error Handling**:
   - Define article-specific error types
   - Add structured error logging with context
   - Document error propagation strategy
   - Add error rate monitoring

3. **Improve Observability**:
   - Add request ID middleware for tracing
   - Add access logging middleware
   - Add performance metrics (request duration, database query duration)
   - Add debug logging for troubleshooting

4. **Document Edge Cases**:
   - Clarify orphaned article handling (INNER JOIN behavior)
   - Document NULL value handling
   - Document concurrent update behavior
   - Document isolation level requirements

5. **Add Testing for Failure Scenarios**:
   - Test database timeout handling
   - Test connection pool exhaustion
   - Test orphaned article handling
   - Test concurrent deletion scenarios

---

## Action Items for Designer

**Status: Request Changes**

The design requires improvements in the following areas:

### Critical (Must Fix Before Implementation)

1. **Define Article-Specific Error Types**:
   - Create `internal/usecase/article/errors.go` with:
     - `ErrArticleNotFound = errors.New("article not found")`
     - `ErrInvalidArticleID = errors.New("invalid article id")`
     - `ErrSourceNotFound = errors.New("source not found for article")`
   - Document which layer should return which errors
   - Update design document Section 7 with error type definitions

2. **Add Database Timeout Configuration**:
   - Specify context timeout for database queries (recommended: 5 seconds)
   - Specify connection pool configuration:
     - MaxOpenConns (recommended: 25)
     - MaxIdleConns (recommended: 5)
     - ConnMaxLifetime (recommended: 5 minutes)
     - ConnMaxIdleTime (recommended: 2 minutes)
   - Update design document Section 3 (Architecture Design) with timeout specifications

3. **Clarify Orphaned Article Handling**:
   - Document behavior when article's source is deleted
   - Decide between INNER JOIN (excludes orphaned articles) vs LEFT JOIN (returns articles with NULL source)
   - Add database constraint specification (ON DELETE CASCADE vs ON DELETE SET NULL)
   - Update design document Section 7 (Error Handling) and Section 9 (Testing Strategy)

### Important (Should Fix Before Implementation)

4. **Add Structured Logging Specification**:
   - Define log levels for different scenarios (DEBUG, INFO, WARN, ERROR)
   - Specify what context should be logged:
     - Request ID (for tracing)
     - User ID (from JWT)
     - Article ID
     - Response time
     - Database query duration
   - Add logging specification to Section 7 (Error Handling)

5. **Add NULL Value Handling Specification**:
   - Document how NULL values in optional fields are handled
   - Specify whether to use `sql.NullString` or zero values
   - Add code examples to Section 4 (Data Model)

6. **Add Request ID and Access Logging**:
   - Specify request ID middleware for tracing
   - Specify access logging middleware for audit trail
   - Add middleware specification to Section 3 (Architecture Design)

### Nice to Have (Future Enhancements)

7. **Consider Response Caching**:
   - Add caching strategy for frequently accessed articles
   - Specify cache invalidation policy
   - Add to Section 12 (Future Enhancements)

8. **Add Performance Metrics Specification**:
   - Specify metrics to track (request duration, error rate, database query duration)
   - Add to Section 9 (Testing Strategy) or new section

9. **Add Database Health Check Endpoint**:
   - Specify health check endpoint for monitoring
   - Add to Section 12 (Future Enhancements) or separate design

---

## Structured Data

```yaml
evaluation_result:
  evaluator: "design-reliability-evaluator"
  design_document: "/Users/yujitsuchiya/catchup-feed/docs/designs/article-detail-endpoint.md"
  timestamp: "2025-12-06T00:00:00Z"
  overall_judgment:
    status: "Request Changes"
    overall_score: 3.4
  detailed_scores:
    error_handling:
      score: 3.5
      weight: 0.35
    fault_tolerance:
      score: 2.5
      weight: 0.30
    transaction_management:
      score: 4.0
      weight: 0.20
    logging_observability:
      score: 3.5
      weight: 0.15
  failure_scenarios:
    - scenario: "Database unavailable"
      handled: true
      strategy: "Returns 500 Internal Server Error, but no retry/fallback strategy"
    - scenario: "Database query timeout"
      handled: false
      strategy: "Not specified - needs context timeout configuration"
    - scenario: "Invalid article ID"
      handled: true
      strategy: "pathutil.ExtractID validates ID, returns 400 Bad Request"
    - scenario: "Article not found"
      handled: true
      strategy: "Repository returns sql.ErrNoRows → Service returns ErrArticleNotFound → Handler returns 404"
    - scenario: "Article exists but source deleted"
      handled: true
      strategy: "INNER JOIN excludes orphaned articles, returns 404 (behavior should be documented)"
    - scenario: "NULL values in optional fields"
      handled: false
      strategy: "Not specified - needs NULL handling specification"
    - scenario: "Unauthorized access"
      handled: true
      strategy: "auth.Authz middleware validates JWT, returns 401/403"
  reliability_risks:
    - severity: "high"
      area: "Database single point of failure"
      description: "No fallback mechanism if PostgreSQL becomes unavailable"
      mitigation: "Add connection pool timeouts, health checks, consider read replicas and caching"
    - severity: "high"
      area: "Missing error type definitions"
      description: "ErrArticleNotFound and ErrInvalidArticleID referenced but not defined"
      mitigation: "Define error types in internal/usecase/article/errors.go"
    - severity: "medium"
      area: "No timeout configuration"
      description: "Database queries lack timeout configuration"
      mitigation: "Add context timeout (5s) to database queries"
    - severity: "medium"
      area: "Limited observability"
      description: "No request tracing, limited logging context"
      mitigation: "Add request ID middleware, structured logging, performance metrics"
    - severity: "medium"
      area: "Orphaned article handling ambiguity"
      description: "Unclear how articles with deleted sources are handled"
      mitigation: "Document INNER JOIN behavior, add database constraints"
    - severity: "low"
      area: "No retry mechanism"
      description: "No server-side retry for transient database failures"
      mitigation: "Consider adding retry logic with exponential backoff (future enhancement)"
  error_handling_coverage: 70
  fault_tolerance_coverage: 40
  observability_coverage: 55
  action_items:
    critical:
      - "Define article-specific error types (ErrArticleNotFound, ErrInvalidArticleID)"
      - "Add database timeout configuration (context timeout, connection pool)"
      - "Clarify orphaned article handling (INNER JOIN vs LEFT JOIN, database constraints)"
    important:
      - "Add structured logging specification (log levels, context fields)"
      - "Add NULL value handling specification"
      - "Add request ID and access logging middleware"
    nice_to_have:
      - "Consider response caching strategy"
      - "Add performance metrics specification"
      - "Add database health check endpoint"
```

---

**Evaluation Complete**: The design shows good understanding of error handling patterns and clean architecture, but requires improvements in fault tolerance, timeout configuration, and observability before proceeding to implementation.
