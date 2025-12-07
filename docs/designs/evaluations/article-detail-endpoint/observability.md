# Design Observability Evaluation - Article Detail Endpoint

**Evaluator**: design-observability-evaluator
**Design Document**: /Users/yujitsuchiya/catchup-feed/docs/designs/article-detail-endpoint.md
**Evaluated**: 2025-12-06T00:00:00Z

---

## Overall Judgment

**Status**: Request Changes
**Overall Score**: 2.1 / 5.0

---

## Detailed Scores

### 1. Logging Strategy: 2.0 / 5.0 (Weight: 35%)

**Findings**:
- Minimal logging strategy mentioned
- Only references `slog` for database error logging
- No structured logging framework defined
- No log context specified (userId, requestId, etc.)
- No log levels strategy defined
- No centralization approach mentioned

**Logging Framework**:
- `slog` mentioned for error logging only
- No comprehensive logging strategy

**Log Context**:
- No structured fields mentioned
- Missing critical context:
  - User ID (from JWT token)
  - Request ID / Trace ID
  - Article ID being accessed
  - Query duration
  - Source ID
  - Error details with stack traces

**Log Levels**:
- Not specified
- No guidance on DEBUG, INFO, WARN, ERROR usage

**Centralization**:
- Not specified
- No mention of log aggregation (ELK, CloudWatch, etc.)

**Issues**:
1. **Incomplete logging coverage**: Only database errors mentioned, no success logs or request tracking
2. **No request tracing**: Cannot correlate logs across layers (Handler → Service → Repository)
3. **No searchability**: Missing structured fields for filtering logs by user, article, or request
4. **No log retention policy**: No mention of log storage or rotation

**Recommendation**:

Implement comprehensive structured logging:

```go
// Handler layer logging
logger.Info("article_detail_request",
    slog.String("request_id", requestID),
    slog.String("user_id", userID),
    slog.Int64("article_id", articleID),
    slog.String("user_role", role),
    slog.String("endpoint", "/articles/{id}"),
)

// Service layer logging
logger.Debug("fetching_article_with_source",
    slog.String("request_id", requestID),
    slog.Int64("article_id", articleID),
)

// Repository layer logging
logger.Debug("executing_join_query",
    slog.String("request_id", requestID),
    slog.Int64("article_id", articleID),
    slog.String("query", "SELECT ... FROM articles a INNER JOIN sources s"),
)

// Success logging
logger.Info("article_detail_retrieved",
    slog.String("request_id", requestID),
    slog.Int64("article_id", articleID),
    slog.String("source_name", sourceName),
    slog.Duration("duration_ms", duration),
    slog.Int("http_status", 200),
)

// Error logging with context
logger.Error("article_not_found",
    slog.String("request_id", requestID),
    slog.Int64("article_id", articleID),
    slog.String("error", err.Error()),
    slog.Int("http_status", 404),
)

// Database error logging
logger.Error("database_query_failed",
    slog.String("request_id", requestID),
    slog.Int64("article_id", articleID),
    slog.String("error", err.Error()),
    slog.String("query", "SELECT ... FROM articles"),
    slog.Duration("duration_ms", duration),
    slog.Int("http_status", 500),
)
```

**Log Centralization**:
- Configure `slog` to output JSON format
- Integrate with log aggregation system (CloudWatch, ELK, Loki)
- Set up log retention policies (e.g., 30 days)

### 2. Metrics & Monitoring: 2.0 / 5.0 (Weight: 30%)

**Findings**:
- No metrics collection mentioned
- No monitoring system specified
- No alerts defined
- No dashboards mentioned
- Performance tests mentioned (benchmarks) but no production metrics

**Key Metrics**:
- Not specified

**Monitoring System**:
- Not specified (Prometheus, Datadog, CloudWatch, etc.)

**Alerts**:
- Not specified

**Dashboards**:
- Not mentioned

**Issues**:
1. **No visibility into system health**: Cannot monitor if endpoint is functioning properly
2. **No performance tracking**: Cannot detect degradation over time
3. **No error rate monitoring**: Cannot alert when errors spike
4. **No SLI/SLO defined**: No service level targets

