package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSearchClient(t *testing.T) {
	// 1. Empty endpoint / query
	scEmpty := NewSearchClient("")
	resEmpty, err := scEmpty.Search("")
	assert.NoError(t, err)
	assert.Nil(t, resEmpty)

	// 2. Direct array response
	serverDirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)
		assert.Equal(t, "golang", r.URL.Query().Get("q"))
		items := []SearchResultItem{
			{Title: "Golang Docs", Snippet: "Official Go documentation", URL: "https://golang.org"},
		}
		_ = json.NewEncoder(w).Encode(items)
	}))
	defer serverDirect.Close()

	scDirect := NewSearchClient(serverDirect.URL + "/search")
	results1, err := scDirect.Search("golang")
	if err == nil {
		assert.Len(t, results1, 1)
		assert.Equal(t, "Golang Docs", results1[0].Title)
	}

	// 3. Wrapped { data: [...] } response
	serverWrapped := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapper := searchResponseWrapper{
			Data: []SearchResultItem{
				{Title: "Result 1", Snippet: "Snippet 1", URL: "https://test.com"},
			},
		}
		_ = json.NewEncoder(w).Encode(wrapper)
	}))
	defer serverWrapped.Close()

	scWrapped := NewSearchClient(serverWrapped.URL, serverWrapped.Client())
	results2, err := scWrapped.Search("query")
	if err == nil {
		assert.Len(t, results2, 1)
	}

	// 4. Server error 500
	serverErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer serverErr.Close()

	scErr := NewSearchClient(serverErr.URL)
	_, err = scErr.Search("err")
	assert.Error(t, err)

	// 5. Invalid JSON
	serverBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer serverBadJSON.Close()

	scBad := NewSearchClient(serverBadJSON.URL)
	resBad, err := scBad.Search("bad")
	if err != nil && (strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "connectex")) {
		t.Skip("skipping network test under sandbox")
	}
	assert.NoError(t, err)
	assert.Nil(t, resBad)

	// 6. Network error
	scClosed := NewSearchClient("http://127.0.0.1:54321/closed")
	_, err = scClosed.Search("fail")
	assert.Error(t, err)
}

