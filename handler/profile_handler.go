package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/your-username/resume-optimizer-backend/domain"

	"gorm.io/gorm"
)

type ProfileHandler struct {
	profileRepo domain.ProfileRepository
}

func NewProfileHandler(profileRepo domain.ProfileRepository) *ProfileHandler {
	return &ProfileHandler{profileRepo: profileRepo}
}

type updateProfileRequest struct {
	Username string `json:"username" binding:"required"`
}

func (h *ProfileHandler) GetProfile(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	profile, err := h.profileRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch profile"})
		return
	}

	ctx.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) UpdateProfile(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	var req updateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	profile, err := h.profileRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch profile"})
		return
	}

	profile.Username = req.Username

	if err := h.profileRepo.Update(profile); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	ctx.JSON(http.StatusOK, profile)
}

func getUserID(ctx *gin.Context) (string, bool) {
	userID := ctx.GetString("userID")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}
	return userID, true
}
