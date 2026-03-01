package repository

import (
	"github.com/your-username/resume-optimizer-backend/domain"

	"gorm.io/gorm"
)

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) domain.ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) GetByID(userID string) (*domain.Profile, error) {
	var profile domain.Profile
	if err := r.db.Where("id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) Update(profile *domain.Profile) error {
	return r.db.Save(profile).Error
}
