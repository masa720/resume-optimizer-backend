package service

import (
	"context"

	"github.com/lib/pq"
)

type SuggestionProvider interface {
	Generate(ctx context.Context, jobDescription string, resumeText string, missingKeywords pq.StringArray) ([]string, error)
}

type TemplateSuggestionProvider struct{}

func NewTemplateSuggestionProvider() *TemplateSuggestionProvider {
	return &TemplateSuggestionProvider{}
}

func (p *TemplateSuggestionProvider) Generate(_ context.Context, _ string, _ string, missingKeywords pq.StringArray) ([]string, error) {
	return GenerateSuggestions(missingKeywords), nil
}
