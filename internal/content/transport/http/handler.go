package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"aphrodite/internal/content"
)

// ListConfig bounds public content list pagination.
type ListConfig struct {
	DefaultLimit int
	MaxLimit     int
}

type Handler struct {
	reader         content.SlugReader
	writer         content.DocumentWriter
	defaultLocale  string
	fallbackLocale string
	listCfg        ListConfig
}

func NewHandler(reader content.SlugReader, defaultLocale, fallbackLocale string) *Handler {
	return NewHandlerWithList(reader, nil, defaultLocale, fallbackLocale, ListConfig{})
}

func NewHandlerWithList(reader content.SlugReader, writer content.DocumentWriter, defaultLocale, fallbackLocale string, list ListConfig) *Handler {
	if strings.TrimSpace(defaultLocale) == "" {
		defaultLocale = "fa"
	}
	if strings.TrimSpace(fallbackLocale) == "" {
		fallbackLocale = defaultLocale
	}
	if list.DefaultLimit <= 0 {
		list.DefaultLimit = 15
	}
	if list.MaxLimit <= 0 || list.MaxLimit < list.DefaultLimit {
		list.MaxLimit = list.DefaultLimit
	}
	return &Handler{reader: reader, writer: writer, defaultLocale: defaultLocale, fallbackLocale: fallbackLocale, listCfg: list}
}

// resourceSpec describes one Strapi-backed content type exposed by the BFF.
type resourceSpec struct {
	resource string
	populate []string
	// filterable lists relations clients may filter by the related slug.
	filterable []string
}

var (
	articleSpec = resourceSpec{
		resource:   "articles",
		populate:   []string{"author", "mainCategory", "subCategories", "products", "affiliateLinks", "coverImage", "gallery", "seo"},
		filterable: []string{"mainCategory", "author"},
	}
	recipeSpec = resourceSpec{
		resource:   "recipes",
		populate:   []string{"author", "mainCategory", "subCategories", "products", "affiliateLinks", "heroImage", "seo"},
		filterable: []string{"mainCategory", "author"},
	}
	mainCategorySpec = resourceSpec{
		resource: "main-categories",
		populate: []string{"subCategories", "articles", "recipes", "image"},
	}
	productSpec = resourceSpec{
		resource:   "products",
		populate:   []string{"brand", "affiliateLinks", "articles", "recipes", "image"},
		filterable: []string{"brand"},
	}
	newsletterSpec = resourceSpec{
		resource: "newsletters",
		populate: []string{"heroImage"},
	}
)

func Register(rg *gin.RouterGroup, h *Handler) {
	contentGroup := rg.Group("")
	contentGroup.GET("/articles/:slug", h.Article)
	contentGroup.GET("/recipes/:slug", h.Recipe)
	contentGroup.GET("/categories/:slug", h.MainCategory)
	contentGroup.GET("/products/:slug", h.Product)
	contentGroup.GET("/newsletters/:slug", h.Newsletter)
	contentGroup.GET("/articles", h.ListArticles)
	contentGroup.GET("/recipes", h.ListRecipes)
	contentGroup.GET("/categories", h.ListMainCategories)
	contentGroup.GET("/products", h.ListProducts)
	contentGroup.GET("/newsletters", h.ListNewsletters)
}

// RegisterRoutes wires public read routes and bearer-authenticated write
// routes. Admins may create, update, and delete; editors may only update.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware, adminRole, editorRole gin.HandlerFunc) {
	Register(rg, h)
	if authMiddleware == nil || adminRole == nil || editorRole == nil || h.writer == nil {
		return
	}
	adminWrites := rg.Group("")
	adminWrites.Use(authMiddleware, adminRole)
	adminWrites.POST("/articles", h.CreateArticle)
	adminWrites.POST("/recipes", h.CreateRecipe)
	adminWrites.POST("/categories", h.CreateMainCategory)
	adminWrites.POST("/products", h.CreateProduct)
	adminWrites.POST("/newsletters", h.CreateNewsletter)
	adminWrites.DELETE("/articles/:id", h.DeleteArticle)
	adminWrites.DELETE("/recipes/:id", h.DeleteRecipe)
	adminWrites.DELETE("/categories/:id", h.DeleteMainCategory)
	adminWrites.DELETE("/products/:id", h.DeleteProduct)
	adminWrites.DELETE("/newsletters/:id", h.DeleteNewsletter)

	updateWrites := rg.Group("")
	updateWrites.Use(authMiddleware, editorRole)
	updateWrites.PUT("/articles/:id", h.UpdateArticle)
	updateWrites.PUT("/recipes/:id", h.UpdateRecipe)
	updateWrites.PUT("/categories/:id", h.UpdateMainCategory)
	updateWrites.PUT("/products/:id", h.UpdateProduct)
	updateWrites.PUT("/newsletters/:id", h.UpdateNewsletter)
}

