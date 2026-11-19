package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}
type Draft struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Excerpt  string   `json:"excerpt"`
	Content  string   `json:"content"`
	Sources  []Source `json:"sources"`
}
type Service struct {
	SearchEndpoint, OpenAIEndpoint, APIKey, Model string
	HTTP                                          *http.Client
}

func New(search, endpoint, key, model string) *Service {
	return &Service{search, strings.TrimRight(endpoint, "/"), key, model, &http.Client{Timeout: 35 * time.Second}}
}
func (s *Service) Generate(topic, category string, useSearch bool) (Draft, error) {
	if strings.TrimSpace(topic) == "" {
		return Draft{}, errors.New("topic is required")
	}
	sources := []Source{}
	if useSearch && s.SearchEndpoint != "" {
		body, _ := json.Marshal(map[string]string{"query": topic})
		req, _ := http.NewRequest(http.MethodPost, s.SearchEndpoint, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res, err := s.HTTP.Do(req)
		if err == nil {
			defer res.Body.Close()
			var raw struct {
				Data    []Source `json:"data"`
				Results []Source `json:"results"`
			}
			if json.NewDecoder(res.Body).Decode(&raw) == nil {
				sources = raw.Data
				if len(sources) == 0 {
					sources = raw.Results
				}
			}
		}
	}
	if s.APIKey == "" || s.Model == "" {
		return Draft{}, errors.New("AI provider belum dikonfigurasi: isi OPENAI_KEY dan OPENAI_MODEL")
	}
	sourceText, _ := json.Marshal(sources)
	prompt := fmt.Sprintf("Buat artikel blog berbahasa Indonesia tentang %q kategori %q. Kembalikan JSON object dengan field title, category, excerpt, content (Markdown). Gunakan sumber berikut bila tersedia: %s", topic, category, string(sourceText))
	payload := map[string]any{"model": s.Model, "messages": []map[string]string{{"role": "system", "content": "Anda adalah editor blog profesional. Jangan mengarang fakta; gunakan sitasi sumber jika tersedia."}, {"role": "user", "content": prompt}}, "response_format": map[string]string{"type": "json_object"}}
	b, _ := json.Marshal(payload)
	endpoint := s.OpenAIEndpoint
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return Draft{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	res, err := s.HTTP.Do(req)
	if err != nil {
		return Draft{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Draft{}, fmt.Errorf("AI provider returned HTTP %d", res.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err = json.NewDecoder(res.Body).Decode(&out); err != nil {
		return Draft{}, err
	}
	if len(out.Choices) == 0 {
		return Draft{}, errors.New("AI provider returned no choices")
	}
	var d Draft
	if err = json.Unmarshal([]byte(out.Choices[0].Message.Content), &d); err != nil {
		return Draft{}, err
	}
	d.Sources = sources
	if d.Category == "" {
		d.Category = category
	}
	return d, nil
}
