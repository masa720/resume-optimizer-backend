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

type OpenAIUnifiedAnalyzer struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAIUnifiedAnalyzer(apiKey, model, baseURL string) *OpenAIUnifiedAnalyzer {
	if model == "" {
		model = defaultOpenAIModel
	}
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	return &OpenAIUnifiedAnalyzer{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

const unifiedAnalysisSystemPrompt = `You are an expert ATS (Applicant Tracking System) resume analyst. Analyze the given job description and resume, then return a comprehensive JSON analysis.

Return a JSON object with these keys:

1. "skills" - Array of skills extracted from the job description, each with:
   - "name": skill name (preserve multi-word names like "project management")
   - "category": "hard" or "soft" (hard = technical skills, tools, languages, certifications; soft = communication, leadership, teamwork, etc.)
   - "importance": "required" or "preferred" (required = must-have, explicitly required; preferred = nice-to-have, bonus)
   - "matched": true/false - whether the resume demonstrates this skill. Use SEMANTIC matching: "JS" matches "JavaScript", "managed projects" matches "project management", synonyms and abbreviations count as matches
   - "resumeEvidence": if matched, quote the relevant excerpt from the resume (max 30 words). If not matched, empty string ""

2. "sectionFeedback" - Array evaluating each resume section:
   - "section": section name ("Summary", "Experience", "Skills", "Education", or other detected sections)
   - "score": 0-100 quality score for this section
   - "feedback": specific actionable feedback (max 30 words)
   If a section is missing from the resume, include it with score 0 and feedback noting it should be added.

3. "formatChecks" - Array of ATS formatting checks:
   - "item": what was checked (e.g. "Standard section headers", "Action verbs", "Bullet point consistency", "Quantified achievements", "Resume length", "Keyword density", "Contact information")
   - "status": "pass" or "warning"
   - "message": brief explanation (max 20 words)
   Check at least 5 formatting items.

4. "rewrites" - Array of 3-5 before/after rewrite suggestions:
   - "section": which section this applies to
   - "before": the original text from the resume to improve (exact quote or close paraphrase)
   - "after": rewritten text that naturally incorporates missing keywords
   - "reason": why this change improves the resume (max 20 words)

Rules:
- Extract at most 30 skills
- Use semantic matching for skills (abbreviations, synonyms, related terms all count)
- Return ONLY the JSON object, no other text`

func (a *OpenAIUnifiedAnalyzer) Analyze(ctx context.Context, jobDescription, resumeText string) (*UnifiedAnalysisResult, error) {
	userPrompt := fmt.Sprintf("Job Description:\n%s\n\nResume:\n%s", jobDescription, resumeText)

	payload := chatCompletionsRequest{
		Model:       a.model,
		Temperature: 0.2,
		Messages: []chatMessage{
			{Role: "system", Content: unifiedAnalysisSystemPrompt},
			{Role: "user", Content: userPrompt},
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

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)

	var result UnifiedAnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse unified analysis response: %w", err)
	}

	result.Skills = validateSkills(result.Skills)
	result.FormatChecks = validateFormatChecks(result.FormatChecks)
	result.Rewrites = validateRewrites(result.Rewrites)

	return &result, nil
}

func validateSkills(skills []domain.StructuredSkill) []domain.StructuredSkill {
	valid := make([]domain.StructuredSkill, 0, len(skills))
	for _, s := range skills {
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
	return valid
}

func validateFormatChecks(checks []domain.FormatCheck) []domain.FormatCheck {
	valid := make([]domain.FormatCheck, 0, len(checks))
	for _, c := range checks {
		if c.Item == "" {
			continue
		}
		if c.Status != "pass" && c.Status != "warning" {
			c.Status = "warning"
		}
		valid = append(valid, c)
	}
	return valid
}

func validateRewrites(rewrites []domain.RewriteSuggestion) []domain.RewriteSuggestion {
	valid := make([]domain.RewriteSuggestion, 0, len(rewrites))
	for _, r := range rewrites {
		if r.Before == "" || r.After == "" {
			continue
		}
		valid = append(valid, r)
	}
	if len(valid) > 5 {
		valid = valid[:5]
	}
	return valid
}
