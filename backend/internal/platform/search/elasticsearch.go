package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LeeJc02/WeHi/backend/pkg/contracts"
)

type Client struct {
	baseURL string
	http    *http.Client
	mock    bool
}

func New(baseURL string) *Client {
	trimmed := strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: trimmed,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		mock: strings.HasPrefix(trimmed, "mock://") || trimmed == "",
	}
}

func (c *Client) Ping(ctx context.Context) error {
	if c.mock {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("elasticsearch status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) EnsureIndex(ctx context.Context, index string, mapping map[string]any) error {
	if c.mock {
		return nil
	}
	body, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/%s", c.baseURL, index), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 300 || resp.StatusCode == http.StatusBadRequest {
		return nil
	}
	data, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("ensure index %s: %s", index, string(data))
}

func (c *Client) IndexDocument(ctx context.Context, index string, id uint64, doc any) error {
	if c.mock {
		return nil
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/%s/_doc/%d", c.baseURL, index, id), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("index document: %s", string(data))
	}
	return nil
}

func (c *Client) SearchMessages(ctx context.Context, index, query string, conversationIDs []uint64, offset, limit int) ([]contracts.SearchMessageHit, error) {
	if c.mock {
		return nil, errors.New("search mock mode does not support direct index queries")
	}
	reqBody := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []any{
			map[string]any{"_score": "desc"},
			map[string]any{"created_at": map[string]any{"order": "desc"}},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"multi_match": map[string]any{
							"query":  query,
							"fields": []string{"content^3", "conversation_name"},
						},
					},
				},
				"filter": []any{
					map[string]any{
						"terms": map[string]any{"conversation_id": conversationIDs},
					},
				},
			},
		},
	}
	var response struct {
		Hits struct {
			Hits []struct {
				Source contracts.SearchMessageHit `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := c.search(ctx, index, reqBody, &response); err != nil {
		return nil, err
	}
	result := make([]contracts.SearchMessageHit, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		result = append(result, hit.Source)
	}
	return result, nil
}

func (c *Client) SearchConversations(ctx context.Context, index, query string, conversationIDs []uint64, offset, limit int) ([]contracts.SearchConversationHit, error) {
	if c.mock {
		return nil, errors.New("search mock mode does not support direct index queries")
	}
	reqBody := map[string]any{
		"from": offset,
		"size": limit,
		"sort": []any{
			map[string]any{"_score": "desc"},
			map[string]any{"updated_at": map[string]any{"order": "desc"}},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"match": map[string]any{"name": query},
					},
				},
				"filter": []any{
					map[string]any{
						"terms": map[string]any{"conversation_id": conversationIDs},
					},
				},
			},
		},
	}
	var response struct {
		Hits struct {
			Hits []struct {
				Source contracts.SearchConversationHit `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := c.search(ctx, index, reqBody, &response); err != nil {
		return nil, err
	}
	result := make([]contracts.SearchConversationHit, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		result = append(result, hit.Source)
	}
	return result, nil
}

func (c *Client) search(ctx context.Context, index string, body any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s/_search", c.baseURL, url.PathEscape(index)), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("search failed: %s", string(data))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *Client) IsMock() bool {
	return c == nil || c.mock
}