**Recommendation**:

Define comprehensive metrics collection:

**Key Metrics to Track**:

1. **Request Metrics**:
   - `article_detail_requests_total` (counter) - Total requests by status code
   - `article_detail_request_duration_seconds` (histogram) - Response time distribution
   - `article_detail_active_requests` (gauge) - Current active requests

2. **Error Metrics**:
   - `article_detail_errors_total` (counter) - Errors by type (400, 404, 500)
   - `article_detail_not_found_total` (counter) - 404 errors specifically
   - `article_detail_database_errors_total` (counter) - Database failures

3. **Database Metrics**:
   - `article_detail_query_duration_seconds` (histogram) - JOIN query performance
   - `article_detail_db_connection_errors_total` (counter) - Connection failures

4. **Business Metrics**:
   - `article_detail_by_source` (counter) - Access patterns by source
   - `article_detail_cache_hit_ratio` (gauge) - If caching implemented

**Implementation Example (Prometheus)**:

```go
// Define metrics
var (
    requestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "article_detail_requests_total",
            Help: "Total requests to article detail endpoint",
        },
        []string{"status", "method"},
    )

    requestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "article_detail_request_duration_seconds",
            Help:    "Request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"status"},
    )

    queryDuration = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "article_detail_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
        },
    )
)

// Instrument handler
func (h *GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        requestDuration.WithLabelValues(fmt.Sprint(statusCode)).Observe(duration)
        requestsTotal.WithLabelValues(fmt.Sprint(statusCode), r.Method).Inc()
    }()

    // Handler logic...
}
```

**Alert Definitions**:

```yaml
# Prometheus alerts
groups:
  - name: article_detail_alerts
    interval: 1m
    rules:
      # High error rate
      - alert: ArticleDetailHighErrorRate
        expr: rate(article_detail_errors_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Article detail endpoint error rate above 5%"

      # Slow response time
      - alert: ArticleDetailSlowResponse
        expr: histogram_quantile(0.95, article_detail_request_duration_seconds) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Article detail p95 response time above 100ms"

      # Database query slow
      - alert: ArticleDetailSlowQuery
        expr: histogram_quantile(0.95, article_detail_query_duration_seconds) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Article detail database query p95 above 50ms"

      # High 404 rate (possible data integrity issue)
      - alert: ArticleDetailHighNotFoundRate
        expr: rate(article_detail_not_found_total[5m]) > 0.1
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "Unusually high 404 rate on article detail endpoint"
```

**Dashboard Recommendations**:

```
Grafana Dashboard: Article Detail Endpoint
- Request Rate (requests/sec)
- Response Time (p50, p95, p99)
- Error Rate by status code (400, 404, 500)
- Database Query Duration
- Active Requests (concurrency)
- Top Accessed Articles
```

### 3. Distributed Tracing: 2.0 / 5.0 (Weight: 20%)

**Findings**:
- No tracing framework mentioned
- No trace ID propagation strategy
- Cannot trace requests across layers (Handler → Service → Repository → Database)
- No span instrumentation discussed

**Tracing Framework**:
- Not specified (OpenTelemetry, Jaeger, Zipkin, etc.)

**Trace ID Propagation**:
- Not mentioned
- No request ID generation strategy

**Span Instrumentation**:
- Not mentioned
- Cannot identify bottlenecks in request flow

**Issues**:
1. **No request correlation**: Cannot link logs from different layers for same request
2. **No bottleneck identification**: Cannot see where time is spent (handler, service, DB query)
3. **No distributed context**: If system scales to microservices, no trace propagation
4. **Difficult debugging**: Cannot trace single request from entry to completion

**Recommendation**:

Implement distributed tracing with OpenTelemetry:

