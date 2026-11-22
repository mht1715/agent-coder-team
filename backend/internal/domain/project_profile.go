package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProjectProfile struct {
	ProjectID       uuid.UUID `json:"project_id"`
	PrimaryLanguage string    `json:"primary_language"`
	Languages       []string  `json:"languages"`
	Frameworks      []string  `json:"frameworks"`
	PackageManager  string    `json:"package_manager"`
	BuildCommand    string    `json:"build_command"`
	TestCommand     string    `json:"test_command"`
	FormatCommand   string    `json:"format_command"`
	FileCount       int       `json:"file_count"`
	TopLevel        []string  `json:"top_level"`
	Truncated       bool      `json:"truncated"`
	DetectedAt      time.Time `json:"detected_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
