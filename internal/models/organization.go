package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Organization struct {
	ID        string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string    `json:"name" gorm:"not null"`
	Plan      string    `json:"plan" gorm:"default:free"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations (temporarily disabled for migration)
	// Users []User `json:"users" gorm:"foreignKey:OrganizationID"`
}

type User struct {
	ID             string    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email          string    `json:"email" gorm:"unique;not null"`
	Name           string    `json:"name" gorm:"not null"`
	OrganizationID string    `json:"organization_id" gorm:"not null"`
	Role           UserRole  `json:"role" gorm:"default:MEMBER"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relations (temporarily disabled for migration)
	// Organization Organization `json:"organization" gorm:"foreignKey:OrganizationID"`
}

type UserRole string

const (
	UserRoleAdmin  UserRole = "ADMIN"
	UserRoleMember UserRole = "MEMBER"
	UserRoleViewer UserRole = "VIEWER"
)

func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}
