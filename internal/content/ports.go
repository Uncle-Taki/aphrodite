package content

import "context"

type DocumentReader interface {
	List(ctx context.Context, resource string, query Query) (DocumentList, error)
	Get(ctx context.Context, resource, documentID string, query Query) (Document, error)
}

type SlugReader interface {
	DocumentReader
	FindBySlug(ctx context.Context, resource, slug string, query Query) (Document, error)
}

type Query struct {
	Locale   string
	Status   string
	Page     int
	PageSize int
	Sort     []string
	Fields   []string
	Populate []string
	Filters  map[string]string
}

type Document struct {
	ID         int
	DocumentID string
	Fields     map[string]any
	Meta       map[string]any
}

type DocumentList struct {
	Items     []Document
	Page      int
	PageSize  int
	PageCount int
	Total     int
}
