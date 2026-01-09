package model

import "fmt"

// Task represents a sub-unit within a project for finer categorization.
type Task struct {
	Key         string `json:"key"`
	SID         string `json:"sid" validate:"required,max=32,sid"`
	ProjectSID  string `json:"project_sid" validate:"required,max=32,sid"`
	DisplayName string `json:"display_name" validate:"required,max=64"`
	Color       string `json:"color,omitempty" validate:"omitempty,hexcolor"`
}

// SetKey sets the database key for this task.
func (t *Task) SetKey(key string) {
	t.Key = key
}

// GetKey returns the database key for this task.
func (t *Task) GetKey() string {
	return t.Key
}

// GenerateKey generates a database key for a task using project SID and task SID.
func GenerateTaskKey(projectSID, taskSID string) string {
	return fmt.Sprintf("%s:%s:%s", PrefixTask, projectSID, taskSID)
}

// NewTask creates a new task with the given parameters.
func NewTask(projectSID, sid, displayName, color string) *Task {
	return &Task{
		Key:         GenerateTaskKey(projectSID, sid),
		SID:         sid,
		ProjectSID:  projectSID,
		DisplayName: displayName,
		Color:       color,
	}
}
