package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type SearchClient interface {
	Search(query string) ([]SearchResultItem, error)
}

type searchClient struct {
	endpoint   string
	httpClient *http.Client
}

func NewSearchClient(endpoint string, customClient ...*http.Client) SearchClient {
	client := &http.Client{Timeout: 10 * time.Second}
	if len(customClient) > 0 && customClient[0] != nil {
		client = customClient[0]
	}
	return &searchClient{
		endpoint:   endpoint,
		httpClient: client,
	}
}

type searchResponseWrapper struct {
	Data []SearchResultItem `json:"data"`
}

func (s *searchClient) Search(query string) ([]SearchResultItem, error) {
	if s.endpoint == "" || query == "" {
		return nil, nil
	}

	searchURL := fmt.Sprintf("%s?q=%s", s.endpoint, url.QueryEscape(query))
	req, _ := http.NewRequest(http.MethodGet, searchURL, nil)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search engine returned status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	var directList []SearchResultItem
	if err := json.Unmarshal(body, &directList); err == nil && len(directList) > 0 {
		return directList, nil
	}

	var wrapped searchResponseWrapper
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Data) > 0 {
		return wrapped.Data, nil
	}

	return nil, nil
}
