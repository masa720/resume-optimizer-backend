package service

import (
	"context"

	"github.com/masa720/resume-optimizer-backend/domain"
)

// UnifiedAnalysisResult holds all analysis results from a single OpenAI call.
type UnifiedAnalysisResult struct {
	Skills          []domain.StructuredSkill   `json:"skills"`
	SectionFeedback []domain.SectionFeedback   `json:"sectionFeedback"`
	FormatChecks    []domain.FormatCheck       `json:"formatChecks"`
	Rewrites        []domain.RewriteSuggestion `json:"rewrites"`
}

// UnifiedAnalyzer performs full resume analysis in a single call.
type UnifiedAnalyzer interface {
	Analyze(ctx context.Context, jobDescription, resumeText string) (*UnifiedAnalysisResult, error)
}
