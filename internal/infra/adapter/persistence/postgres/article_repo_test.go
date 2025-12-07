package postgres_test

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"

	"catchup-feed/internal/domain/entity"
	pg "catchup-feed/internal/infra/adapter/persistence/postgres"
)

/* ─────────────────────────── ヘルパ ─────────────────────────── */

func artRow(a *entity.Article) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "source_id", "title", "url",
		"summary", "published_at", "created_at",
	}).AddRow(
		a.ID, a.SourceID, a.Title, a.URL,
		a.Summary, a.PublishedAt, a.CreatedAt,
	)
}

/* ─────────────────────────── 1. Get ─────────────────────────── */

func TestArticleRepo_Get(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	now := time.Date(2025, 7, 19, 0, 0, 0, 0, time.UTC)
	want := &entity.Article{
		ID: 1, SourceID: 2, Title: "Go 1.24 released",
		URL: "https://example.com", Summary: "sum",
		PublishedAt: now, CreatedAt: now,
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id")).
		WithArgs(int64(1)).
		WillReturnRows(artRow(want))

	repo := pg.NewArticleRepo(db)
	got, err := repo.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("Get err=%v", err)
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(entity.Article{})); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

/* ─────────────────────────── 2. List ─────────────────────────── */

func TestArticleRepo_List(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	now := time.Now()
	mock.ExpectQuery("FROM articles").
		WillReturnRows(artRow(&entity.Article{
			ID: 1, SourceID: 2, Title: "x", URL: "y",
			Summary: "s", PublishedAt: now, CreatedAt: now,
		}))

	repo := pg.NewArticleRepo(db)
	got, err := repo.List(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("List err=%v len=%d", err, len(got))
	}
}

/* ─────────────────────────── 3. Search ─────────────────────────── */

func TestArticleRepo_Search(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("FROM articles").
		WithArgs("%go%").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "source_id", "title", "url",
			"summary", "published_at", "created_at",
		})) // 空集合で OK

	repo := pg.NewArticleRepo(db)
	if _, err := repo.Search(context.Background(), "go"); err != nil {
		t.Fatalf("Search err=%v", err)
	}
}

/* ─────────────────────────── 4. Create ─────────────────────────── */

func TestArticleRepo_Create(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	now := time.Now()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO articles")).
		WithArgs(int64(2), "title", "https://u",
			"summary", now, now).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := pg.NewArticleRepo(db)
	err := repo.Create(context.Background(), &entity.Article{
		SourceID: 2, Title: "title", URL: "https://u",
		Summary: "summary", PublishedAt: now, CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create err=%v", err)
	}
}

/* ─────────────────────────── 5. Update ─────────────────────────── */

func TestArticleRepo_Update(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	now := time.Now()

	mock.ExpectExec("UPDATE articles").
		WithArgs(int64(2), "new", "https://u",
			"sum", now, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := pg.NewArticleRepo(db)
	err := repo.Update(context.Background(), &entity.Article{
		ID: 1, SourceID: 2, Title: "new", URL: "https://u",
		Summary: "sum", PublishedAt: now,
	})
	if err != nil {
		t.Fatalf("Update err=%v", err)
	}
}

/* ─────────────────────────── 6. Delete ─────────────────────────── */

func TestArticleRepo_Delete(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM articles").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := pg.NewArticleRepo(db)
	if err := repo.Delete(context.Background(), 1); err != nil {
		t.Fatalf("Delete err=%v", err)
	}
}

/* ─────────────────────────── 7. ExistsByURL ─────────────────────────── */

func TestArticleRepo_ExistsByURL(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	// PostgreSQLはSELECT EXISTSを使用し、常に1行返す（trueまたはfalse）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS (SELECT 1 FROM articles WHERE url = $1)")).
		WithArgs("https://u").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	repo := pg.NewArticleRepo(db)
	ok, err := repo.ExistsByURL(context.Background(), "https://u")
	if err != nil || !ok {
		t.Fatalf("ExistsByURL err=%v ok=%v", err, ok)
	}
}

func TestArticleRepo_ExistsByURL_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	// PostgreSQLはSELECT EXISTSを使用し、常に1行返す（falseの場合）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS (SELECT 1 FROM articles WHERE url = $1)")).
		WithArgs("https://notfound").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	repo := pg.NewArticleRepo(db)
	ok, err := repo.ExistsByURL(context.Background(), "https://notfound")
	if err != nil {
		t.Fatalf("ExistsByURL err=%v", err)
	}
	if ok {
		t.Fatalf("ExistsByURL want false, got true")
	}
}

/* ─────────────────────────── 8. ExistsByURLBatch ─────────────────────────── */

func TestArticleRepo_ExistsByURLBatch(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	urls := []string{
		"https://example.com/article1",
		"https://example.com/article2",
		"https://example.com/article3",
	}

	// article1とarticle3が存在する
	mock.ExpectQuery(regexp.QuoteMeta("SELECT url FROM articles WHERE url = ANY($1)")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).
			AddRow("https://example.com/article1").
			AddRow("https://example.com/article3"))

	repo := pg.NewArticleRepo(db)
	result, err := repo.ExistsByURLBatch(context.Background(), urls)
	if err != nil {
		t.Fatalf("ExistsByURLBatch err=%v", err)
	}

	// 結果を検証
	if len(result) != 2 {
		t.Fatalf("result length = %d, want 2", len(result))
	}
	if !result["https://example.com/article1"] {
		t.Errorf("article1 should exist")
	}
	if result["https://example.com/article2"] {
		t.Errorf("article2 should not exist")
	}
	if !result["https://example.com/article3"] {
		t.Errorf("article3 should exist")
	}
}