**1. Add OpenTelemetry Dependencies**:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)
```

**2. Instrument Handler Layer**:

```go
func (h *GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := otel.Tracer("article-handler").Start(r.Context(), "GET /articles/{id}")
    defer span.End()

    // Extract article ID
    id, err := pathutil.ExtractID(r.URL.Path, "/articles/")
    if err != nil {
        span.RecordError(err)
        span.SetAttributes(attribute.String("error.type", "invalid_id"))
        respond.Error(w, http.StatusBadRequest, err)
        return
    }

    span.SetAttributes(
        attribute.Int64("article.id", id),
        attribute.String("http.method", r.Method),
        attribute.String("http.route", "/articles/{id}"),
    )

    // Call service with traced context
    article, sourceName, err := h.service.GetWithSource(ctx, id)
    // ...
}
```

**3. Instrument Service Layer**:

```go
func (s *Service) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
    ctx, span := otel.Tracer("article-service").Start(ctx, "GetWithSource")
    defer span.End()

    span.SetAttributes(attribute.Int64("article.id", id))

    if id <= 0 {
        span.RecordError(ErrInvalidArticleID)
        return nil, "", ErrInvalidArticleID
    }

    // Call repository with traced context
    article, sourceName, err := s.repo.GetWithSource(ctx, id)
    // ...
}
```

**4. Instrument Repository Layer**:

```go
func (r *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error) {
    ctx, span := otel.Tracer("article-repository").Start(ctx, "GetWithSource")
    defer span.End()

    span.SetAttributes(
        attribute.Int64("article.id", id),
        attribute.String("db.system", "postgresql"),
        attribute.String("db.operation", "SELECT JOIN"),
    )

    queryStart := time.Now()
    row := r.db.QueryRowContext(ctx, query, id)
    queryDuration := time.Since(queryStart)

    span.SetAttributes(attribute.Int64("db.query_duration_ms", queryDuration.Milliseconds()))

    // ...
}
```

**5. Trace ID Propagation**:

```go
// Generate request ID at entry point (middleware)
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        ctx := context.WithValue(r.Context(), "request_id", requestID)
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**6. Integrate with Jaeger/Zipkin**:

```go
// Initialize tracer
import (
    "go.opentelemetry.io/otel/exporters/jaeger"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() {
    exporter, _ := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://jaeger:14268/api/traces")))
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("catchup-feed-api"),
        )),
    )
    otel.SetTracerProvider(tp)
}
```

**Trace Visualization Benefits**:
- See full request path: `Handler (5ms) → Service (2ms) → Repository (40ms) → Database (35ms)`
- Identify bottlenecks: Database query taking longest time
- Correlate errors across layers
- Debug production issues with full request trace

### 4. Health Checks & Diagnostics: 2.5 / 5.0 (Weight: 15%)

**Findings**:
- No health check endpoints mentioned
- No diagnostic endpoints (/metrics, /debug)
- No dependency health checks (database, S3 if future caching)
- Performance benchmarks mentioned but not production health monitoring

**Health Check Endpoints**:
- Not specified

**Dependency Checks**:
- Database health not monitored
- No mention of connection pool status

**Diagnostic Endpoints**:
- Not specified
- No `/health` endpoint
- No `/metrics` endpoint (for Prometheus scraping)
- No `/debug/pprof` for profiling

**Issues**:
1. **No load balancer health checks**: Cannot determine if instance is healthy
2. **No proactive monitoring**: Cannot detect issues before user complaints
3. **No runtime diagnostics**: Cannot profile CPU/memory usage in production
4. **No dependency visibility**: Cannot see if database is responsive

**Recommendation**:

Implement comprehensive health checks and diagnostics:

**1. Health Check Endpoint**:

