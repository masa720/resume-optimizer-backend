package domain

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Analysis struct {
	ID              string         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          string         `json:"user_id" gorm:"type:uuid;not null"`
	JobDescription  string         `json:"job_description" gorm:"not null"`
	ResumeText      string         `json:"resume_text" gorm:"not null"`
	CompanyName     string         `json:"company_name"`
	JobPosition     string         `json:"job_position"`
	MatchScore      int            `json:"match_score"`
	MatchedKeywords pq.StringArray `json:"matched_keywords" gorm:"type:text[]"`
	MissingKeywords pq.StringArray `json:"missing_keywords" gorm:"type:text[]"`
	Suggestions     datatypes.JSON `json:"suggestions" gorm:"type:jsonb"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type AnalysisRepository interface {
	Create(analysis *Analysis) error
	GetByID(userID, analysisID string) (*Analysis, error)
	GetAllByUserID(userID string) ([]Analysis, error)
	Delete(userID, analysisID string) error
}
