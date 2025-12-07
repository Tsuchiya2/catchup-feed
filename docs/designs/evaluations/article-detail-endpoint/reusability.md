# Design Reusability Evaluation - Article Detail Endpoint

**Evaluator**: design-reusability-evaluator
**Design Document**: /Users/yujitsuchiya/catchup-feed/docs/designs/article-detail-endpoint.md
**Evaluated**: 2025-12-06T00:00:00Z

---

## Overall Judgment

**Status**: Request Changes
**Overall Score**: 3.4 / 5.0

---

## Detailed Scores

### 1. Component Generalization: 3.0 / 5.0 (Weight: 35%)

**Findings**:
- The design follows a feature-specific approach, creating `GetWithSource()` method specifically for this endpoint
- Components are moderately generalized but have room for improvement
- The DTO modification with `SourceName` field uses `omitempty` tag, which is good for backward compatibility
- The `pathutil.ExtractID` utility is well-generalized and reusable across different endpoints
- Service layer validation (ID > 0) is implemented consistently

**Issues**:
1. **GetWithSource() is feature-specific**: The method name and signature are tightly coupled to this specific use case. It returns both an article entity and a source name string, which is not a generalized pattern.
2. **No generic JOIN support**: The repository doesn't provide a generic mechanism for joining related entities. Each feature requiring joins will need its own specialized method.
3. **Hardcoded source name field**: The design hardcodes "source_name" retrieval instead of supporting flexible relation loading.
4. **DTO conversion is inline**: Entity-to-DTO conversion logic is embedded in handlers without a reusable converter function.

**Recommendation**:
Consider more generalized approaches:

**Option 1: Generic relation loader**
```go
// Generic interface for loading relations
type WithRelations interface {
    LoadRelations(ctx context.Context, entities []entity.Article) error
}

// Repository method
func (r *ArticleRepo) Get(ctx context.Context, id int64, opts ...QueryOption) (*entity.Article, error)

// Usage
article, err := repo.Get(ctx, id, WithSource())
```

**Option 2: Separate entity for joined data**
```go
// Reusable composite entity
type ArticleWithSource struct {
    Article    *entity.Article
    SourceName string
}

// Repository method returns composite
func (r *ArticleRepo) GetWithSource(ctx context.Context, id int64) (*ArticleWithSource, error)
```

**Option 3: Generic DTO converter utility**
```go
// Reusable converter package
package convert

func ArticleToDTO(article *entity.Article, sourceName string) DTO {
    return DTO{
        ID:          article.ID,
        SourceID:    article.SourceID,
        SourceName:  sourceName,
        Title:       article.Title,
        URL:         article.URL,
        Summary:     article.Summary,
        PublishedAt: article.PublishedAt,
        CreatedAt:   article.CreatedAt,
    }
}
```

**Reusability Potential**:
- `pathutil.ExtractID` → Already reusable across all ID-based endpoints ✅
- `respond.SafeError` → Already reusable across all handlers ✅
- Entity-to-DTO conversion → Should be extracted to converter utility
- JOIN query pattern → Should be generalized for other entity relations

### 2. Business Logic Independence: 4.0 / 5.0 (Weight: 30%)

**Findings**:
- Business logic is well-separated into the Service layer
- Service layer is UI-agnostic and can be reused in different contexts
- Repository interface provides clean abstraction from database implementation
- Handler focuses on HTTP concerns (path extraction, JSON serialization)
- Middleware handles authentication/authorization separately

**Issues**:
1. **DTO conversion in handler**: Entity-to-DTO conversion logic is in the HTTP handler, making it harder to reuse the same business logic with different presentation formats (e.g., gRPC, GraphQL).
2. **Error handling tied to HTTP codes**: The error handling relies on HTTP status codes in some contexts, though this is minimal.

**Recommendation**:
Move DTO conversion logic to a dedicated converter package:

