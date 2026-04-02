package domain

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// StructuredSkill represents a skill extracted from a JD with match result.
type StructuredSkill struct {
	Name           string `json:"name"`
	Category       string `json:"category"`   // "hard" or "soft"
	Importance     string `json:"importance"` // "required" or "preferred"
	Matched        bool   `json:"matched"`
	ResumeEvidence string `json:"resumeEvidence"` // excerpt from resume that matches, empty if not matched
}

// SectionFeedback represents evaluation of a resume section.
type SectionFeedback struct {
	Section  string `json:"section"` // e.g. "Summary", "Experience", "Skills", "Education"
	Score    int    `json:"score"`   // 0-100
	Feedback string `json:"feedback"`
}

// FormatCheck represents an ATS formatting warning.
type FormatCheck struct {
	Item    string `json:"item"`   // what was checked
	Status  string `json:"status"` // "pass" or "warning"
	Message string `json:"message"`
}

// RewriteSuggestion represents a before/after rewrite suggestion.
type RewriteSuggestion struct {
	Section string `json:"section"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Reason  string `json:"reason"`
}

// SubScores represents category-level match scores.
type SubScores struct {
	HardSkillRequired  int `json:"hardSkillRequired"`  // 0-100
	HardSkillPreferred int `json:"hardSkillPreferred"` // 0-100
	SoftSkill          int `json:"softSkill"`          // 0-100
	Overall            int `json:"overall"`            // 0-100
}

// Analysis is the parent entity that holds metadata. All result data lives in AnalysisVersion.
type Analysis struct {
	ID          string            `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      string            `json:"userId" gorm:"type:uuid;not null"`
	CompanyName string            `json:"companyName"`
	JobPosition string            `json:"jobPosition"`
	Status      string            `json:"status" gorm:"type:varchar(20);not null;default:'not_applied'"`
	Versions    []AnalysisVersion `json:"versions" gorm:"foreignKey:AnalysisID;constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt    `json:"-" gorm:"index"`
}

// AnalysisVersion stores a single version of analysis results for an Analysis.
type AnalysisVersion struct {
	ID               string         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AnalysisID       string         `json:"analysisId" gorm:"type:uuid;not null;uniqueIndex:idx_analysis_version"`
	Version          int            `json:"version" gorm:"not null;uniqueIndex:idx_analysis_version"`
	JobDescription   string         `json:"jobDescription" gorm:"not null"`
	ResumeText       string         `json:"resumeText" gorm:"not null"`
	MatchScore       int            `json:"matchScore"`
	MatchedKeywords  pq.StringArray `json:"matchedKeywords" gorm:"type:text[]"`
	MissingKeywords  pq.StringArray `json:"missingKeywords" gorm:"type:text[]"`
	Suggestions      datatypes.JSON `json:"suggestions" gorm:"type:jsonb"`
	StructuredSkills datatypes.JSON `json:"structuredSkills,omitempty" gorm:"type:jsonb"`
	SectionFeedback  datatypes.JSON `json:"sectionFeedback,omitempty" gorm:"type:jsonb"`
	FormatChecks     datatypes.JSON `json:"formatChecks,omitempty" gorm:"type:jsonb"`
	Rewrites         datatypes.JSON `json:"rewrites,omitempty" gorm:"type:jsonb"`
	ScoreBreakdown   datatypes.JSON `json:"scoreBreakdown,omitempty" gorm:"type:jsonb"`
	CreatedAt        time.Time      `json:"createdAt"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// ListQuery holds pagination, sorting, and filtering parameters for listing analyses.
type ListQuery struct {
	Page        int    // 1-based page number
	Limit       int    // items per page
	SortBy      string // column to sort by (e.g. "created_at", "updated_at")
	Order       string // "asc" or "desc"
	CompanyName string // partial match filter
	JobPosition string // partial match filter
	Status      string // exact match filter
}

// ListResult holds paginated analysis results.
type ListResult struct {
	Data       []Analysis `json:"data"`
	TotalCount int64      `json:"totalCount"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"totalPages"`
}

type AnalysisRepository interface {
	Create(analysis *Analysis) error
	GetByID(userID, analysisID string) (*Analysis, error)
	GetAllByUserID(userID string, query ListQuery) (*ListResult, error)
	Delete(userID, analysisID string) error
	UpdateStatus(userID, analysisID, status string) (*Analysis, error)
	CreateVersion(version *AnalysisVersion) error
	GetMaxVersion(analysisID string) (int, error)
}
