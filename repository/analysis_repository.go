package repository

import (
	"github.com/masa720/resume-optimizer-backend/domain"

	"gorm.io/gorm"
)

type analysisRepository struct {
	db *gorm.DB
}

func NewAnalysisRepository(db *gorm.DB) domain.AnalysisRepository {
	return &analysisRepository{db: db}
}

func (r *analysisRepository) Create(analysis *domain.Analysis) error {
	return r.db.Create(analysis).Error
}

func (r *analysisRepository) GetByID(userID, analysisID string) (*domain.Analysis, error) {
	var analysis domain.Analysis
	if err := r.db.Where("id = ? AND user_id = ?", analysisID, userID).First(&analysis).Error; err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (r *analysisRepository) GetAllByUserID(userID string) ([]domain.Analysis, error) {
	var analyses []domain.Analysis
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&analyses).Error; err != nil {
		return nil, err
	}
	return analyses, nil
}

func (r *analysisRepository) Delete(userID, analysisID string) error {
	result := r.db.Where("id = ? AND user_id = ?", analysisID, userID).Delete(&domain.Analysis{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
