package domain

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// StructuredSkill represents a skill extracted from a job description with classification.
type StructuredSkill struct {
	Name       string `json:"name"`
	Category   string `json:"category"`   // "hard" or "soft"
	Importance string `json:"importance"` // "required" or "preferred"
}

type Analysis struct {
	ID               string         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           string         `json:"userId" gorm:"type:uuid;not null"`
	JobDescription   string         `json:"jobDescription" gorm:"not null"`
	ResumeText       string         `json:"resumeText" gorm:"not null"`
	CompanyName      string         `json:"companyName"`
	JobPosition      string         `json:"jobPosition"`
	MatchScore       int            `json:"matchScore"`
	MatchedKeywords  pq.StringArray `json:"matchedKeywords" gorm:"type:text[]"`
	MissingKeywords  pq.StringArray `json:"missingKeywords" gorm:"type:text[]"`
	Suggestions      datatypes.JSON `json:"suggestions" gorm:"type:jsonb"`
	StructuredSkills datatypes.JSON `json:"structuredSkills,omitempty" gorm:"type:jsonb"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

type AnalysisRepository interface {
	Create(analysis *Analysis) error
	GetByID(userID, analysisID string) (*Analysis, error)
	GetAllByUserID(userID string) ([]Analysis, error)
	Delete(userID, analysisID string) error
}
