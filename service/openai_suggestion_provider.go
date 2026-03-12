package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lib/pq"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
)

type OpenAISuggestionProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAISuggestionProvider(apiKey string, model string, baseURL string) *OpenAISuggestionProvider {
	if model == "" {
		model = defaultOpenAIModel
	}
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	return &OpenAISuggestionProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type chatCompletionsRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (p *OpenAISuggestionProvider) Generate(ctx context.Context, jobDescription string, resumeText string, missingKeywords pq.StringArray) ([]string, error) {
	prompt := buildSuggestionPrompt(jobDescription, resumeText, missingKeywords)

	payload := chatCompletionsRequest{
		Model:       p.model,
		Temperature: 0.4,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "You are a resume coach. Return 3 to 5 concise resume improvement suggestions."},
			{
				Role:    "user",
				Content: prompt},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OpenAI API error: %d", resp.StatusCode)
	}

	var parsed chatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	suggestions := parseSuggestions(parsed.Choices[0].Message.Content)
	if len(suggestions) == 0 {
		return nil, fmt.Errorf("no suggestions parsed from OpenAI response")
	}
	if len(suggestions) > 5 {
		return suggestions[:5], nil
	}
	return suggestions, nil
}

func buildSuggestionPrompt(jobDescription, resumeText string, missingKeywords pq.StringArray) string {
	missing := strings.Join(missingKeywords, ", ")
	if missing == "" {
		missing = "None"
	}

	return fmt.Sprintf(
		`Job Description:
		%s

		Resume Text:
		%s

		Missing Keywords:
		%s

		Provide 3 to 5 bullet-point suggestions. Keep each suggestion under 20 words.`,
		jobDescription, resumeText, missing,
	)
}

func parseSuggestions(content string) []string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, 5)

	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		s = strings.TrimPrefix(s, "-")
		s = strings.TrimSpace(s)
		s = strings.TrimLeft(s, "0123456789. ")
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
		if len(out) >= 5 {
			break
		}
	}

	return out
}
