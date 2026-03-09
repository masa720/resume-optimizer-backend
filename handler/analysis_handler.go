package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/your-username/resume-optimizer-backend/domain"
)

type AnalysisHandler struct {
	analysisRepo domain.AnalysisRepository
}

func NewAnalysisHandler(analysisRepo domain.AnalysisRepository) *AnalysisHandler {
	return &AnalysisHandler{analysisRepo: analysisRepo}
}

type createAnalysisRequest struct {
	JobDescription string `json:"job_description" binding:"required"`
	ResumeText     string `json:"resume_text" binding:"required"`
	CompnayName    string `json:"company_name"`
	JobPosition    string `json:"job_position"`
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

	analisis := &domain.Analysis{
		UserID:         userID,
		JobDescription: req.JobDescription,
		ResumeText:     req.ResumeText,
		CompanyName:    req.CompnayName,
		JobPosition:    req.JobPosition,
		MatchScore:     0,
	}

	if err := h.analysisRepo.Create(analisis); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create analysis"})
		return
	}

	ctx.JSON(http.StatusCreated, analisis)
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
		ctx.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
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
		ctx.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func getUserID(ctx *gin.Context) (string, bool) {
	userID := ctx.GetString("userID")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}
	return userID, true
}
