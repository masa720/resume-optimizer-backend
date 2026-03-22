package domain

import "time"

type Profile struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ProfileRepository interface {
	GetByID(userID string) (*Profile, error)
	Update(profile *Profile) error
}