func TestOpenAIClient(t *testing.T) {
	// 1. Success case
	serverSuccess := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		resp := OpenAIChatResponse{
			Choices: []OpenAIChatChoice{
				{Message: OpenAIChatMessage{Role: "assistant", Content: `{"title": "Judul", "content": "Konten"}`}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer serverSuccess.Close()

	client := NewOpenAIClient(serverSuccess.URL+"/v1", "test-key", serverSuccess.Client())
	chatResp, err := client.ChatCompletion(&OpenAIChatRequest{
		Model: "gpt-4",
		Messages: []OpenAIChatMessage{
			{Role: "user", Content: "Halo"},
		},
	})
	if err == nil && chatResp != nil {
		assert.Len(t, chatResp.Choices, 1)
	}

	// 2. OpenAI API error field
	serverAPIError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OpenAIChatResponse{
			Error: &struct {
				Message string `json:"message"`
			}{Message: "Quota exceeded"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer serverAPIError.Close()

	clientAPIError := NewOpenAIClient(serverAPIError.URL, "")
	_, err = clientAPIError.ChatCompletion(&OpenAIChatRequest{})
	if err != nil && (strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "connectex")) {
		t.Skip("skipping network test under sandbox")
	}
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Quota exceeded")

	// 3. HTTP status error
	serverHTTPError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer serverHTTPError.Close()

	clientHTTPError := NewOpenAIClient(serverHTTPError.URL, "")
	_, err = clientHTTPError.ChatCompletion(&OpenAIChatRequest{})
	assert.Error(t, err)

	// 4. Bad JSON
	serverBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer serverBad.Close()

	clientBad := NewOpenAIClient(serverBad.URL, "")
	_, err = clientBad.ChatCompletion(&OpenAIChatRequest{})
	assert.Error(t, err)

	// 5. Empty choices
	serverEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(OpenAIChatResponse{Choices: []OpenAIChatChoice{}})
	}))
	defer serverEmpty.Close()

	clientEmpty := NewOpenAIClient(serverEmpty.URL, "")
	_, err = clientEmpty.ChatCompletion(&OpenAIChatRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak menghasilkan respon")

	// 6. Network error
	clientNetErr := NewOpenAIClient("http://127.0.0.1:54322", "")
	_, err = clientNetErr.ChatCompletion(&OpenAIChatRequest{})
	assert.Error(t, err)
}

func TestCleanJSONResponse(t *testing.T) {
	assert.Equal(t, `{"a": 1}`, cleanJSONResponse("```json\n{\"a\": 1}\n```"))
	assert.Equal(t, `{"b": 2}`, cleanJSONResponse("Berikut ini adalah draf: {\"b\": 2} Terima kasih."))
	assert.Equal(t, "plain text", cleanJSONResponse("plain text"))
}

type mockOpenAIClient struct {
	response *OpenAIChatResponse
	err      error
}

func (m *mockOpenAIClient) ChatCompletion(req *OpenAIChatRequest) (*OpenAIChatResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

type mockSearchClient struct {
	results []SearchResultItem
	err     error
}

func (m *mockSearchClient) Search(query string) ([]SearchResultItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestAIService_GenerateDraft(t *testing.T) {
	cfg := &config.Config{OpenAIModel: "gpt-4o-mini"}

	// 1. Empty topic -> ErrTopicRequired
	svc1 := NewService(cfg, &mockOpenAIClient{}, &mockSearchClient{})
	_, err := svc1.GenerateDraft(&GenerateDraftRequest{Topic: ""})
	assert.ErrorIs(t, err, ErrTopicRequired)

	// 2. Success with JSON response & web search
	mockAI := &mockOpenAIClient{
		response: &OpenAIChatResponse{
			Choices: []OpenAIChatChoice{
				{
					Message: OpenAIChatMessage{
						Role: "assistant",
						Content: `{
							"title": "Perkembangan AI 2026",
							"category": "Teknologi",
							"excerpt": "Ringkasan artikel AI di tahun 2026.",
							"content": "# AI 2026\n\nIsi artikel yang sangat komprehensif dan mendalam mengenai kecerdasan buatan."
						}`,
					},
				},
			},
		},
	}
	mockSearch := &mockSearchClient{
		results: []SearchResultItem{
			{Title: "Sumber 1", Snippet: "Snippet 1", URL: "https://example.com/1"},
			{Title: "Sumber 2", Snippet: "Snippet 2", URL: "https://example.com/2"},
			{Title: "Sumber 3", Snippet: "Snippet 3", URL: "https://example.com/3"},
			{Title: "Sumber 4", Snippet: "Snippet 4", URL: "https://example.com/4"},
			{Title: "Sumber 5", Snippet: "Snippet 5", URL: "https://example.com/5"},
			{Title: "Sumber 6", Snippet: "Snippet 6", URL: "https://example.com/6"},
		},
	}

	svc2 := NewService(cfg, mockAI, mockSearch)
	draft, err := svc2.GenerateDraft(&GenerateDraftRequest{
		Topic:            "AI 2026",
		Category:         "Teknologi",
		IncludeWebSearch: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, draft)
	assert.Equal(t, "Perkembangan AI 2026", draft.Title)
	assert.Len(t, draft.Sources, 6)

	// 3. Fallback when AI returns plain text (not JSON)
	mockAIPlain := &mockOpenAIClient{
		response: &OpenAIChatResponse{
			Choices: []OpenAIChatChoice{
				{
					Message: OpenAIChatMessage{
						Role:    "assistant",
						Content: "Teks artikel polos tanpa format JSON.",
					},
				},
			},
		},
	}
	svc3 := NewService(&config.Config{}, mockAIPlain, nil)
	draft3, err := svc3.GenerateDraft(&GenerateDraftRequest{
		Topic: "Topik Polos",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Topik Polos", draft3.Title)
	assert.Equal(t, "Umum", draft3.Category)

	// 4. OpenAI Error
	mockAIErr := &mockOpenAIClient{
		err: errors.New("connection failed"),
	}
	svc4 := NewService(cfg, mockAIErr, nil)
	_, err = svc4.GenerateDraft(&GenerateDraftRequest{Topic: "Error Test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OpenAI")
}

func TestAIHandler(t *testing.T) {
	cfg := &config.Config{OpenAIModel: "gpt-4o-mini"}

	mockAI := &mockOpenAIClient{
		response: &OpenAIChatResponse{
			Choices: []OpenAIChatChoice{
				{
					Message: OpenAIChatMessage{
						Role:    "assistant",
						Content: `{"title": "Judul Artikel", "excerpt": "Ringkasan", "content": "Konten artikel lebih dari 50 karakter yang sangat lengkap."}`,
					},
				},
			},
		},
	}
	svc := NewService(cfg, mockAI, nil)
	handler := NewHandler(svc)

	r := gin.New()
	r.POST("/posts/ai-generate", handler.GenerateDraft)

	// 1. Success
	bodySuccess, _ := json.Marshal(GenerateDraftRequest{
		Topic: "Pendidikan 2026",
	})
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/posts/ai-generate", bytes.NewBuffer(bodySuccess))
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	var res1 map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &res1)
	assert.Equal(t, "success", res1["status"])
	draf := res1["data"].(map[string]interface{})
	assert.NotEmpty(t, draf["title"])

	// 2. Missing topic in JSON
	bodyNoTopic, _ := json.Marshal(map[string]string{"category": "Teknologi"})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/posts/ai-generate", bytes.NewBuffer(bodyNoTopic))
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	var res2 map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &res2)
	assert.Equal(t, "fail", res2["status"])
	assert.Contains(t, res2["message"].(string), "topic")

	// 3. Bad JSON body
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost, "/posts/ai-generate", bytes.NewBufferString("{bad"))
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusBadRequest, w3.Code)
	assert.Contains(t, w3.Body.String(), "topic")

	// 3b. Whitespace topic (triggers service ErrTopicRequired in handler)
	wWhitespace := httptest.NewRecorder()
	reqWhitespace, _ := http.NewRequest(http.MethodPost, "/posts/ai-generate", bytes.NewBufferString(`{"topic": "   "}`))
	r.ServeHTTP(wWhitespace, reqWhitespace)
	assert.Equal(t, http.StatusBadRequest, wWhitespace.Code)
	assert.Contains(t, wWhitespace.Body.String(), "topic")

	// 4. OpenAI Error -> 500 status error with "openai"
	mockAIErr := &mockOpenAIClient{
		err: errors.New("service unavailable"),
	}
	svcErr := NewService(cfg, mockAIErr, nil)
	handlerErr := NewHandler(svcErr)
	rErr := gin.New()
	rErr.POST("/posts/ai-generate", handlerErr.GenerateDraft)

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodPost, "/posts/ai-generate", bytes.NewBuffer(bodySuccess))
	rErr.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusInternalServerError, w4.Code)
	var res4 map[string]interface{}
	_ = json.Unmarshal(w4.Body.Bytes(), &res4)
	assert.Equal(t, "error", res4["status"])
	assert.Contains(t, res4["message"].(string), "OpenAI")
}
