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
	if err := r.db.
		Preload("Versions", func(db *gorm.DB) *gorm.DB {
			return db.Order("version ASC")
		}).
		Where("id = ? AND user_id = ?", analysisID, userID).
		First(&analysis).Error; err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (r *analysisRepository) buildListBase(userID string, query domain.ListQuery) *gorm.DB {
	base := r.db.Where("user_id = ?", userID)

	// company and position are OR search (match either)
	if query.CompanyName != "" && query.JobPosition != "" {
		base = base.Where("company_name ILIKE ? OR job_position ILIKE ?", "%"+query.CompanyName+"%", "%"+query.JobPosition+"%")
	} else if query.CompanyName != "" {
		base = base.Where("company_name ILIKE ?", "%"+query.CompanyName+"%")
	} else if query.JobPosition != "" {
		base = base.Where("job_position ILIKE ?", "%"+query.JobPosition+"%")
	}

	// status is AND filter (exact match)
	if query.Status != "" {
		base = base.Where("status = ?", query.Status)
	}
	return base
}

func (r *analysisRepository) GetAllByUserID(userID string, query domain.ListQuery) (*domain.ListResult, error) {
	base := r.buildListBase(userID, query)

	var totalCount int64
	if err := base.Session(&gorm.Session{}).Model(&domain.Analysis{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	orderClause := query.SortBy + " " + query.Order
	offset := (query.Page - 1) * query.Limit

	var analyses []domain.Analysis
	if err := base.Session(&gorm.Session{}).
		Preload("Versions", func(db *gorm.DB) *gorm.DB {
			return db.Where("version = (SELECT MAX(av.version) FROM analysis_versions av WHERE av.analysis_id = analysis_versions.analysis_id AND av.deleted_at IS NULL)")
		}).
		Order(orderClause).
		Limit(query.Limit).
		Offset(offset).
		Find(&analyses).Error; err != nil {
		return nil, err
	}

	totalPages := int(totalCount) / query.Limit
	if int(totalCount)%query.Limit != 0 {
		totalPages++
	}

	return &domain.ListResult{
		Data:       analyses,
		TotalCount: totalCount,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *analysisRepository) Delete(userID, analysisID string) error {
	result := r.db.Where("id = ? AND user_id = ?", analysisID, userID).Delete(&domain.Analysis{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *analysisRepository) UpdateStatus(userID, analysisID, status string) (*domain.Analysis, error) {
	var analysis domain.Analysis
	if err := r.db.Where("id = ? AND user_id = ?", analysisID, userID).First(&analysis).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&analysis).Update("status", status).Error; err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (r *analysisRepository) CreateVersion(version *domain.AnalysisVersion) error {
	return r.db.Create(version).Error
}

func (r *analysisRepository) GetMaxVersion(analysisID string) (int, error) {
	var maxVersion int
	err := r.db.Model(&domain.AnalysisVersion{}).
		Where("analysis_id = ?", analysisID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error
	return maxVersion, err
}
