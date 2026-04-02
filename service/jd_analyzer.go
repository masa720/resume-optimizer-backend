package service

import (
	"context"

	"github.com/masa720/resume-optimizer-backend/domain"
)

// JDAnalyzerResult holds the structured analysis of a job description.
type JDAnalyzerResult struct {
	Skills []domain.StructuredSkill `json:"skills"`
}

// JDAnalyzer extracts structured skill data from a job description.
type JDAnalyzer interface {
	Analyze(ctx context.Context, jobDescription string) (*JDAnalyzerResult, error)
}
