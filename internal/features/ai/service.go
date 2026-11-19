package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"app/internal/config"
)

var (
	ErrTopicRequired = errors.New("field topic wajib diisi")
)

type Service interface {
	GenerateDraft(req *GenerateDraftRequest) (*DraftResponse, error)
}

type service struct {
	openAIClient OpenAIClient
	searchClient SearchClient
	model        string
}

func NewService(cfg *config.Config, openAIClient OpenAIClient, searchClient SearchClient) Service {
	model := cfg.OpenAIModel
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &service{
		openAIClient: openAIClient,
		searchClient: searchClient,
		model:        model,
	}
}

var jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

func cleanJSONResponse(raw string) string {
	raw = strings.TrimSpace(raw)
	if matches := jsonBlockRegex.FindStringSubmatch(raw); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start != -1 && end != -1 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func (s *service) GenerateDraft(req *GenerateDraftRequest) (*DraftResponse, error) {
	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		return nil, ErrTopicRequired
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Umum"
	}

	var searchSources []SearchResultItem
	searchContext := ""
	if req.IncludeWebSearch && s.searchClient != nil {
		results, err := s.searchClient.Search(topic)
		if err == nil && len(results) > 0 {
			searchSources = results
			searchContext = "\n\nBerikut konteks informasi aktual dari mesin pencari untuk dijadikan referensi:\n"
			for i, r := range results {
				if i >= 5 {
					break
				}
				searchContext += fmt.Sprintf("- [%s] (%s): %s\n", r.Title, r.URL, r.Snippet)
			}
		}
	}

	systemPrompt := "Anda adalah asisten penulis artikel blog profesional. Tulislah draf artikel blog yang informatif, menarik, dan terstruktur rapi dalam format Markdown. " +
		"Berikan respon HANYA dalam bentuk JSON valid dengan format: " +
		"{\"title\": \"string\", \"category\": \"string\", \"excerpt\": \"string\", \"content\": \"string Markdown lengkap minimal 200 kata\"}."

	userPrompt := fmt.Sprintf("Topik Artikel: %s\nKategori: %s%s\n\nBuat draf artikel lengkap sekarang:", topic, category, searchContext)

	chatReq := &OpenAIChatRequest{
		Model: s.model,
		Messages: []OpenAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
	}

	chatResp, err := s.openAIClient.ChatCompletion(chatReq)
	if err != nil {
		return nil, fmt.Errorf("layanan OpenAI bermasalah: %w", err)
	}

	rawContent := chatResp.Choices[0].Message.Content
	cleanedJSON := cleanJSONResponse(rawContent)

	var draft DraftResponse
	if err := json.Unmarshal([]byte(cleanedJSON), &draft); err != nil {
		// Fallback if model returned plain text markdown
		draft = DraftResponse{
			Title:    topic,
			Category: category,
			Excerpt:  fmt.Sprintf("Artikel ulasan tentang %s.", topic),
			Content:  rawContent,
		}
	}

	if draft.Category == "" {
		draft.Category = category
	}
	if len(searchSources) > 0 {
		draft.Sources = searchSources
	}

	return &draft, nil
}