func TestArticleRepo_ExistsByURLBatch_Empty(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	repo := pg.NewArticleRepo(db)
	result, err := repo.ExistsByURLBatch(context.Background(), []string{})
	if err != nil {
		t.Fatalf("ExistsByURLBatch err=%v", err)
	}

	// 空のURLリストは空の結果を返す
	if len(result) != 0 {
		t.Fatalf("result length = %d, want 0", len(result))
	}
}

func TestArticleRepo_ExistsByURLBatch_AllNew(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	urls := []string{
		"https://example.com/new1",
		"https://example.com/new2",
	}

	// すべて存在しない（空の結果）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT url FROM articles WHERE url = ANY($1)")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"url"}))

	repo := pg.NewArticleRepo(db)
	result, err := repo.ExistsByURLBatch(context.Background(), urls)
	if err != nil {
		t.Fatalf("ExistsByURLBatch err=%v", err)
	}

	// すべて存在しないので、結果は空
	if len(result) != 0 {
		t.Fatalf("result length = %d, want 0", len(result))
	}
}

func TestArticleRepo_ExistsByURLBatch_AllExist(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	urls := []string{
		"https://example.com/article1",
		"https://example.com/article2",
	}

	// すべて存在する
	mock.ExpectQuery(regexp.QuoteMeta("SELECT url FROM articles WHERE url = ANY($1)")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).
			AddRow("https://example.com/article1").
			AddRow("https://example.com/article2"))

	repo := pg.NewArticleRepo(db)
	result, err := repo.ExistsByURLBatch(context.Background(), urls)
	if err != nil {
		t.Fatalf("ExistsByURLBatch err=%v", err)
	}

	// すべて存在する
	if len(result) != 2 {
		t.Fatalf("result length = %d, want 2", len(result))
	}
	if !result["https://example.com/article1"] {
		t.Errorf("article1 should exist")
	}
	if !result["https://example.com/article2"] {
		t.Errorf("article2 should exist")
	}
}

/* ─────────────────────────── 9. GetWithSource ─────────────────────────── */

func TestArticleRepo_GetWithSource_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	now := time.Date(2025, 7, 19, 0, 0, 0, 0, time.UTC)
	want := &entity.Article{
		ID:          1,
		SourceID:    2,
		Title:       "Go 1.24 released",
		URL:         "https://example.com",
		Summary:     "sum",
		PublishedAt: now,
		CreatedAt:   now,
	}
	wantSourceName := "Tech News"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT a.id, a.source_id, a.title, a.url, a.summary, a.published_at, a.created_at, s.name AS source_name")).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "source_id", "title", "url",
			"summary", "published_at", "created_at", "source_name",
		}).AddRow(
			want.ID, want.SourceID, want.Title, want.URL,
			want.Summary, want.PublishedAt, want.CreatedAt, wantSourceName,
		))

	repo := pg.NewArticleRepo(db)
	got, sourceName, err := repo.GetWithSource(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetWithSource err=%v", err)
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(entity.Article{})); diff != "" {
		t.Fatalf("article mismatch (-want +got):\n%s", diff)
	}
	if sourceName != wantSourceName {
		t.Errorf("sourceName = %q, want %q", sourceName, wantSourceName)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestArticleRepo_GetWithSource_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT a.id")).
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "source_id", "title", "url",
			"summary", "published_at", "created_at", "source_name",
		}))

	repo := pg.NewArticleRepo(db)
	got, sourceName, err := repo.GetWithSource(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetWithSource should not return error for not found, err=%v", err)
	}
	if got != nil {
		t.Errorf("GetWithSource should return nil article for not found, got=%v", got)
	}
	if sourceName != "" {
		t.Errorf("GetWithSource should return empty source name for not found, got=%q", sourceName)
	}
}

func TestArticleRepo_GetWithSource_DatabaseError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	dbError := errors.New("connection lost")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT a.id")).
		WithArgs(int64(1)).
		WillReturnError(dbError)

	repo := pg.NewArticleRepo(db)
	got, sourceName, err := repo.GetWithSource(context.Background(), 1)
	if err == nil {
		t.Fatalf("GetWithSource should return error for database error")
	}
	if got != nil {
		t.Errorf("GetWithSource should return nil article on error, got=%v", got)
	}
	if sourceName != "" {
		t.Errorf("GetWithSource should return empty source name on error, got=%q", sourceName)
	}
}

func TestArticleRepo_GetWithSource_JoinWithSourceName(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer func() { _ = db.Close() }()

	now := time.Now()
	tests := []struct {
		name           string
		articleID      int64
		sourceName     string
		wantSourceName string
	}{
		{
			name:           "source with simple name",
			articleID:      1,
			sourceName:     "TechCrunch",
			wantSourceName: "TechCrunch",
		},
		{
			name:           "source with space in name",
			articleID:      2,
			sourceName:     "Hacker News",
			wantSourceName: "Hacker News",
		},
		{
			name:           "source with special characters",
			articleID:      3,
			sourceName:     "Dev.to - Community",
			wantSourceName: "Dev.to - Community",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT a.id")).
				WithArgs(tt.articleID).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "source_id", "title", "url",
					"summary", "published_at", "created_at", "source_name",
				}).AddRow(
					tt.articleID, int64(10), "Test Title", "https://example.com",
					"Test Summary", now, now, tt.sourceName,
				))

			repo := pg.NewArticleRepo(db)
			_, sourceName, err := repo.GetWithSource(context.Background(), tt.articleID)
			if err != nil {
				t.Fatalf("GetWithSource err=%v", err)
			}
			if sourceName != tt.wantSourceName {
				t.Errorf("sourceName = %q, want %q", sourceName, tt.wantSourceName)
			}
		})
	}
}
