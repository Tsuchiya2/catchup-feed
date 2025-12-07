package article

import (
	"net/http"

	"catchup-feed/internal/handler/http/respond"
	artUC "catchup-feed/internal/usecase/article"
)

type ListHandler struct{ Svc artUC.Service }

// ServeHTTP 記事一覧取得
// @Summary      記事一覧取得
// @Description  登録されているすべての記事を取得します
// @Tags         articles
// @Security     BearerAuth
// @Produce      json
// @Success      200 {array} DTO "記事一覧"
// @Failure      401 {string} string "Authentication required - missing or invalid JWT token"
// @Failure      403 {string} string "Forbidden - insufficient permissions"
// @Failure      500 {string} string "サーバーエラー"
// @Router       /articles [get]
func (h ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	list, err := h.Svc.ListWithSource(r.Context())
	if err != nil {
		respond.SafeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]DTO, 0, len(list))
	for _, item := range list {
		out = append(out, DTO{
			ID:          item.Article.ID,
			SourceID:    item.Article.SourceID,
			SourceName:  item.SourceName,
			Title:       item.Article.Title,
			URL:         item.Article.URL,
			Summary:     item.Article.Summary,
			PublishedAt: item.Article.PublishedAt,
			CreatedAt:   item.Article.CreatedAt,
		})
	}
	respond.JSON(w, http.StatusOK, out)
}