// Article returns a published localized article by slug.
// @Summary Get an article
// @Tags content
// @Produce json
// @Param slug path string true "Article slug"
// @Param locale query string false "Locale (defaults to configured Persian locale)"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/articles/{slug} [get]
func (h *Handler) Article(c *gin.Context) {
	h.getBySlug(c, articleSpec.resource, articleSpec.populate)
}

// Recipe returns a published localized recipe by slug.
// @Summary Get a recipe
// @Tags content
// @Produce json
// @Param slug path string true "Recipe slug"
// @Param locale query string false "Locale"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/recipes/{slug} [get]
func (h *Handler) Recipe(c *gin.Context) {
	h.getBySlug(c, recipeSpec.resource, recipeSpec.populate)
}

// MainCategory returns a published main category by slug.
// @Summary Get a category
// @Tags content
// @Produce json
// @Param slug path string true "Category slug"
// @Param locale query string false "Locale"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/categories/{slug} [get]
func (h *Handler) MainCategory(c *gin.Context) {
	h.getBySlug(c, mainCategorySpec.resource, mainCategorySpec.populate)
}

// Product returns a published product by slug.
// @Summary Get a product
// @Tags content
// @Produce json
// @Param slug path string true "Product slug"
// @Param locale query string false "Locale"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/products/{slug} [get]
func (h *Handler) Product(c *gin.Context) {
	h.getBySlug(c, productSpec.resource, productSpec.populate)
}

// Newsletter returns a published newsletter by slug.
// @Summary Get a newsletter
// @Tags content
// @Produce json
// @Param slug path string true "Newsletter slug"
// @Param locale query string false "Locale"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/newsletters/{slug} [get]
func (h *Handler) Newsletter(c *gin.Context) {
	h.getBySlug(c, newsletterSpec.resource, newsletterSpec.populate)
}

// ListArticles returns published localized articles.
//
// @Summary List articles (paginated)
// @Tags content
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param pageSize query int false "Page size (defaults to CONTENT_DEFAULT_LIMIT, capped by CONTENT_MAX_LIMIT)"
// @Param locale query string false "Locale (defaults to configured Persian locale)"
// @Param sort query string false "Sort field, e.g. title:asc or createdAt:desc (repeatable)"
// @Param mainCategory query string false "Filter by main category slug"
// @Param author query string false "Filter by author slug"
// @Success 200 {object} ContentListResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/articles [get]
func (h *Handler) ListArticles(c *gin.Context) {
	h.list(c, articleSpec)
}

// ListRecipes returns published localized recipes.
//
// @Summary List recipes (paginated)
// @Tags content
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param pageSize query int false "Page size (defaults to CONTENT_DEFAULT_LIMIT, capped by CONTENT_MAX_LIMIT)"
// @Param locale query string false "Locale (defaults to configured Persian locale)"
// @Param sort query string false "Sort field, e.g. title:asc or createdAt:desc (repeatable)"
// @Param mainCategory query string false "Filter by main category slug"
// @Param author query string false "Filter by author slug"
// @Success 200 {object} ContentListResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/recipes [get]
func (h *Handler) ListRecipes(c *gin.Context) {
	h.list(c, recipeSpec)
}

// ListMainCategories returns published main categories.
//
// @Summary List categories (paginated)
// @Tags content
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param pageSize query int false "Page size (defaults to CONTENT_DEFAULT_LIMIT, capped by CONTENT_MAX_LIMIT)"
// @Param locale query string false "Locale (defaults to configured Persian locale)"
// @Param sort query string false "Sort field, e.g. name:asc (repeatable)"
// @Success 200 {object} ContentListResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/categories [get]
func (h *Handler) ListMainCategories(c *gin.Context) {
	h.list(c, mainCategorySpec)
}

// ListProducts returns published products.
//
// @Summary List products (paginated)
// @Tags content
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param pageSize query int false "Page size (defaults to CONTENT_DEFAULT_LIMIT, capped by CONTENT_MAX_LIMIT)"
// @Param locale query string false "Locale (defaults to configured Persian locale)"
// @Param sort query string false "Sort field, e.g. name:asc (repeatable)"
// @Param brand query string false "Filter by brand slug"
// @Success 200 {object} ContentListResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/products [get]
func (h *Handler) ListProducts(c *gin.Context) {
	h.list(c, productSpec)
}