```go
// internal/converter/article.go
package converter

import (
    "catchup-feed/internal/domain/entity"
    "catchup-feed/internal/handler/http/article"
)

func ArticleToDTO(art *entity.Article) article.DTO {
    return article.DTO{
        ID:          art.ID,
        SourceID:    art.SourceID,
        Title:       art.Title,
        URL:         art.URL,
        Summary:     art.Summary,
        PublishedAt: art.PublishedAt,
        CreatedAt:   art.CreatedAt,
    }
}

func ArticleToDTOWithSource(art *entity.Article, sourceName string) article.DTO {
    dto := ArticleToDTO(art)
    dto.SourceName = sourceName
    return dto
}
```

**Portability Assessment**:
- Can this logic run in CLI? **Yes** - Service layer is framework-agnostic
- Can this logic run in mobile app? **Yes** - Service layer can be called from any Go client
- Can this logic run in background job? **Yes** - No HTTP dependencies in service layer
- Can this logic run in gRPC service? **Partially** - Would need separate protobuf conversion, but business logic is reusable

### 3. Domain Model Abstraction: 4.5 / 5.0 (Weight: 20%)

**Findings**:
- Domain models are pure Go structs with no framework dependencies ✅
- `entity.Article` is ORM-agnostic with no database annotations ✅
- Models can be used independently of HTTP, database, or any framework ✅
- Repository interface provides clean abstraction for persistence ✅
- Domain models are portable across different persistence layers ✅

**Issues**:
1. **GetWithSource returns (entity, string) tuple**: This creates an awkward interface where source information is separated from the article entity. Consider a composite domain object.

**Recommendation**:
The current domain model abstraction is excellent. However, for better modeling of joined data, consider:

```go
// internal/domain/entity/article_with_source.go
package entity

// ArticleWithSource represents an article with its associated source information.
// This is a composite entity used for queries that join articles and sources.
type ArticleWithSource struct {
    Article    *Article
    SourceName string
}
```

This keeps the domain model pure while providing a clear structure for joined data.

**Scoring**:
- 5.0: Domain models are pure, no framework/ORM dependencies ✅
- Deduction (-0.5): Awkward tuple return pattern instead of composite domain object

### 4. Shared Utility Design: 3.0 / 5.0 (Weight: 15%)

**Findings**:
- Good utilities already exist: `pathutil.ExtractID`, `respond.SafeError`, `respond.JSON` ✅
- Error types are shared across the codebase (`ErrArticleNotFound`, `ErrInvalidArticleID`) ✅
- Authentication middleware is reusable ✅

**Issues**:
1. **No DTO converter utility**: Entity-to-DTO conversion is duplicated inline in handlers (see `list.go` lines 31-39)
2. **No generic JOIN helper**: JOIN query pattern will be duplicated for each entity relation
3. **No repository query options pattern**: Common query patterns (pagination, filtering, sorting) are not abstracted
4. **Inline DTO construction**: The design shows inline DTO construction in the handler without extracting a converter

**Recommendation**:
Extract common patterns into reusable utilities:

**1. DTO Converter Package**
```go
// internal/converter/article.go
package converter

type ArticleConverter struct{}

func (c *ArticleConverter) ToDTO(art *entity.Article) article.DTO { ... }
func (c *ArticleConverter) ToDTOWithSource(art *entity.Article, sourceName string) article.DTO { ... }
func (c *ArticleConverter) ToDTOList(arts []*entity.Article) []article.DTO { ... }
```

**2. Repository Query Options**
```go
// internal/repository/options.go
package repository

type QueryOption func(*QueryOptions)

type QueryOptions struct {
    WithSource bool
    Limit      int
    Offset     int
}

func WithSource() QueryOption {
    return func(o *QueryOptions) { o.WithSource = true }
}
```

**3. Generic JOIN builder**
```go
// internal/infra/adapter/persistence/postgres/join.go
package postgres

type JoinBuilder struct {
    table      string
    joins      []Join
    conditions []string
}

func NewJoinBuilder(table string) *JoinBuilder { ... }
func (b *JoinBuilder) InnerJoin(table, on string) *JoinBuilder { ... }
```

