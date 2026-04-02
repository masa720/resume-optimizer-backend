package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
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

type createVersionRequest struct {
	JobDescription string `json:"jobDescription" binding:"required"`
	ResumeText     string `json:"resumeText" binding:"required"`
}

type updateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// analysisResult holds the computed analysis data used to create an AnalysisVersion.
type analysisResult struct {
	MatchScore       int
	MatchedKeywords  []string
	MissingKeywords  []string
	Suggestions      []byte
	StructuredSkills []byte
	SectionFeedback  []byte
	FormatChecks     []byte
	Rewrites         []byte
	ScoreBreakdown   []byte
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

	result := h.runAnalysis(ctx.Request.Context(), req.JobDescription, req.ResumeText)
	if result == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze resume"})
		return
	}

	analysis := &domain.Analysis{
		UserID:      userID,
		CompanyName: req.CompanyName,
		JobPosition: req.JobPosition,
	}

	if err := h.analysisRepo.Create(analysis); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create analysis"})
		return
	}

	version := h.buildVersion(analysis.ID, 1, req.JobDescription, req.ResumeText, result)
	if err := h.analysisRepo.CreateVersion(version); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create analysis version"})
		return
	}

	analysis.Versions = []domain.AnalysisVersion{*version}
	ctx.JSON(http.StatusCreated, analysis)
}

func (h *AnalysisHandler) CreateVersion(ctx *gin.Context) {
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

	var req createVersionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result := h.runAnalysis(ctx.Request.Context(), req.JobDescription, req.ResumeText)
	if result == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze resume"})
		return
	}

	maxVersion, err := h.analysisRepo.GetMaxVersion(analysisID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to determine version number"})
		return
	}

	version := h.buildVersion(analysisID, maxVersion+1, req.JobDescription, req.ResumeText, result)
	if err := h.analysisRepo.CreateVersion(version); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create version"})
		return
	}

	analysis.Versions = append(analysis.Versions, *version)
	ctx.JSON(http.StatusCreated, analysis)
}

// allowedSortColumns is a whitelist of columns that can be used for sorting.
var allowedSortColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"status":     true,
}

func (h *AnalysisHandler) List(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	sortBy := ctx.DefaultQuery("sort", "created_at")
	if !allowedSortColumns[sortBy] {
		sortBy = "created_at"
	}

	order := strings.ToLower(ctx.DefaultQuery("order", "desc"))
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	query := domain.ListQuery{
		Page:        page,
		Limit:       limit,
		SortBy:      sortBy,
		Order:       order,
		CompanyName: strings.TrimSpace(ctx.Query("company")),
		JobPosition: strings.TrimSpace(ctx.Query("position")),
		Status:      strings.TrimSpace(ctx.Query("status")),
	}

	result, err := h.analysisRepo.GetAllByUserID(userID, query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch analyses"})
		return
	}

	ctx.JSON(http.StatusOK, result)
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

func (h *AnalysisHandler) UpdateStatus(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	analysisID := ctx.Param("id")

	var req updateStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	if len(req.Status) > 50 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "status must be 50 characters or less"})
		return
	}

	analysis, err := h.analysisRepo.UpdateStatus(userID, analysisID, req.Status)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	ctx.JSON(http.StatusOK, analysis)
}

// buildVersion creates an AnalysisVersion from analysis result data.
func (h *AnalysisHandler) buildVersion(analysisID string, versionNum int, jobDescription, resumeText string, result *analysisResult) *domain.AnalysisVersion {
	return &domain.AnalysisVersion{
		AnalysisID:       analysisID,
		Version:          versionNum,
		JobDescription:   jobDescription,
		ResumeText:       resumeText,
		MatchScore:       result.MatchScore,
		MatchedKeywords:  result.MatchedKeywords,
		MissingKeywords:  result.MissingKeywords,
		Suggestions:      result.Suggestions,
		StructuredSkills: result.StructuredSkills,
		SectionFeedback:  result.SectionFeedback,
		FormatChecks:     result.FormatChecks,
		Rewrites:         result.Rewrites,
		ScoreBreakdown:   result.ScoreBreakdown,
	}
}

// runAnalysis executes the analysis pipeline and returns the computed result data.
// Tries the unified analyzer first, then falls back to JD analyzer + suggestions.
func (h *AnalysisHandler) runAnalysis(ctx context.Context, jobDescription, resumeText string) *analysisResult {
	if h.unifiedAnalyzer != nil {
		result, err := h.unifiedAnalyzer.Analyze(ctx, jobDescription, resumeText)
		if err == nil && result != nil && len(result.Skills) > 0 {
			return buildFromUnified(result)
		}
		if err != nil {
			log.Printf("unified analyzer error, falling back: %v", err)
		}
	}

	return h.buildFromFallback(ctx, jobDescription, resumeText)
}

func buildFromUnified(result *service.UnifiedAnalysisResult) *analysisResult {
	matched, missing := service.ExtractMatchedMissing(result.Skills)
	subScores := service.CalculateSubScores(result.Skills)

	structuredSkillsJSON, _ := json.Marshal(result.Skills)
	sectionFeedbackJSON, _ := json.Marshal(result.SectionFeedback)
	formatChecksJSON, _ := json.Marshal(result.FormatChecks)
	rewritesJSON, _ := json.Marshal(result.Rewrites)
	scoreBreakdownJSON, _ := json.Marshal(subScores)

	suggestions := make([]string, 0, len(result.Rewrites))
	for _, r := range result.Rewrites {
		suggestions = append(suggestions, r.Reason+": "+r.After)
	}
	suggestionsJSON, _ := json.Marshal(suggestions)

	return &analysisResult{
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

func (h *AnalysisHandler) buildFromFallback(ctx context.Context, jobDescription, resumeText string) *analysisResult {
	var matchedKeywords, missingKeywords []string
	var score int
	var structuredSkillsJSON []byte

	jdResult, err := h.jdAnalyzer.Analyze(ctx, jobDescription)
	if err != nil || jdResult == nil || len(jdResult.Skills) == 0 {
		if err != nil {
			log.Printf("JD analyzer error, falling back to keyword extraction: %v", err)
		}
		keywords := service.ExtractKeywords(jobDescription)
		matchedKeywords, missingKeywords = service.MatchKeywords(keywords, resumeText)
		score = service.CalculateMatchScore(len(matchedKeywords), len(matchedKeywords)+len(missingKeywords))
	} else {
		matchedKeywords, missingKeywords = service.MatchStructuredSkills(jdResult.Skills, resumeText)
		matchedSet := make(map[string]bool, len(matchedKeywords))
		for _, kw := range matchedKeywords {
			matchedSet[strings.ToLower(kw)] = true
		}
		score = service.CalculateWeightedMatchScore(jdResult.Skills, matchedSet)
		structuredSkillsJSON, _ = json.Marshal(jdResult.Skills)
	}

	suggestions, err := h.suggestionProvider.Generate(ctx, jobDescription, resumeText, missingKeywords)
	if err != nil {
		log.Printf("suggestion provider error: %v", err)
		suggestions = service.GenerateSuggestions(missingKeywords)
	}

	suggestionsJSON, _ := json.Marshal(suggestions)

	return &analysisResult{
		MatchScore:       score,
		MatchedKeywords:  matchedKeywords,
		MissingKeywords:  missingKeywords,
		Suggestions:      suggestionsJSON,
		StructuredSkills: structuredSkillsJSON,
	}
}
