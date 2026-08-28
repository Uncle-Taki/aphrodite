package http

import (
	"context"
	"testing"

	"aphrodite/internal/content"
)

type fakeSlugReader struct {
	queries []content.Query
	result  content.Document
	callErr error
}

func (f *fakeSlugReader) List(context.Context, string, content.Query) (content.DocumentList, error) {
	return content.DocumentList{}, nil
}

func (f *fakeSlugReader) Get(context.Context, string, string, content.Query) (content.Document, error) {
	return content.Document{}, nil
}

func (f *fakeSlugReader) FindBySlug(_ context.Context, _ string, _ string, query content.Query) (content.Document, error) {
	f.queries = append(f.queries, query)
	return f.result, f.callErr
}

func TestNewHandlerDefaultsLocales(t *testing.T) {
	h := NewHandler(&fakeSlugReader{}, "", "")
	if h.defaultLocale != "fa" || h.fallbackLocale != "fa" {
		t.Fatalf("unexpected defaults: %+v", h)
	}
}