```go
// GET /health
func HealthHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
        defer cancel()

        health := struct {
            Status      string            `json:"status"`
            Timestamp   time.Time         `json:"timestamp"`
            Uptime      string            `json:"uptime"`
            Dependencies map[string]string `json:"dependencies"`
        }{
            Status:      "healthy",
            Timestamp:   time.Now(),
            Uptime:      time.Since(startTime).String(),
            Dependencies: make(map[string]string),
        }

        // Check database connection
        if err := db.PingContext(ctx); err != nil {
            health.Status = "unhealthy"
            health.Dependencies["database"] = "unhealthy: " + err.Error()
            w.WriteHeader(http.StatusServiceUnavailable)
        } else {
            health.Dependencies["database"] = "healthy"
        }

        // Check connection pool stats
        stats := db.Stats()
        if stats.OpenConnections >= stats.MaxOpenConnections {
            health.Status = "degraded"
            health.Dependencies["database_pool"] = "degraded: pool exhausted"
        } else {
            health.Dependencies["database_pool"] = fmt.Sprintf("healthy: %d/%d connections",
                stats.OpenConnections, stats.MaxOpenConnections)
        }

        json.NewEncoder(w).Encode(health)
    }
}
```

**2. Readiness Check** (for Kubernetes):

```go
// GET /ready
func ReadinessHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
        defer cancel()

        // Quick database check
        if err := db.PingContext(ctx); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            json.NewEncoder(w).Encode(map[string]string{
                "status": "not_ready",
                "reason": "database_unavailable",
            })
            return
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
    }
}
```

**3. Liveness Check** (for Kubernetes):

```go
// GET /live
func LivenessHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
    }
}
```

**4. Metrics Endpoint** (for Prometheus):

```go
// GET /metrics
import "github.com/prometheus/client_golang/prometheus/promhttp"

func RegisterMetricsEndpoint(mux *http.ServeMux) {
    mux.Handle("/metrics", promhttp.Handler())
}
```

**5. Debug Endpoints** (Development/Staging only):

```go
import _ "net/http/pprof"

// GET /debug/pprof/
func RegisterDebugEndpoints(mux *http.ServeMux) {
    // CPU profiling: /debug/pprof/profile
    // Heap profiling: /debug/pprof/heap
    // Goroutine profiling: /debug/pprof/goroutine
    // etc.
}
```

**6. Database Connection Pool Monitoring**:

```go
// Expose DB pool metrics
var (
    dbOpenConnections = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "db_open_connections",
        Help: "Number of open database connections",
    })

    dbInUseConnections = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "db_in_use_connections",
        Help: "Number of in-use database connections",
    })
)

// Collect stats periodically
func CollectDBStats(db *sql.DB) {
    ticker := time.NewTicker(10 * time.Second)
    go func() {
        for range ticker.C {
            stats := db.Stats()
            dbOpenConnections.Set(float64(stats.OpenConnections))
            dbInUseConnections.Set(float64(stats.InUse))
        }
    }()
}
```

**Health Check Integration**:
- Load balancer checks `GET /health` every 10 seconds
- Kubernetes liveness probe: `GET /live`
- Kubernetes readiness probe: `GET /ready`
- Prometheus scrapes `GET /metrics` every 15 seconds

---

## Observability Gaps

### Critical Gaps

1. **No Logging Strategy**: Cannot search logs for specific users, requests, or articles. Debugging production issues will be extremely difficult.
   - **Impact**: High - Unable to diagnose issues efficiently
   - **Priority**: P0 - Must fix before production

2. **No Metrics Collection**: Cannot monitor endpoint health, error rates, or performance degradation.
   - **Impact**: High - No visibility into system behavior
   - **Priority**: P0 - Must fix before production

3. **No Distributed Tracing**: Cannot trace requests across layers or identify bottlenecks.
   - **Impact**: Medium - Difficult to optimize performance
   - **Priority**: P1 - Should fix before production

### Minor Gaps

1. **No Health Check Endpoints**: Load balancers cannot determine instance health.
   - **Impact**: Medium - May route traffic to unhealthy instances
   - **Priority**: P1 - Should fix before production

2. **No Alert Definitions**: Team will not be notified when errors occur or performance degrades.
   - **Impact**: Medium - Reactive instead of proactive
   - **Priority**: P1 - Should fix before production

3. **No Dashboards**: No visual monitoring of endpoint performance.
   - **Impact**: Low - Can be added post-launch
   - **Priority**: P2 - Nice to have

---

## Recommended Observability Stack

Based on design and Go ecosystem, recommend:

