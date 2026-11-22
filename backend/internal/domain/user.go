package domain

import (
	"time"

	"github.com/google/uuid"
)

const UserStatusActive = "active"
const UserStatusDisabled = "disabled"

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	DisplayName  string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}
