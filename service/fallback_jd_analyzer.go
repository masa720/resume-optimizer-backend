package service

import "context"

// FallbackJDAnalyzer returns nil so the handler falls back to keyword-based extraction.
type FallbackJDAnalyzer struct{}

func (f *FallbackJDAnalyzer) Analyze(_ context.Context, _ string) (*JDAnalyzerResult, error) {
	return nil, nil
}