**Logging**:
- **Framework**: `slog` (standard library, already mentioned)
- **Format**: JSON for structured logging
- **Centralization**: CloudWatch Logs (AWS) or ELK Stack (self-hosted)
- **Context Fields**: requestID, userID, articleID, duration, error, httpStatus

**Metrics**:
- **Collection**: Prometheus client library (`github.com/prometheus/client_golang`)
- **Storage**: Prometheus server
- **Visualization**: Grafana dashboards
- **Metrics Type**: Counters (requests, errors), Histograms (duration), Gauges (active requests)

**Tracing**:
- **Framework**: OpenTelemetry Go SDK
- **Backend**: Jaeger (open-source) or AWS X-Ray
- **Sampling**: 100% for low-traffic endpoints, 10% for high-traffic
- **Context Propagation**: Via `context.Context` across all layers

**Health Checks**:
- **Endpoints**: `/health` (comprehensive), `/ready` (Kubernetes), `/live` (Kubernetes)
- **Dependencies**: Database ping, connection pool stats
- **Format**: JSON response with status and dependency details

**Dashboards**:
- **Tool**: Grafana
- **Key Metrics**: Request rate, error rate, p95/p99 latency, database query duration
- **Alerts**: High error rate (>5%), slow response (p95 > 100ms), database errors

---

## Action Items for Designer

To improve observability score from **2.1/5.0 to ≥3.0/5.0**, implement the following:

### Priority 0 (Critical - Must Have)

1. **Add Structured Logging Section to Design**:
   - Define log context fields: requestID, userID, articleID, duration, error, httpStatus
   - Specify log levels: INFO for success, ERROR for failures, DEBUG for development
   - Define logging at each layer: Handler, Service, Repository
   - Include example log entries

2. **Add Metrics Collection Section to Design**:
   - Define key metrics: request count, duration histogram, error count, query duration
   - Specify Prometheus as monitoring system
   - Include code examples for metric instrumentation
   - Define metric labels: status, method, error_type

3. **Add Distributed Tracing Section to Design**:
   - Specify OpenTelemetry as tracing framework
   - Define span instrumentation at each layer
   - Show trace ID propagation via context
   - Include example trace flow

### Priority 1 (Important - Should Have)

4. **Add Health Check Endpoints to API Design**:
   - `GET /health` - Comprehensive health check with database status
   - `GET /ready` - Kubernetes readiness probe
   - `GET /live` - Kubernetes liveness probe
   - Include response format examples

5. **Add Alert Definitions**:
   - High error rate alert (>5% for 5 minutes)
   - Slow response alert (p95 > 100ms for 5 minutes)
   - Database error alert
   - High 404 rate alert (possible data integrity issue)

### Priority 2 (Nice to Have)

6. **Add Dashboard Requirements**:
   - List key dashboard panels: request rate, latency, error rate
   - Specify visualization tool (Grafana)
   - Define SLI/SLO targets (e.g., p95 < 100ms, error rate < 1%)

---

## Example Observability Section for Design Document

Add this section to the design document:

