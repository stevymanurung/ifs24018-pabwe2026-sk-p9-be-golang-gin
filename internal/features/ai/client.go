package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient interface {
	ChatCompletion(req *OpenAIChatRequest) (*OpenAIChatResponse, error)
}

type openAIClient struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

func NewOpenAIClient(endpoint, apiKey string, customClient ...*http.Client) OpenAIClient {
	client := &http.Client{Timeout: 60 * time.Second}
	if len(customClient) > 0 && customClient[0] != nil {
		client = customClient[0]
	}
	cleanEndpoint := strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(cleanEndpoint, "/chat/completions") {
		cleanEndpoint = cleanEndpoint + "/chat/completions"
	}
	return &openAIClient{
		endpoint:   cleanEndpoint,
		apiKey:     apiKey,
		httpClient: client,
	}
}

func (c *openAIClient) ChatCompletion(req *OpenAIChatRequest) (*OpenAIChatResponse, error) {
	reqBytes, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewBuffer(reqBytes))

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("koneksi ke layanan OpenAI gagal: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("layanan OpenAI mengembalikan status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp OpenAIChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("gagal parse respons JSON OpenAI: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("kesalahan OpenAI API: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("layanan OpenAI tidak menghasilkan respon (pilihan kosong)")
	}

	return &chatResp, nil
}