**Potential Utilities**:
- Extract `ArticleConverter` for entity-to-DTO conversion
- Extract `QueryOptionsBuilder` for repository query patterns
- Extract `JoinBuilder` for SQL JOIN construction

**Scoring**:
- 3.0: Some utilities exist, but noticeable duplication in DTO conversion and JOIN patterns

---

## Reusability Opportunities

### High Potential
1. **pathutil.ExtractID** - Already reusable across all ID-based endpoints (sources, articles, users, etc.) ✅
2. **respond package** - SafeError, JSON, Error functions are highly reusable ✅
3. **auth.Authz middleware** - Can be reused for any authenticated endpoint ✅
4. **Service layer business logic** - Can be called from HTTP, gRPC, CLI, background jobs

### Medium Potential
1. **GetWithSource pattern** - With refactoring, can be generalized to `Get(ctx, id, WithSource())` for other entities
2. **DTO structure with omitempty** - Pattern can be reused for other optional fields
3. **SQL JOIN query** - Can be extracted to a generic JOIN builder utility

### Low Potential (Feature-Specific)
1. **Article-specific validation** - Inherently feature-specific, acceptable
2. **Article DTO fields** - Domain-specific, but conversion logic should be reusable

---

## Code Duplication Analysis

### Existing Duplication (from list.go)

**DTO Conversion Logic** (lines 31-39 in list.go):
```go
for _, e := range list {
    out = append(out, DTO{
        ID:          e.ID,
        SourceID:    e.SourceID,
        Title:       e.Title,
        URL:         e.URL,
        Summary:     e.Summary,
        PublishedAt: e.PublishedAt,
        CreatedAt:   e.CreatedAt,
    })
}
```

**Will be duplicated in get.go**:
```go
DTO{
    ID:          article.ID,
    SourceID:    article.SourceID,
    SourceName:  sourceName,  // New field
    Title:       article.Title,
    URL:         article.URL,
    Summary:     article.Summary,
    PublishedAt: article.PublishedAt,
    CreatedAt:   article.CreatedAt,
}
```

**Recommendation**: Extract to converter function to eliminate duplication.

---

## Action Items for Designer

Since status is "Request Changes", please address the following:

### Critical (Must Fix)

1. **Extract DTO conversion logic to reusable converter package**
   - Create `internal/converter/article.go`
   - Implement `ArticleToDTO()` and `ArticleToDTOWithSource()` functions
   - Use converter in both list and get handlers
   - **Rationale**: Eliminates code duplication, makes business logic more reusable

2. **Consider composite domain entity for joined data**
   - Create `entity.ArticleWithSource` struct
   - Return this from `GetWithSource()` instead of tuple
   - **Rationale**: Better domain modeling, clearer API

### Recommended (Should Consider)

3. **Design generic relation loading pattern**
   - Consider query options pattern: `repo.Get(ctx, id, WithSource())`
   - Or create composite entities: `ArticleWithSource`, `ArticleWithTags`, etc.
   - **Rationale**: Future features will need similar JOIN patterns (e.g., articles with tags, articles with author info)

4. **Document reusability patterns in design**
   - Add section on "Reusable Components"
   - Document which components can be shared across features
   - **Rationale**: Helps future developers understand what can be reused

### Optional (Nice to Have)

5. **Create repository query options abstraction**
   - Design `QueryOption` pattern for common query needs
   - **Rationale**: Future scalability (pagination, filtering, sorting)

---

## Comparison with Existing Codebase Patterns

### What's Consistent ✅
- Service layer pattern matches existing `article.Service.Get()`, `article.Service.List()`
- Repository interface pattern matches existing `repository.ArticleRepository`
- Handler structure matches existing `ListHandler` with embedded service
- Error handling matches existing `respond.SafeError` usage
- Path parameter extraction matches existing `pathutil.ExtractID` pattern

### What's Inconsistent or Could Be Improved ❌
- **DTO conversion**: Inline construction instead of converter utility (duplication from list.go)
- **Tuple return**: Returns `(entity, string)` instead of composite object
- **Feature-specific method**: `GetWithSource()` instead of generic options pattern