// ListNewsletters returns published newsletters.
//
// @Summary List newsletters (paginated)
// @Tags content
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param pageSize query int false "Page size (defaults to CONTENT_DEFAULT_LIMIT, capped by CONTENT_MAX_LIMIT)"
// @Param locale query string false "Locale (defaults to configured Persian locale)"
// @Param sort query string false "Sort field, e.g. subject:asc (repeatable)"
// @Success 200 {object} ContentListResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/newsletters [get]
func (h *Handler) ListNewsletters(c *gin.Context) {
	h.list(c, newsletterSpec)
}

func (h *Handler) list(c *gin.Context, spec resourceSpec) {
	locale := c.Query("locale")
	if locale == "" {
		locale = h.defaultLocale
	}
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize < 1 {
		pageSize = h.listCfg.DefaultLimit
	}
	if pageSize > h.listCfg.MaxLimit {
		pageSize = h.listCfg.MaxLimit
	}

	query := content.Query{
		Locale:   locale,
		Status:   "published",
		Page:     page,
		PageSize: pageSize,
		Populate: spec.populate,
	}
	if sorts := c.QueryArray("sort"); len(sorts) > 0 {
		query.Sort = sorts
	}

	if len(spec.filterable) > 0 {
		filters := map[string]string{}
		for _, field := range spec.filterable {
			if value := strings.TrimSpace(c.Query(field)); value != "" {
				filters[field+".slug"] = value
			}
		}
		if len(filters) > 0 {
			query.Filters = filters
		}
	}

	result, err := h.reader.List(c.Request.Context(), spec.resource, query)
	if err != nil {
		writeContentError(c, err)
		return
	}

	items := make([]interface{}, 0, len(result.Items))
	for _, doc := range result.Items {
		items = append(items, gin.H{
			"id":          doc.ID,
			"document_id": doc.DocumentID,
			"locale":      locale,
			"data":        doc.Fields,
			"meta":        doc.Meta,
		})
	}
	c.JSON(http.StatusOK, ContentListResponse{
		Items:     items,
		Total:     result.Total,
		Limit:     pageSize,
		Page:      page,
		PageCount: result.PageCount,
	})
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
	message := "content service unavailable"
	switch {
	case errors.Is(err, content.ErrNotFound):
		status = http.StatusNotFound
		message = "content not found"
	case errors.Is(err, content.ErrInvalidRequest):
		// Strapi rejected the query; details are in server logs only.
		status = http.StatusBadGateway
	case errors.Is(err, content.ErrUnauthorized), errors.Is(err, content.ErrForbidden):
		status = http.StatusBadGateway
	case errors.Is(err, content.ErrRateLimited), errors.Is(err, content.ErrUnavailable):
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"error": message})
}

// ── Write handlers (CMS-gated) ──────────────────────────────────────────────

func writeContentWriteError(c *gin.Context, err error) {
	if errors.Is(err, content.ErrInvalidRequest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content payload"})
		return
	}
	writeContentError(c, err)
}

// writePayload is the passthrough editorial payload forwarded to Strapi.
type writePayload map[string]any

func (h *Handler) create(c *gin.Context, resource string) {
	var payload writePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	query := content.Query{Locale: h.writeLocale(c)}
	doc, err := h.writer.Create(c.Request.Context(), resource, payload, query)
	if err != nil {
		writeContentWriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, documentResponse(doc, query.Locale))
}

func (h *Handler) update(c *gin.Context, resource string) {
	var payload writePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	query := content.Query{Locale: h.writeLocale(c)}
	doc, err := h.writer.Update(c.Request.Context(), resource, c.Param("id"), payload, query)
	if err != nil {
		writeContentWriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, documentResponse(doc, query.Locale))
}

