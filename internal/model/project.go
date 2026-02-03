package model

import (
	"fmt"
	"regexp"
)

// Project represents a top-level organizational unit for time tracking.
type Project struct {
	Key         string `json:"key"`
	SID         string `json:"sid" validate:"required,max=32,sid"`
	DisplayName string `json:"display_name" validate:"required,max=64"`
	Color       string `json:"color,omitempty" validate:"omitempty,hexcolor"`
	Archived    bool   `json:"archived,omitempty"`
}

// SetKey sets the database key for this project.
func (p *Project) SetKey(key string) {
	p.Key = key
}

// GetKey returns the database key for this project.
func (p *Project) GetKey() string {
	return p.Key
}

// GenerateKey generates a database key for a project using its SID.
func GenerateProjectKey(sid string) string {
	return fmt.Sprintf("%s:%s", PrefixProject, sid)
}

// NewProject creates a new project with the given parameters.
func NewProject(sid, displayName, color string) *Project {
	return &Project{
		Key:         GenerateProjectKey(sid),
		SID:         sid,
		DisplayName: displayName,
		Color:       color,
	}
}

// hexColorRegex validates hex color format.
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// ValidateColor checks if a color string is a valid hex color.
func ValidateColor(color string) bool {
	if color == "" {
		return true
	}
	return hexColorRegex.MatchString(color)
}
