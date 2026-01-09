package output

import (
	"time"

	"github.com/manav03panchal/humantime/internal/model"
)

// JSONFormatter provides JSON-specific formatting.
type JSONFormatter struct {
	*Formatter
}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter(f *Formatter) *JSONFormatter {
	return &JSONFormatter{Formatter: f}
}

// StatusResponse represents the status output in JSON.
type StatusResponse struct {
	Status      string       `json:"status"`
	ActiveBlock *BlockOutput `json:"active_block,omitempty"`
}

// BlockOutput represents a block in JSON output.
type BlockOutput struct {
	Key             string  `json:"key"`
	ProjectSID      string  `json:"project_sid"`
	TaskSID         string  `json:"task_sid,omitempty"`
	Note            string  `json:"note,omitempty"`
	TimestampStart  string  `json:"timestamp_start"`
	TimestampEnd    string  `json:"timestamp_end,omitempty"`
	DurationSeconds int64   `json:"duration_seconds"`
	IsActive        bool    `json:"is_active"`
}

// NewBlockOutput creates a BlockOutput from a Block.
func NewBlockOutput(b *model.Block) *BlockOutput {
	out := &BlockOutput{
		Key:             b.Key,
		ProjectSID:      b.ProjectSID,
		TaskSID:         b.TaskSID,
		Note:            b.Note,
		TimestampStart:  b.TimestampStart.Format(time.RFC3339),
		DurationSeconds: b.DurationSeconds(),
		IsActive:        b.IsActive(),
	}
	if !b.TimestampEnd.IsZero() {
		out.TimestampEnd = b.TimestampEnd.Format(time.RFC3339)
	}
	return out
}

// StartResponse represents the start command output in JSON.
type StartResponse struct {
	Status        string       `json:"status"`
	Block         *BlockOutput `json:"block"`
	PreviousBlock *BlockOutput `json:"previous_block,omitempty"`
}

// StopResponse represents the stop command output in JSON.
type StopResponse struct {
	Status string       `json:"status"`
	Block  *BlockOutput `json:"block"`
}

// ErrorResponse represents an error in JSON.
type ErrorResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// BlocksResponse represents the blocks list output in JSON.
type BlocksResponse struct {
	Blocks               []*BlockOutput `json:"blocks"`
	TotalCount           int            `json:"total_count"`
	ShownCount           int            `json:"shown_count"`
	TotalDurationSeconds int64          `json:"total_duration_seconds"`
}

// NewBlocksResponse creates a BlocksResponse from blocks.
func NewBlocksResponse(blocks []*model.Block, total int) *BlocksResponse {
	outputs := make([]*BlockOutput, len(blocks))
	var totalDuration int64
	for i, b := range blocks {
		outputs[i] = NewBlockOutput(b)
		totalDuration += b.DurationSeconds()
	}
	return &BlocksResponse{
		Blocks:               outputs,
		TotalCount:           total,
		ShownCount:           len(blocks),
		TotalDurationSeconds: totalDuration,
	}
}

// ProjectOutput represents a project in JSON output.
type ProjectOutput struct {
	SID                  string        `json:"sid"`
	DisplayName          string        `json:"display_name"`
	Color                string        `json:"color,omitempty"`
	TotalDurationSeconds int64         `json:"total_duration_seconds"`
	Tasks                []*TaskOutput `json:"tasks,omitempty"`
}

// TaskOutput represents a task in JSON output.
type TaskOutput struct {
	SID                  string `json:"sid"`
	ProjectSID           string `json:"project_sid"`
	DisplayName          string `json:"display_name"`
	Color                string `json:"color,omitempty"`
	TotalDurationSeconds int64  `json:"total_duration_seconds"`
}

// NewProjectOutput creates a ProjectOutput from a Project.
func NewProjectOutput(p *model.Project, duration time.Duration) *ProjectOutput {
	return &ProjectOutput{
		SID:                  p.SID,
		DisplayName:          p.DisplayName,
		Color:                p.Color,
		TotalDurationSeconds: int64(duration.Seconds()),
	}
}

// NewTaskOutput creates a TaskOutput from a Task.
func NewTaskOutput(t *model.Task, duration time.Duration) *TaskOutput {
	return &TaskOutput{
		SID:                  t.SID,
		ProjectSID:           t.ProjectSID,
		DisplayName:          t.DisplayName,
		Color:                t.Color,
		TotalDurationSeconds: int64(duration.Seconds()),
	}
}

// ProjectsResponse represents the projects list output in JSON.
type ProjectsResponse struct {
	Projects []*ProjectOutput `json:"projects"`
}

// StatsResponse represents statistics output in JSON.
type StatsResponse struct {
	Period  *PeriodOutput   `json:"period"`
	Summary *SummaryOutput  `json:"summary"`
	Groups  []*GroupOutput  `json:"groups"`
}

// PeriodOutput represents a time period in JSON.
type PeriodOutput struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Grouping string `json:"grouping"`
}

// SummaryOutput represents a statistics summary.
type SummaryOutput struct {
	TotalDurationSeconds int64                    `json:"total_duration_seconds"`
	ByProject            []*ProjectSummaryOutput  `json:"by_project"`
}

// ProjectSummaryOutput represents a project summary.
type ProjectSummaryOutput struct {
	ProjectSID      string             `json:"project_sid"`
	DurationSeconds int64              `json:"duration_seconds"`
	Percentage      float64            `json:"percentage"`
	ByTask          []*TaskSummaryOutput `json:"by_task,omitempty"`
}

// TaskSummaryOutput represents a task summary.
type TaskSummaryOutput struct {
	TaskSID         string `json:"task_sid"`
	DurationSeconds int64  `json:"duration_seconds"`
}

// GroupOutput represents a time group (day, week, month).
type GroupOutput struct {
	Label           string `json:"label"`
	Start           string `json:"start"`
	End             string `json:"end"`
	DurationSeconds int64  `json:"duration_seconds"`
}

// PrintStatus outputs status in JSON format.
func (j *JSONFormatter) PrintStatus(block *model.Block) error {
	resp := StatusResponse{
		Status: "idle",
	}
	if block != nil {
		resp.Status = "tracking"
		resp.ActiveBlock = NewBlockOutput(block)
	}
	return j.JSON(resp)
}

// PrintStart outputs start response in JSON format.
func (j *JSONFormatter) PrintStart(block *model.Block, previous *model.Block) error {
	resp := StartResponse{
		Status: "started",
		Block:  NewBlockOutput(block),
	}
	if previous != nil {
		resp.PreviousBlock = NewBlockOutput(previous)
	}
	return j.JSON(resp)
}

// PrintStop outputs stop response in JSON format.
func (j *JSONFormatter) PrintStop(block *model.Block) error {
	resp := StopResponse{
		Status: "stopped",
		Block:  NewBlockOutput(block),
	}
	return j.JSON(resp)
}

// PrintError outputs an error in JSON format.
func (j *JSONFormatter) PrintError(status, errMsg, message string) error {
	resp := ErrorResponse{
		Status:  status,
		Error:   errMsg,
		Message: message,
	}
	return j.JSON(resp)
}

// PrintBlocks outputs blocks in JSON format.
func (j *JSONFormatter) PrintBlocks(blocks []*model.Block, total int) error {
	return j.JSON(NewBlocksResponse(blocks, total))
}
