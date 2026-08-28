package content

import (
	"context"
	"testing"
	"time"
)

type memoryCache struct{ values map[string]string }

func (m *memoryCache) Get(_ context.Context, key string) (string, error) {
	value, ok := m.values[key]
	if !ok {
		return "", context.Canceled
	}
	return value, nil
}

func (m *memoryCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[key] = string(value.([]byte))
	return nil
}

type countingReader struct{ calls int }

func (r *countingReader) List(context.Context, string, Query) (DocumentList, error) {
	return DocumentList{}, nil
}
func (r *countingReader) Get(context.Context, string, string, Query) (Document, error) {
	return Document{}, nil
}
func (r *countingReader) FindBySlug(context.Context, string, string, Query) (Document, error) {
	r.calls++
	return Document{DocumentID: "doc-1", Fields: map[string]any{"title": "cached"}}, nil
}

func TestCachedSlugReader(t *testing.T) {
	reader := &countingReader{}
	cached := NewCachedSlugReader(reader, &memoryCache{}, time.Minute)
	for i := 0; i < 2; i++ {
		if _, err := cached.FindBySlug(context.Background(), "articles", "hello", Query{Locale: "fa", Status: "published"}); err != nil {
			t.Fatal(err)
		}
	}
	if reader.calls != 1 {
		t.Fatalf("expected one upstream call, got %d", reader.calls)
	}
}

func TestCachedSlugReaderSeparatesPopulateQueries(t *testing.T) {
	reader := &countingReader{}
	cached := NewCachedSlugReader(reader, &memoryCache{}, time.Minute)
	query := Query{Locale: "fa", Status: "published", Populate: []string{"author"}}
	if _, err := cached.FindBySlug(context.Background(), "articles", "hello", query); err != nil {
		t.Fatal(err)
	}
	if _, err := cached.FindBySlug(context.Background(), "articles", "hello", Query{Locale: "fa", Status: "published", Populate: []string{"coverImage"}}); err != nil {
		t.Fatal(err)
	}
	if reader.calls != 2 {
		t.Fatalf("different populate selections must not share a cache entry, calls=%d", reader.calls)
	}
}
