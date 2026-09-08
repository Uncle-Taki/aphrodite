package strapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"aphrodite/internal/content"
)

var _ content.DocumentReader = (*Client)(nil)

type Client struct {
	baseURL         string
	token           string
	httpClient      *http.Client
	maxRetries      int
	retryBackoff    time.Duration
	maxRetryBackoff time.Duration
	sleep           func(context.Context, time.Duration) error
}

func New(cfg Config) (*Client, error) {
	return newClient(cfg, &http.Client{Timeout: cfg.Timeout}, defaultSleep)
}

func newClient(cfg Config, httpClient *http.Client, sleep func(context.Context, time.Duration) error) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	if sleep == nil {
		sleep = defaultSleep
	}
	return &Client{
		baseURL:         strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		token:           cfg.APIToken,
		httpClient:      httpClient,
		maxRetries:      cfg.MaxRetries,
		retryBackoff:    cfg.RetryBackoff,
		maxRetryBackoff: cfg.MaxRetryBackoff,
		sleep:           sleep,
	}, nil
}

func (c *Client) List(ctx context.Context, resource string, query content.Query) (content.DocumentList, error) {
	var envelope response
	if err := c.do(ctx, http.MethodGet, resource, query, &envelope); err != nil {
		return content.DocumentList{}, err
	}
	var items []documentDTO
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, &items); err != nil {
			return content.DocumentList{}, fmt.Errorf("%w: list data: %v", ErrInvalidResponse, err)
		}
	}
	result := content.DocumentList{Items: make([]content.Document, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, toDocument(item))
	}
	if pagination, ok := envelope.Meta["pagination"].(map[string]any); ok {
		result.Page = intValue(pagination["page"])
		result.PageSize = intValue(pagination["pageSize"])
		result.PageCount = intValue(pagination["pageCount"])
		result.Total = intValue(pagination["total"])
	}
	return result, nil
}

func (c *Client) Get(ctx context.Context, resource, documentID string, query content.Query) (content.Document, error) {
	if strings.TrimSpace(documentID) == "" {
		return content.Document{}, fmt.Errorf("%w: empty document ID", ErrInvalidResponse)
	}
	var envelope response
	if err := c.do(ctx, http.MethodGet, resource+"/"+url.PathEscape(documentID), query, &envelope); err != nil {
		return content.Document{}, err
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return content.Document{}, ErrNotFound
	}
	var item documentDTO
	if err := json.Unmarshal(envelope.Data, &item); err != nil {
		return content.Document{}, fmt.Errorf("%w: document data: %v", ErrInvalidResponse, err)
	}
	result := toDocument(item)
	if result.Meta == nil {
		result.Meta = envelope.Meta
	}
	return result, nil
}

func (c *Client) FindBySlug(ctx context.Context, resource, slug string, query content.Query) (content.Document, error) {
	if strings.TrimSpace(slug) == "" {
		return content.Document{}, fmt.Errorf("%w: empty slug", ErrInvalidResponse)
	}
	// Do not mutate the caller's query (or its map) while adding the slug
	// constraint; callers commonly reuse a query for fallback locales.
	filters := make(map[string]string, len(query.Filters)+1)
	for key, value := range query.Filters {
		filters[key] = value
	}
	query.Filters = filters
	query.Filters["slug[$eq]"] = slug
	query.Page = 1
	query.PageSize = 1
	items, err := c.List(ctx, resource, query)
	if err != nil {
		return content.Document{}, err
	}
	if len(items.Items) == 0 {
		return content.Document{}, ErrNotFound
	}
	return items.Items[0], nil
}

func (c *Client) do(ctx context.Context, method, resource string, query content.Query, out *response) error {
	if strings.TrimSpace(resource) == "" || strings.HasPrefix(resource, "/") {
		return fmt.Errorf("%w: resource must be a non-empty API ID", ErrInvalidResponse)
	}
	requestURL := c.baseURL + "/api/" + resource
	values := queryValues(query)
	if encoded := values.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
		if err != nil {
			return fmt.Errorf("%w: create request: %v", ErrInvalidResponse, err)
		}
		req.Header.Set("Accept", "application/json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < c.maxRetries && (retryableTransportError(err) || errors.Is(err, context.DeadlineExceeded)) {
				if err := c.sleep(ctx, c.backoff(attempt)); err != nil {
					return err
				}
				continue
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return err
			}
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}

		body, readErr := io.ReadAll(io.LimitReader(res.Body, 4<<20))
		res.Body.Close()
		if readErr != nil {
			if attempt < c.maxRetries {
				if err := c.sleep(ctx, c.backoff(attempt)); err != nil {
					return err
				}
				continue
			}
			return fmt.Errorf("%w: read response: %v", ErrUnavailable, readErr)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			httpErr := decodeHTTPError(res.StatusCode, body)
			if attempt < c.maxRetries && retryableStatus(res.StatusCode) {
				delay := c.backoff(attempt)
				if retryAfter := retryAfter(res.Header.Get("Retry-After")); retryAfter > delay {
					delay = retryAfter
				}
				if err := c.sleep(ctx, delay); err != nil {
					return err
				}
				continue
			}
			return httpErr
		}
		if len(body) == 0 {
			return fmt.Errorf("%w: empty response", ErrInvalidResponse)
		}
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
		}
		if out.Error != nil {
			return &HTTPError{Status: out.Error.Status, Name: out.Error.Name, Message: out.Error.Message}
		}
		return nil
	}
}

func queryValues(query content.Query) url.Values {
	values := url.Values{}
	if query.Locale != "" {
		values.Set("locale", query.Locale)
	}
	if query.Status != "" {
		values.Set("status", query.Status)
	}
	if query.Page > 0 {
		values.Set("pagination[page]", strconv.Itoa(query.Page))
	}
	if query.PageSize > 0 {
		values.Set("pagination[pageSize]", strconv.Itoa(query.PageSize))
	}
	for _, sort := range query.Sort {
		values.Add("sort[]", sort)
	}
	for _, field := range query.Fields {
		values.Add("fields[]", field)
	}
	for _, populate := range query.Populate {
		values.Add("populate[]", populate)
	}
	for key, value := range query.Filters {
		values.Set("filters["+key+"]", value)
	}
	return values
}

func decodeHTTPError(status int, body []byte) error {
	var envelope response
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != nil {
		return &HTTPError{Status: status, Name: envelope.Error.Name, Message: envelope.Error.Message}
	}
	return &HTTPError{Status: status, Message: http.StatusText(status)}
}

func toDocument(item documentDTO) content.Document {
	return content.Document{ID: item.ID, DocumentID: item.DocumentID, Fields: item.Fields, Meta: item.Meta}
}

func intValue(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func retryableTransportError(err error) bool {
	return !errors.Is(err, context.Canceled)
}

func retryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status == 500 || status == 502 || status == 503 || status == 504
}

func retryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		if delay := time.Until(at); delay > 0 {
			return delay
		}
	}
	return 0
}

func (c *Client) backoff(attempt int) time.Duration {
	delay := c.retryBackoff
	for i := 0; i < attempt; i++ {
		if delay >= c.maxRetryBackoff/2 && c.maxRetryBackoff > 0 {
			delay = c.maxRetryBackoff
			break
		}
		delay *= 2
	}
	if c.maxRetryBackoff > 0 && delay > c.maxRetryBackoff {
		delay = c.maxRetryBackoff
	}
	if delay <= 0 {
		return 0
	}
	return delay/2 + time.Duration(rand.Int63n(int64(delay/2)+1))
}

func defaultSleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
