package http

import (
	"aphrodite/internal/content"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContentRoutes_ReadAndWriteAllResources(t *testing.T) {
	for _, tc := range []struct{ path, resource string }{{"articles", "articles"}, {"recipes", "recipes"}, {"categories", "main-categories"}, {"products", "products"}, {"newsletters", "newsletters"}} {
		t.Run(tc.path, func(t *testing.T) {
			reader := &fakeSlugReader{result: content.Document{DocumentID: "doc-1", Fields: map[string]any{"title": "hello"}}, list: content.DocumentList{Items: []content.Document{{DocumentID: "doc-1", Fields: map[string]any{"title": "hello"}}}, Total: 1}}
			writer := &fakeWriter{reader: reader}
			router := newWriteRouter(t, writer, authGate(false), allow(), allow())
			for _, op := range []struct {
				method, path string
				status       int
			}{{"GET", "", 200}, {"GET", "/hello?locale=en", 200}, {"POST", "", 201}, {"PUT", "/doc-1", 200}, {"DELETE", "/doc-1", 204}} {
				res := httptest.NewRecorder()
				router.ServeHTTP(res, httptest.NewRequest(op.method, "/v1/"+tc.path+op.path, strings.NewReader(`{"title":"hello"}`)))
				if res.Code != op.status {
					t.Fatalf("%s %s: %d %s", op.method, op.path, res.Code, res.Body.String())
				}
				if op.method == "POST" && writer.create.resource != tc.resource {
					t.Fatalf("create routed to %s", writer.create.resource)
				}
				if op.method == "PUT" && writer.update.resource != tc.resource {
					t.Fatalf("update routed to %s", writer.update.resource)
				}
				if op.method == "GET" && op.path == "" {
					var body ContentListResponse
					if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
						t.Fatal(err)
					}
					if len(body.Items) != 1 || body.Total != 1 {
						t.Fatalf("list response: %+v", body)
					}
				}
			}
		})
	}
}

func TestContentRoutes_RejectInvalidUpdateAndFailedDelete(t *testing.T) {
	writer := &fakeWriter{reader: &fakeSlugReader{}, deleteErr: content.ErrNotFound}
	router := newWriteRouter(t, writer, authGate(false), allow(), allow())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/v1/articles/doc", strings.NewReader("{")))
	if res.Code != 400 || writer.update.called {
		t.Fatalf("invalid update: %d, called=%v", res.Code, writer.update.called)
	}
	res = httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodDelete, "/v1/articles/doc", nil))
	if res.Code != 404 {
		t.Fatalf("failed delete: %d", res.Code)
	}
}
