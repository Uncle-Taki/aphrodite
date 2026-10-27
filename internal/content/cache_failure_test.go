package content

import (
	"context"
	"errors"
	"testing"
	"time"
)

type failingReader struct {
	*countingReader
	err error
}

func (r failingReader) FindBySlug(context.Context, string, string, Query) (Document, error) {
	return Document{}, r.err
}

func TestCachedSlugReader_DoesNotCacheFailures(t *testing.T) {
	failure := errors.New("upstream failed")
	cache := &memoryCache{}
	reader := NewCachedSlugReader(failingReader{&countingReader{}, failure}, cache, time.Minute)
	if _, err := reader.FindBySlug(context.Background(), "articles", "hello", Query{}); !errors.Is(err, failure) {
		t.Fatalf("error=%v", err)
	}
	if len(cache.values) != 0 {
		t.Fatal("cached failed lookup")
	}
}

func TestCachedSlugReader_RefreshesCorruptCache(t *testing.T) {
	query := Query{Locale: "fa"}
	cache := &memoryCache{values: map[string]string{cacheKeyForQuery("articles", "hello", query): "invalid JSON"}}
	upstream := &countingReader{}
	reader := NewCachedSlugReader(upstream, cache, time.Minute)
	doc, err := reader.FindBySlug(context.Background(), "articles", "hello", query)
	if err != nil || doc.DocumentID != "doc-1" || upstream.calls != 1 {
		t.Fatalf("corrupt cache recovery: %+v %v", doc, err)
	}
}
