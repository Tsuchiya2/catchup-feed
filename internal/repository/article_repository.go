package repository

import (
	"context"

	"catchup-feed/internal/domain/entity"
)

// ArticleWithSource represents an article along with its source name.
type ArticleWithSource struct {
	Article    *entity.Article
	SourceName string
}

type ArticleRepository interface {
	List(ctx context.Context) ([]*entity.Article, error)
	// ListWithSource retrieves all articles with their source names.
	// Returns a slice of ArticleWithSource containing article and source name pairs.
	ListWithSource(ctx context.Context) ([]ArticleWithSource, error)
	Get(ctx context.Context, id int64) (*entity.Article, error)
	// GetWithSource retrieves an article by ID and includes the source name.
	// Returns the article entity, source name, and error.
	// Returns (nil, "", nil) if the article is not found.
	GetWithSource(ctx context.Context, id int64) (*entity.Article, string, error)
	Search(ctx context.Context, keyword string) ([]*entity.Article, error)
	Create(ctx context.Context, article *entity.Article) error
	Update(ctx context.Context, article *entity.Article) error
	Delete(ctx context.Context, id int64) error
	ExistsByURL(ctx context.Context, url string) (bool, error)
	// ExistsByURLBatch はバッチでURL存在チェックを行い、N+1問題を解消する
	ExistsByURLBatch(ctx context.Context, urls []string) (map[string]bool, error)
}