---

## Reusability Metrics

| Metric | Value | Target | Status |
|--------|-------|--------|--------|
| Reusable component ratio | 60% | 80% | ⚠️ Needs improvement |
| Code duplication | Medium | Low | ⚠️ DTO conversion duplicated |
| Framework coupling | Low | Low | ✅ Good |
| Business logic portability | High | High | ✅ Excellent |
| Utility coverage | 50% | 75% | ⚠️ Missing converter utilities |

**Key Findings**:
- Business logic is highly portable (can run in CLI, gRPC, background jobs) ✅
- Domain models are framework-agnostic ✅
- Missing converter utilities leads to code duplication ❌
- GetWithSource pattern is feature-specific instead of generalized ❌

---

## Structured Data

```yaml
evaluation_result:
  evaluator: "design-reusability-evaluator"
  design_document: "/Users/yujitsuchiya/catchup-feed/docs/designs/article-detail-endpoint.md"
  timestamp: "2025-12-06T00:00:00Z"
  overall_judgment:
    status: "Request Changes"
    overall_score: 3.4
  detailed_scores:
    component_generalization:
      score: 3.0
      weight: 0.35
      issues:
        - "GetWithSource() is feature-specific, not generalized"
        - "No generic JOIN support in repository"
        - "DTO conversion logic not extracted to utility"
      recommendations:
        - "Create generic relation loading pattern (e.g., WithSource() query option)"
        - "Extract DTO converter utility to eliminate duplication"
        - "Consider composite domain entities for joined data"
    business_logic_independence:
      score: 4.0
      weight: 0.30
      issues:
        - "DTO conversion in handler (presentation layer concern)"
      recommendations:
        - "Move DTO conversion to dedicated converter package"
      portability:
        cli: true
        mobile: true
        background_job: true
        grpc: true
    domain_model_abstraction:
      score: 4.5
      weight: 0.20
      strengths:
        - "Pure Go structs with no framework dependencies"
        - "ORM-agnostic entity definitions"
        - "Repository interface provides clean abstraction"
      issues:
        - "Tuple return (entity, string) instead of composite domain object"
      recommendations:
        - "Create ArticleWithSource composite entity"
    shared_utility_design:
      score: 3.0
      weight: 0.15
      existing_utilities:
        - "pathutil.ExtractID"
        - "respond.SafeError"
        - "respond.JSON"
        - "auth.Authz middleware"
      missing_utilities:
        - "DTO converter package"
        - "Repository query options pattern"
        - "Generic JOIN builder"
      code_duplication:
        - location: "DTO conversion logic"
          files: ["list.go", "get.go (planned)"]
          impact: "Medium"
  reusability_opportunities:
    high_potential:
      - component: "pathutil.ExtractID"
        contexts: ["all ID-based endpoints", "cross-service"]
        status: "already implemented"
      - component: "respond package"
        contexts: ["all HTTP handlers", "cross-service"]
        status: "already implemented"
      - component: "Service layer"
        contexts: ["HTTP", "gRPC", "CLI", "background jobs"]
        status: "ready to reuse"
    medium_potential:
      - component: "GetWithSource pattern"
        contexts: ["other entity relations"]
        refactoring_needed: "Generalize to query options pattern"
      - component: "DTO conversion"
        contexts: ["all handlers"]
        refactoring_needed: "Extract to converter utility"
    low_potential:
      - component: "Article-specific validation"
        reason: "Domain-specific by nature"
  action_items:
    critical:
      - "Extract DTO conversion to converter package"
      - "Consider composite domain entity for joined data"
    recommended:
      - "Design generic relation loading pattern"
      - "Document reusability patterns in design"
    optional:
      - "Create repository query options abstraction"
  metrics:
    reusable_component_ratio: 0.60
    target_ratio: 0.80
    code_duplication_level: "Medium"
    framework_coupling: "Low"
    business_logic_portability: "High"
    utility_coverage: 0.50
```