func (h *Handler) remove(c *gin.Context, resource string) {
	query := content.Query{Locale: h.writeLocale(c)}
	if err := h.writer.Delete(c.Request.Context(), resource, c.Param("id"), query); err != nil {
		writeContentError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) writeLocale(c *gin.Context) string {
	locale := c.Query("locale")
	if locale == "" {
		locale = h.defaultLocale
	}
	return locale
}

func documentResponse(doc content.Document, locale string) gin.H {
	return gin.H{
		"id":          doc.ID,
		"document_id": doc.DocumentID,
		"locale":      locale,
		"data":        doc.Fields,
		"meta":        doc.Meta,
	}
}

// CreateArticle stores a new article in Strapi.
//
// @Summary Create an article (admin only)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param locale query string false "Locale for the new entry (defaults to configured Persian locale)"
// @Param request body object true "Strapi article attributes (name, slug, content, relations)"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/articles [post]
func (h *Handler) CreateArticle(c *gin.Context) {
	h.create(c, articleSpec.resource)
}

// UpdateArticle modifies an existing article in Strapi.
//
// @Summary Update an article (admin or editor)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Param locale query string false "Locale of the entry to update"
// @Param request body object true "Strapi article attributes to change"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/articles/{id} [put]
func (h *Handler) UpdateArticle(c *gin.Context) {
	h.update(c, articleSpec.resource)
}

// DeleteArticle removes an article from Strapi.
//
// @Summary Delete an article (admin only)
// @Tags content
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/articles/{id} [delete]
func (h *Handler) DeleteArticle(c *gin.Context) {
	h.remove(c, articleSpec.resource)
}

// CreateRecipe stores a new recipe in Strapi.
//
// @Summary Create a recipe (admin only)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param locale query string false "Locale for the new entry (defaults to configured Persian locale)"
// @Param request body object true "Strapi recipe attributes (title, slug, ingredients, steps, relations)"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/recipes [post]
func (h *Handler) CreateRecipe(c *gin.Context) {
	h.create(c, recipeSpec.resource)
}

// UpdateRecipe modifies an existing recipe in Strapi.
//
// @Summary Update a recipe (admin or editor)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Param locale query string false "Locale of the entry to update"
// @Param request body object true "Strapi recipe attributes to change"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/recipes/{id} [put]
func (h *Handler) UpdateRecipe(c *gin.Context) {
	h.update(c, recipeSpec.resource)
}

// DeleteRecipe removes a recipe from Strapi.
//
// @Summary Delete a recipe (admin only)
// @Tags content
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/recipes/{id} [delete]
func (h *Handler) DeleteRecipe(c *gin.Context) {
	h.remove(c, recipeSpec.resource)
}

// CreateMainCategory stores a new main category in Strapi.
//
// @Summary Create a category (admin only)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param locale query string false "Locale for the new entry (defaults to configured Persian locale)"
// @Param request body object true "Strapi main-category attributes (name, slug, description)"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/categories [post]
func (h *Handler) CreateMainCategory(c *gin.Context) {
	h.create(c, mainCategorySpec.resource)
}

// UpdateMainCategory modifies an existing main category in Strapi.
//
// @Summary Update a category (admin or editor)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Param locale query string false "Locale of the entry to update"
// @Param request body object true "Strapi main-category attributes to change"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/categories/{id} [put]
func (h *Handler) UpdateMainCategory(c *gin.Context) {
	h.update(c, mainCategorySpec.resource)
}

// DeleteMainCategory removes a main category from Strapi.
//
// @Summary Delete a category (admin only)
// @Tags content
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/categories/{id} [delete]
func (h *Handler) DeleteMainCategory(c *gin.Context) {
	h.remove(c, mainCategorySpec.resource)
}

// CreateProduct stores a new product in Strapi.
//
// @Summary Create a product (admin only)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param locale query string false "Locale for the new entry (defaults to configured Persian locale)"
// @Param request body object true "Strapi product attributes (name, slug, brand, relations)"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/products [post]
func (h *Handler) CreateProduct(c *gin.Context) {
	h.create(c, productSpec.resource)
}

// UpdateProduct modifies an existing product in Strapi.
//
// @Summary Update a product (admin or editor)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Param locale query string false "Locale of the entry to update"
// @Param request body object true "Strapi product attributes to change"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/products/{id} [put]
func (h *Handler) UpdateProduct(c *gin.Context) {
	h.update(c, productSpec.resource)
}

// DeleteProduct removes a product from Strapi.
//
// @Summary Delete a product (admin only)
// @Tags content
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/products/{id} [delete]
func (h *Handler) DeleteProduct(c *gin.Context) {
	h.remove(c, productSpec.resource)
}

// CreateNewsletter stores a new newsletter in Strapi.
//
// @Summary Create a newsletter (admin only)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param locale query string false "Locale for the new entry (defaults to configured Persian locale)"
// @Param request body object true "Strapi newsletter attributes (subject, body, heroImage)"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /v1/newsletters [post]
func (h *Handler) CreateNewsletter(c *gin.Context) {
	h.create(c, newsletterSpec.resource)
}

// UpdateNewsletter modifies an existing newsletter in Strapi.
//
// @Summary Update a newsletter (admin or editor)
// @Tags content
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Param locale query string false "Locale of the entry to update"
// @Param request body object true "Strapi newsletter attributes to change"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/newsletters/{id} [put]
func (h *Handler) UpdateNewsletter(c *gin.Context) {
	h.update(c, newsletterSpec.resource)
}

// DeleteNewsletter removes a newsletter from Strapi.
//
// @Summary Delete a newsletter (admin only)
// @Tags content
// @Security BearerAuth
// @Param id path string true "Strapi documentId"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Router /v1/newsletters/{id} [delete]
func (h *Handler) DeleteNewsletter(c *gin.Context) {
	h.remove(c, newsletterSpec.resource)
}
