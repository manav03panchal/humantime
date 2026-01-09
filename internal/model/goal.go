package model

import (
	"fmt"
	"time"
)

// GoalType represents the type of goal (daily or weekly).
type GoalType string

const (
	GoalTypeDaily  GoalType = "daily"
	GoalTypeWeekly GoalType = "weekly"
)

// Goal represents a time target for a project.
type Goal struct {
	Key        string        `json:"key"`
	ProjectSID string        `json:"project_sid" validate:"required,max=32,sid"`
	Type       GoalType      `json:"type" validate:"required,oneof=daily weekly"`
	Target     time.Duration `json:"target" validate:"required,gt=0"`
}

// SetKey sets the database key for this goal.
func (g *Goal) SetKey(key string) {
	g.Key = key
}

// GetKey returns the database key for this goal.
func (g *Goal) GetKey() string {
	return g.Key
}

// GenerateKey generates a database key for a goal using project SID.
func GenerateGoalKey(projectSID string) string {
	return fmt.Sprintf("%s:%s", PrefixGoal, projectSID)
}

// NewGoal creates a new goal with the given parameters.
func NewGoal(projectSID string, goalType GoalType, target time.Duration) *Goal {
	return &Goal{
		Key:        GenerateGoalKey(projectSID),
		ProjectSID: projectSID,
		Type:       goalType,
		Target:     target,
	}
}

// TargetSeconds returns the target duration in seconds.
func (g *Goal) TargetSeconds() int64 {
	return int64(g.Target.Seconds())
}

// Progress calculates progress toward the goal.
type Progress struct {
	Current    time.Duration `json:"current_seconds"`
	Remaining  time.Duration `json:"remaining_seconds"`
	Percentage float64       `json:"percentage"`
	IsComplete bool          `json:"is_complete"`
}

// CalculateProgress calculates progress given current tracked time.
func (g *Goal) CalculateProgress(current time.Duration) Progress {
	percentage := float64(current) / float64(g.Target) * 100
	if percentage > 100 {
		percentage = 100 + (percentage - 100)
	}

	remaining := g.Target - current
	if remaining < 0 {
		remaining = 0
	}

	return Progress{
		Current:    current,
		Remaining:  remaining,
		Percentage: percentage,
		IsComplete: current >= g.Target,
	}
}
