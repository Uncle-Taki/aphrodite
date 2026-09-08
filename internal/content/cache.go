package content

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type CachedSlugReader struct {
	reader SlugReader
	cache  Cache
	ttl    time.Duration
}

func NewCachedSlugReader(reader SlugReader, cache Cache, ttl time.Duration) *CachedSlugReader {
	return &CachedSlugReader{reader: reader, cache: cache, ttl: ttl}
}

var _ SlugReader = (*CachedSlugReader)(nil)

func (r *CachedSlugReader) List(ctx context.Context, resource string, query Query) (DocumentList, error) {
	return r.reader.List(ctx, resource, query)
}

func (r *CachedSlugReader) Get(ctx context.Context, resource, documentID string, query Query) (Document, error) {
	return r.reader.Get(ctx, resource, documentID, query)
}

func (r *CachedSlugReader) FindBySlug(ctx context.Context, resource, slug string, query Query) (Document, error) {
	if r.cache == nil || r.ttl <= 0 {
		return r.reader.FindBySlug(ctx, resource, slug, query)
	}
	key := cacheKeyForQuery(resource, slug, query)
	if encoded, err := r.cache.Get(ctx, key); err == nil {
		var document Document
		if json.Unmarshal([]byte(encoded), &document) == nil {
			return document, nil
		}
	}
	document, err := r.reader.FindBySlug(ctx, resource, slug, query)
	if err != nil {
		return Document{}, err
	}
	if encoded, marshalErr := json.Marshal(document); marshalErr == nil {
		_ = r.cache.Set(ctx, key, encoded, r.ttl)
	}
	return document, nil
}

func cacheKey(resource, slug, locale, status string) string {
	return cacheKeyForQuery(resource, slug, Query{Locale: locale, Status: status})
}

func cacheKeyForQuery(resource, slug string, query Query) string {
	// The response also depends on fields, populate, pagination, sorting, and
	// filters. Canonical JSON gives us a deterministic representation (including
	// sorted map keys) without exposing user-controlled values in Redis keys.
	encoded, err := json.Marshal(struct {
		Resource string `json:"resource"`
		Slug     string `json:"slug"`
		Query    Query  `json:"query"`
	}{resource, slug, query})
	if err != nil {
		encoded = []byte(fmt.Sprintf("%s|%s|%s|%s", resource, slug, query.Locale, query.Status))
	}
	sum := sha256.Sum256(encoded)
	return "content:strapi:" + hex.EncodeToString(sum[:])
}
