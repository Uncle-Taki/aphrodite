package strapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"aphrodite/internal/content"
)

func TestClient_RejectsInvalidConfiguration(t *testing.T) {
	valid := Config{BaseURL: "http://strapi.test", APIToken: "test-token", Timeout: time.Second}
	for _, tc := range []struct {
		name   string
		mutate func(*Config)
	}{
		{"timeout", func(c *Config) { c.Timeout = 0 }},
		{"retries", func(c *Config) { c.MaxRetries = -1 }},
		{"backoff", func(c *Config) { c.RetryBackoff = -1 }},
		{"maximum backoff", func(c *Config) { c.MaxRetryBackoff = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			tc.mutate(&cfg)
			if _, err := New(cfg); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
	c, err := newClient(valid, nil, nil)
	if err != nil || c.httpClient.Timeout != time.Second || c.sleep == nil {
		t.Fatalf("default dependencies: %v, %v", c, err)
	}
}

func TestClient_RejectsInvalidResponses(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       error
	}{
		{"invalid envelope", "{", ErrInvalidResponse},
		{"wrong document type", `{"data":42}`, ErrInvalidResponse},
		{"missing document", `{"data":null}`, ErrNotFound},
		{"error in success envelope", `{"error":{"status":403,"message":"denied"}}`, ErrForbidden},
	} {
		for _, operation := range []string{"get", "create", "update"} {
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				c, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, tc.body) }))
				var err error
				switch operation {
				case "get":
					_, err = c.Get(context.Background(), "articles", "doc", content.Query{})
				case "create":
					_, err = c.Create(context.Background(), "articles", nil, content.Query{})
				case "update":
					_, err = c.Update(context.Background(), "articles", "doc", nil, content.Query{})
				}
				if !errors.Is(err, tc.want) {
					t.Fatalf("error=%v, want %v", err, tc.want)
				}
			})
		}
	}
	c, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{"data":{}}`) }))
	if _, err := c.List(context.Background(), "articles", content.Query{}); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("list: %v", err)
	}
	if _, err := c.FindBySlug(context.Background(), "articles", " ", content.Query{}); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("blank slug: %v", err)
	}
	if _, err := c.FindBySlug(context.Background(), "articles", "slug", content.Query{}); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("invalid slug response: %v", err)
	}
	c.baseURL = "http://invalid%host"
	if _, err := c.List(context.Background(), "articles", content.Query{}); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("malformed URL: %v", err)
	}
}

func TestClient_RejectsUnencodablePayload(t *testing.T) {
	c, _ := testClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("invalid payload sent to Strapi") }))
	invalid := map[string]any{"bad": make(chan int)}
	if _, err := c.Create(context.Background(), "articles", invalid, content.Query{}); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("create: %v", err)
	}
	if _, err := c.Update(context.Background(), "articles", "doc", invalid, content.Query{}); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("update: %v", err)
	}
}

func TestClient_CancellationStopsRetries(t *testing.T) {
	for _, failure := range []string{"transport", "body", "status", "canceled transport"} {
		t.Run(failure, func(t *testing.T) {
			calls := 0
			c, _ := testClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				switch failure {
				case "transport":
					return nil, io.ErrUnexpectedEOF
				case "canceled transport":
					return nil, context.Canceled
				case "body":
					return &http.Response{StatusCode: 200, Body: io.NopCloser(iotest.ErrReader(io.ErrUnexpectedEOF))}, nil
				default:
					return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"5"}}, Body: http.NoBody}, nil
				}
			})
			c.sleep = func(context.Context, time.Duration) error { return context.Canceled }
			_, err := c.List(context.Background(), "articles", content.Query{})
			if !errors.Is(err, context.Canceled) || calls != 1 {
				t.Fatalf("error=%v calls=%d", err, calls)
			}
		})
	}
}

func TestDocumentDTO_ValidatesFields(t *testing.T) {
	for _, input := range []string{`[]`, `{"id":"bad"}`, `{"documentId":3}`, `{"meta":[]}`, `{"price":1e1000}`} {
		var doc documentDTO
		if err := json.Unmarshal([]byte(input), &doc); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	var doc documentDTO
	if err := json.Unmarshal([]byte(`{"id":7,"documentId":"doc","meta":{"source":"document"},"title":"hello"}`), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.ID != 7 || doc.DocumentID != "doc" || doc.Meta["source"] != "document" || doc.Fields["title"] != "hello" {
		t.Fatalf("document: %+v", doc)
	}
	result, err := decodeSingleDocument(response{Data: json.RawMessage(`{"id":7}`), Meta: map[string]any{"source": "envelope"}})
	if err != nil || result.Meta["source"] != "envelope" {
		t.Fatalf("metadata: %+v %v", result, err)
	}
}

func TestHTTPError_PreservesErrorIdentity(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   error
	}{
		{400, ErrInvalidRequest}, {401, ErrUnauthorized}, {403, ErrForbidden}, {404, ErrNotFound}, {409, ErrConflict}, {429, ErrRateLimited}, {500, ErrUnavailable}, {502, ErrUnavailable}, {503, ErrUnavailable}, {504, ErrUnavailable}, {418, nil},
	} {
		err := &HTTPError{Status: tc.status}
		if errors.Unwrap(err) != tc.want {
			t.Errorf("status %d: %v", tc.status, errors.Unwrap(err))
		}
		if !strings.Contains(err.Error(), "HTTP") {
			t.Errorf("missing status: %v", err)
		}
	}
}

func TestRetryTiming(t *testing.T) {
	if retryAfter("nonsense") != 0 || retryAfter(time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)) != 0 {
		t.Fatal("invalid or past retry-after accepted")
	}
	if delay := retryAfter(time.Now().Add(time.Minute).UTC().Format(http.TimeFormat)); delay < 58*time.Second || delay > time.Minute {
		t.Fatalf("date retry-after: %v", delay)
	}
	if retryAfter(" 3 ") != 3*time.Second {
		t.Fatal("numeric retry-after")
	}
	c := &Client{retryBackoff: 10 * time.Millisecond, maxRetryBackoff: 8 * time.Millisecond}
	if delay := c.backoff(0); delay < 4*time.Millisecond || delay > 8*time.Millisecond {
		t.Fatalf("capped delay: %v", delay)
	}
	c.retryBackoff = time.Millisecond
	if delay := c.backoff(10); delay < 4*time.Millisecond || delay > 8*time.Millisecond {
		t.Fatalf("exponential cap: %v", delay)
	}
	c.retryBackoff = 0
	if c.backoff(0) != 0 {
		t.Fatal("zero backoff")
	}
	if err := defaultSleep(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if err := defaultSleep(context.Background(), time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := defaultSleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleep: %v", err)
	}
	if intValue(3) != 3 || intValue("bad") != 0 {
		t.Fatal("pagination conversion")
	}
	q := queryValues(content.Query{Sort: []string{"title:asc"}, Fields: []string{"title"}})
	if q.Get("sort[]") != "title:asc" || q.Get("fields[]") != "title" {
		t.Fatalf("query: %v", q)
	}
}
