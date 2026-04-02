package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/masa720/resume-optimizer-backend/domain"
)

type OpenAIJDAnalyzer struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAIJDAnalyzer(apiKey, model, baseURL string) *OpenAIJDAnalyzer {
	if model == "" {
		model = defaultOpenAIModel
	}
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	return &OpenAIJDAnalyzer{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

const jdAnalyzerSystemPrompt = `You are a job description analyst. Extract all technical and professional skills from the given job description. Classify each skill.

Return a JSON object with a single key "skills" containing an array of objects:
{"name": "skill name", "category": "hard|soft", "importance": "required|preferred"}

Rules:
- "hard" = technical skills, tools, programming languages, certifications
- "soft" = communication, leadership, teamwork, etc.
- "required" = explicitly stated as required, must-have, or listed in requirements section
- "preferred" = nice-to-have, preferred, bonus, or listed in preferred qualifications
- If unclear, default to "required" for hard skills and "preferred" for soft skills
- Extract multi-word skill names as-is (e.g., "project management", not split)
- Return at most 30 skills
- Return ONLY the JSON object, no other text`

func (a *OpenAIJDAnalyzer) Analyze(ctx context.Context, jobDescription string) (*JDAnalyzerResult, error) {
	payload := chatCompletionsRequest{
		Model:       a.model,
		Temperature: 0.2,
		Messages: []chatMessage{
			{Role: "system", Content: jdAnalyzerSystemPrompt},
			{Role: "user", Content: jobDescription},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
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

	content := stripMarkdownCodeFence(parsed.Choices[0].Message.Content)

	var result JDAnalyzerResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JD analysis response: %w", err)
	}

	// Filter out invalid entries
	valid := make([]domain.StructuredSkill, 0, len(result.Skills))
	for _, s := range result.Skills {
		if s.Name == "" {
			continue
		}
		if s.Category != "hard" && s.Category != "soft" {
			s.Category = "hard"
		}
		if s.Importance != "required" && s.Importance != "preferred" {
			s.Importance = "required"
		}
		valid = append(valid, s)
	}
	result.Skills = valid

	return &result, nil
}
