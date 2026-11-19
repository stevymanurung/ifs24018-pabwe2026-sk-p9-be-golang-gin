package ai

type GenerateDraftRequest struct {
	Topic            string `json:"topic"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}

type SearchResultItem struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

type DraftResponse struct {
	Title    string             `json:"title"`
	Category string             `json:"category"`
	Excerpt  string             `json:"excerpt"`
	Content  string             `json:"content"`
	Sources  []SearchResultItem `json:"sources,omitempty"`
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
}

type OpenAIChatChoice struct {
	Message OpenAIChatMessage `json:"message"`
}

type OpenAIChatResponse struct {
	Choices []OpenAIChatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
