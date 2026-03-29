package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/masa720/resume-optimizer-backend/domain"
	"github.com/masa720/resume-optimizer-backend/service"
)

type AnalysisHandler struct {
	analysisRepo       domain.AnalysisRepository
	suggestionProvider service.SuggestionProvider
	jdAnalyzer         service.JDAnalyzer
	unifiedAnalyzer    service.UnifiedAnalyzer
}

func NewAnalysisHandler(analysisRepo domain.AnalysisRepository, suggestionProvider service.SuggestionProvider, jdAnalyzer service.JDAnalyzer, unifiedAnalyzer service.UnifiedAnalyzer) *AnalysisHandler {
	return &AnalysisHandler{
		analysisRepo:       analysisRepo,
		suggestionProvider: suggestionProvider,
		jdAnalyzer:         jdAnalyzer,
		unifiedAnalyzer:    unifiedAnalyzer,
	}
}

type createAnalysisRequest struct {
	JobDescription string `json:"jobDescription" binding:"required"`
	ResumeText     string `json:"resumeText" binding:"required"`
	CompanyName    string `json:"companyName"`
	JobPosition    string `json:"jobPosition"`
}

func (h *AnalysisHandler) Create(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	var req createAnalysisRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	analysis := h.buildAnalysis(ctx.Request.Context(), userID, req)
	if analysis == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze resume"})
		return
	}

	if err := h.analysisRepo.Create(analysis); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create analysis"})
		return
	}

	ctx.JSON(http.StatusCreated, analysis)
}

func (h *AnalysisHandler) List(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	analyses, err := h.analysisRepo.GetAllByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch analyses"})
		return
	}

	ctx.JSON(http.StatusOK, analyses)
}

func (h *AnalysisHandler) GetByID(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	analysisID := ctx.Param("id")

	analysis, err := h.analysisRepo.GetByID(userID, analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch analysis"})
		return
	}

	ctx.JSON(http.StatusOK, analysis)
}

func (h *AnalysisHandler) Delete(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	analysisID := ctx.Param("id")

	if err := h.analysisRepo.Delete(userID, analysisID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete analysis"})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// buildAnalysis tries the unified analyzer first, then falls back to the #3 JD analyzer + suggestions.
func (h *AnalysisHandler) buildAnalysis(ctx context.Context, userID string, req createAnalysisRequest) *domain.Analysis {
	// Try unified analysis (single OpenAI call)
	if h.unifiedAnalyzer != nil {
		result, err := h.unifiedAnalyzer.Analyze(ctx, req.JobDescription, req.ResumeText)
		if err == nil && result != nil && len(result.Skills) > 0 {
			return h.buildFromUnified(userID, req, result)
		}
		if err != nil {
			log.Printf("unified analyzer error, falling back: %v", err)
		}
	}

	// Fallback: #3 JD analyzer + separate suggestion call
	return h.buildFromFallback(ctx, userID, req)
}

func (h *AnalysisHandler) buildFromUnified(userID string, req createAnalysisRequest, result *service.UnifiedAnalysisResult) *domain.Analysis {
	matched, missing := service.ExtractMatchedMissing(result.Skills)
	subScores := service.CalculateSubScores(result.Skills)

	structuredSkillsJSON, _ := json.Marshal(result.Skills)
	sectionFeedbackJSON, _ := json.Marshal(result.SectionFeedback)
	formatChecksJSON, _ := json.Marshal(result.FormatChecks)
	rewritesJSON, _ := json.Marshal(result.Rewrites)
	scoreBreakdownJSON, _ := json.Marshal(subScores)

	// Build suggestion strings from rewrites for backward compatibility
	suggestions := make([]string, 0, len(result.Rewrites))
	for _, r := range result.Rewrites {
		suggestions = append(suggestions, r.Reason+": "+r.After)
	}
	suggestionsJSON, _ := json.Marshal(suggestions)

	return &domain.Analysis{
		UserID:           userID,
		JobDescription:   req.JobDescription,
		ResumeText:       req.ResumeText,
		CompanyName:      req.CompanyName,
		JobPosition:      req.JobPosition,
		MatchScore:       subScores.Overall,
		MatchedKeywords:  matched,
		MissingKeywords:  missing,
		Suggestions:      suggestionsJSON,
		StructuredSkills: structuredSkillsJSON,
		SectionFeedback:  sectionFeedbackJSON,
		FormatChecks:     formatChecksJSON,
		Rewrites:         rewritesJSON,
		ScoreBreakdown:   scoreBreakdownJSON,
	}
}

func (h *AnalysisHandler) buildFromFallback(ctx context.Context, userID string, req createAnalysisRequest) *domain.Analysis {
	var matchedKeywords, missingKeywords []string
	var score int
	var structuredSkillsJSON []byte

	jdResult, err := h.jdAnalyzer.Analyze(ctx, req.JobDescription)
	if err != nil || jdResult == nil || len(jdResult.Skills) == 0 {
		if err != nil {
			log.Printf("JD analyzer error, falling back to keyword extraction: %v", err)
		}
		keywords := service.ExtractKeywords(req.JobDescription)
		matchedKeywords, missingKeywords = service.MatchKeywords(keywords, req.ResumeText)
		score = service.CalculateMatchScore(len(matchedKeywords), len(matchedKeywords)+len(missingKeywords))
	} else {
		matchedKeywords, missingKeywords = service.MatchStructuredSkills(jdResult.Skills, req.ResumeText)
		matchedSet := make(map[string]bool, len(matchedKeywords))
		for _, kw := range matchedKeywords {
			matchedSet[strings.ToLower(kw)] = true
		}
		score = service.CalculateWeightedMatchScore(jdResult.Skills, matchedSet)
		structuredSkillsJSON, _ = json.Marshal(jdResult.Skills)
	}

	suggestions, err := h.suggestionProvider.Generate(ctx, req.JobDescription, req.ResumeText, missingKeywords)
	if err != nil {
		log.Printf("suggestion provider error: %v", err)
		suggestions = service.GenerateSuggestions(missingKeywords)
	}

	suggestionsJSON, _ := json.Marshal(suggestions)

	return &domain.Analysis{
		UserID:           userID,
		JobDescription:   req.JobDescription,
		ResumeText:       req.ResumeText,
		CompanyName:      req.CompanyName,
		JobPosition:      req.JobPosition,
		MatchScore:       score,
		MatchedKeywords:  matchedKeywords,
		MissingKeywords:  missingKeywords,
		Suggestions:      suggestionsJSON,
		StructuredSkills: structuredSkillsJSON,
	}
}