```markdown
## X. Observability & Monitoring

### Logging Strategy

**Framework**: `slog` (Go standard library)

**Log Format**: JSON for structured logging

**Log Context Fields**:
- `request_id`: Unique request identifier
- `user_id`: User ID from JWT token
- `article_id`: Article ID being accessed
- `source_id`: Source ID (if retrieved)
- `source_name`: Source name (if retrieved)
- `duration_ms`: Request duration in milliseconds
- `http_status`: HTTP status code
- `error`: Error message (if any)
- `layer`: Handler/Service/Repository

**Log Levels**:
- `DEBUG`: Development debugging, query execution details
- `INFO`: Successful requests, important state changes
- `WARN`: Degraded performance, unusual patterns
- `ERROR`: Errors, exceptions, failures

**Example Logs**:

```json
{
  "time": "2025-12-06T10:30:45Z",
  "level": "INFO",
  "msg": "article_detail_retrieved",
  "request_id": "abc-123-def",
  "user_id": "user-456",
  "article_id": 789,
  "source_name": "Go Blog",
  "duration_ms": 42,
  "http_status": 200,
  "layer": "handler"
}
```

### Metrics Collection

**Monitoring System**: Prometheus + Grafana

**Key Metrics**:

1. `article_detail_requests_total{status, method}` - Total requests (Counter)
2. `article_detail_request_duration_seconds{status}` - Response time (Histogram)
3. `article_detail_errors_total{error_type}` - Errors by type (Counter)
4. `article_detail_query_duration_seconds` - DB query time (Histogram)
5. `article_detail_active_requests` - Concurrent requests (Gauge)

**Metrics Endpoint**: `GET /metrics` (Prometheus format)

**Alert Rules**:
- High error rate: >5% for 5 minutes
- Slow response: p95 > 100ms for 5 minutes
- Database errors: >10 per minute

### Distributed Tracing

**Tracing Framework**: OpenTelemetry Go SDK

**Trace Backend**: Jaeger

**Span Hierarchy**:
```
GET /articles/{id} (Handler)
  └─ GetWithSource (Service)
      └─ GetWithSource (Repository)
          └─ SELECT JOIN (Database)
```

**Span Attributes**:
- `article.id`: Article ID
- `http.method`: HTTP method
- `http.route`: Route pattern
- `db.system`: postgresql
- `db.operation`: SELECT JOIN
- `error.type`: Error classification

**Trace ID Propagation**: Via `context.Context` and `X-Request-ID` header

### Health Checks

**Endpoints**:
- `GET /health` - Comprehensive health (database, pool, uptime)
- `GET /ready` - Readiness probe (database ping)
- `GET /live` - Liveness probe (process alive)

**Response Format**:
```json
{
  "status": "healthy",
  "timestamp": "2025-12-06T10:30:45Z",
  "uptime": "24h15m",
  "dependencies": {
    "database": "healthy",
    "database_pool": "healthy: 5/20 connections"
  }
}
```

### Dashboards

**Grafana Dashboard - Article Detail Endpoint**:
- Request Rate (requests/sec)
- Response Time (p50, p95, p99)
- Error Rate by Status Code
- Database Query Duration
- Active Requests
- Top Accessed Articles
```
```

---

## Structured Data

```yaml
evaluation_result:
  evaluator: "design-observability-evaluator"
  design_document: "/Users/yujitsuchiya/catchup-feed/docs/designs/article-detail-endpoint.md"
  timestamp: "2025-12-06T00:00:00Z"
  overall_judgment:
    status: "Request Changes"
    overall_score: 2.1
  detailed_scores:
    logging_strategy:
      score: 2.0
      weight: 0.35
      weighted_score: 0.70
    metrics_monitoring:
      score: 2.0
      weight: 0.30
      weighted_score: 0.60
    distributed_tracing:
      score: 2.0
      weight: 0.20
      weighted_score: 0.40
    health_checks:
      score: 2.5
      weight: 0.15
      weighted_score: 0.375
  observability_gaps:
    - severity: "critical"
      gap: "No structured logging strategy"
      impact: "Cannot search logs for specific users, requests, or articles. Debugging production issues extremely difficult."
    - severity: "critical"
      gap: "No metrics collection defined"
      impact: "No visibility into endpoint health, error rates, or performance. Cannot detect degradation."
    - severity: "critical"
      gap: "No distributed tracing framework"
      impact: "Cannot trace requests across layers or identify bottlenecks in request flow."
    - severity: "minor"
      gap: "No health check endpoints"
      impact: "Load balancers cannot determine instance health. May route to unhealthy instances."
    - severity: "minor"
      gap: "No alert definitions"
      impact: "Team not notified when errors occur. Reactive instead of proactive monitoring."
  observability_coverage: 42%
  recommended_stack:
    logging: "slog (JSON format) + CloudWatch/ELK"
    metrics: "Prometheus + Grafana"
    tracing: "OpenTelemetry + Jaeger/AWS X-Ray"
    dashboards: "Grafana"
    health_checks: "/health, /ready, /live endpoints"
```
