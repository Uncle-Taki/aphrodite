package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"aphrodite/internal/content"
)

type Handler struct {
	reader         content.SlugReader
	defaultLocale  string
	fallbackLocale string
}

func NewHandler(reader content.SlugReader, defaultLocale, fallbackLocale string) *Handler {
	if strings.TrimSpace(defaultLocale) == "" {
		defaultLocale = "fa"
	}
	if strings.TrimSpace(fallbackLocale) == "" {
		fallbackLocale = defaultLocale
	}
	return &Handler{reader: reader, defaultLocale: defaultLocale, fallbackLocale: fallbackLocale}
}

func Register(rg *gin.RouterGroup, h *Handler) {
	contentGroup := rg.Group("")
	contentGroup.GET("/articles/:slug", h.Article)
	contentGroup.GET("/recipes/:slug", h.Recipe)
	contentGroup.GET("/categories/:slug", h.MainCategory)
	contentGroup.GET("/products/:slug", h.Product)
	contentGroup.GET("/newsletters/:slug", h.Newsletter)
}

// Article returns a published localized article by slug.
// @Summary Get an article
// @Tags content
// @Produce json
// @Param slug path string true "Article slug"
// @Param locale query string false "Locale (defaults to configured Persian locale)"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /v1/articles/{slug} [get]
func (h *Handler) Article(c *gin.Context) {
	h.getBySlug(c, "articles", []string{"author", "mainCategory", "subCategories", "products", "affiliateLinks", "coverImage", "gallery", "seo"})
}

// Recipe returns a published localized recipe by slug.
// @Summary Get a recipe
// @Tags content
// @Produce json
// @Param slug path string true "Recipe slug"
// @Param locale query string false "Locale"
// @Success 200 {object} map[string]interface{}
// @Router /v1/recipes/{slug} [get]
func (h *Handler) Recipe(c *gin.Context) {
	h.getBySlug(c, "recipes", []string{"author", "mainCategory", "subCategories", "products", "affiliateLinks", "heroImage", "seo"})
}

// MainCategory returns a published main category by slug.
// @Summary Get a category
// @Tags content
// @Produce json
// @Param slug path string true "Category slug"
// @Param locale query string false "Locale"
// @Success 200 {object} map[string]interface{}
// @Router /v1/categories/{slug} [get]
func (h *Handler) MainCategory(c *gin.Context) {
	h.getBySlug(c, "main-categories", []string{"subCategories", "articles", "recipes", "image"})
}

// Product returns a published product by slug.
// @Summary Get a product
// @Tags content
// @Produce json
// @Param slug path string true "Product slug"
// @Param locale query string false "Locale"
// @Success 200 {object} map[string]interface{}
// @Router /v1/products/{slug} [get]
func (h *Handler) Product(c *gin.Context) {
	h.getBySlug(c, "products", []string{"brand", "affiliateLinks", "articles", "recipes", "image"})
}

// Newsletter returns a published newsletter by slug.
// @Summary Get a newsletter
// @Tags content
// @Produce json
// @Param slug path string true "Newsletter slug"
// @Param locale query string false "Locale"
// @Success 200 {object} map[string]interface{}
// @Router /v1/newsletters/{slug} [get]
func (h *Handler) Newsletter(c *gin.Context) {
	h.getBySlug(c, "newsletters", []string{"heroImage"})
}

func (h *Handler) getBySlug(c *gin.Context, resource string, populate []string) {
	locale := c.Query("locale")
	if locale == "" {
		locale = h.defaultLocale
	}
	query := content.Query{Locale: locale, Status: "published", Populate: populate}
	doc, err := h.reader.FindBySlug(c.Request.Context(), resource, c.Param("slug"), query)
	if errors.Is(err, content.ErrNotFound) && locale != h.fallbackLocale {
		query.Locale = h.fallbackLocale
		doc, err = h.reader.FindBySlug(c.Request.Context(), resource, c.Param("slug"), query)
	}
	if err != nil {
		writeContentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":          doc.ID,
		"document_id": doc.DocumentID,
		"locale":      query.Locale,
		"data":        doc.Fields,
		"meta":        doc.Meta,
	})
}

func writeContentError(c *gin.Context, err error) {
	status := http.StatusBadGateway
	switch {
	case errors.Is(err, content.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, content.ErrUnauthorized), errors.Is(err, content.ErrForbidden):
		status = http.StatusBadGateway
	case errors.Is(err, content.ErrRateLimited), errors.Is(err, content.ErrUnavailable):
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"error": "content service unavailable"})
}
