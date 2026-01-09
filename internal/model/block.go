package model

import (
	"fmt"
	"time"
)

// Block represents a tracked time period.
type Block struct {
	Key            string    `json:"key"`
	OwnerKey       string    `json:"owner_key" validate:"required"`
	ProjectSID     string    `json:"project_sid" validate:"required,max=32"`
	TaskSID        string    `json:"task_sid,omitempty" validate:"max=32"`
	Note           string    `json:"note,omitempty" validate:"max=65536"`
	TimestampStart time.Time `json:"timestamp_start" validate:"required"`
	TimestampEnd   time.Time `json:"timestamp_end,omitempty"`
}

// SetKey sets the database key for this block.
func (b *Block) SetKey(key string) {
	b.Key = key
}

// GetKey returns the database key for this block.
func (b *Block) GetKey() string {
	return b.Key
}

// IsActive returns true if the block has no end time (currently tracking).
func (b *Block) IsActive() bool {
	return b.TimestampEnd.IsZero()
}

// Duration returns the duration of the block.
// If the block is active, it returns the duration from start until now.
func (b *Block) Duration() time.Duration {
	if b.IsActive() {
		return time.Since(b.TimestampStart)
	}
	return b.TimestampEnd.Sub(b.TimestampStart)
}

// DurationSeconds returns the duration in seconds.
func (b *Block) DurationSeconds() int64 {
	return int64(b.Duration().Seconds())
}

// GenerateKey generates a database key for a block using UUID v7.
func GenerateBlockKey(uuid string) string {
	return fmt.Sprintf("%s:%s", PrefixBlock, uuid)
}

// NewBlock creates a new block with the given parameters.
func NewBlock(ownerKey, projectSID, taskSID, note string, start time.Time) *Block {
	return &Block{
		OwnerKey:       ownerKey,
		ProjectSID:     projectSID,
		TaskSID:        taskSID,
		Note:           note,
		TimestampStart: start,
	}
}
