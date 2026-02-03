package storage

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/manav03panchal/humantime/internal/model"
)

// BlockRepo provides operations for Block entities.
type BlockRepo struct {
	db *DB
}

// NewBlockRepo creates a new block repository.
func NewBlockRepo(db *DB) *BlockRepo {
	return &BlockRepo{db: db}
}

// Create creates a new block with a generated key.
func (r *BlockRepo) Create(block *model.Block) error {
	// Generate UUID v7 for time-sortable keys
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	block.Key = model.GenerateBlockKey(id.String())
	return r.db.Set(block)
}

// Get retrieves a block by key.
func (r *BlockRepo) Get(key string) (*model.Block, error) {
	block := &model.Block{}
	if err := r.db.Get(key, block); err != nil {
		return nil, err
	}
	return block, nil
}

// Update updates an existing block.
func (r *BlockRepo) Update(block *model.Block) error {
	return r.db.Set(block)
}

// Delete removes a block by key.
func (r *BlockRepo) Delete(key string) error {
	return r.db.Delete(key)
}

// List retrieves all blocks.
func (r *BlockRepo) List() ([]*model.Block, error) {
	return GetAllByPrefix(r.db, model.PrefixBlock+":", func() *model.Block {
		return &model.Block{}
	})
}

// ListByProject retrieves all blocks for a specific project.
// Uses filtered iteration to avoid loading all blocks into memory.
func (r *BlockRepo) ListByProject(projectSID string) ([]*model.Block, error) {
	return GetFilteredByPrefix(r.db, model.PrefixBlock+":", func() *model.Block {
		return &model.Block{}
	}, func(b *model.Block) bool {
		return b.ProjectSID == projectSID
	}, 0)
}

// ListByProjectAndTask retrieves all blocks for a specific project and task.
// Uses filtered iteration to avoid loading all blocks into memory.
func (r *BlockRepo) ListByProjectAndTask(projectSID, taskSID string) ([]*model.Block, error) {
	return GetFilteredByPrefix(r.db, model.PrefixBlock+":", func() *model.Block {
		return &model.Block{}
	}, func(b *model.Block) bool {
		return b.ProjectSID == projectSID && b.TaskSID == taskSID
	}, 0)
}

// ListByTimeRange retrieves blocks within a time range.
// Uses filtered iteration to avoid loading all blocks into memory.
func (r *BlockRepo) ListByTimeRange(start, end time.Time) ([]*model.Block, error) {
	return GetFilteredByPrefix(r.db, model.PrefixBlock+":", func() *model.Block {
		return &model.Block{}
	}, func(b *model.Block) bool {
		// Block overlaps with range if:
		// - Block starts before range ends AND
		// - Block ends after range starts (or is still active)
		blockEnd := b.TimestampEnd
		if blockEnd.IsZero() {
			blockEnd = time.Now()
		}
		return b.TimestampStart.Before(end) && blockEnd.After(start)
	}, 0)
}

// BlockFilter defines filtering criteria for blocks.
type BlockFilter struct {
	ProjectSID string
	TaskSID    string
	Tag        string
	StartAfter time.Time
	EndBefore  time.Time
	Limit      int
}

// ListFiltered retrieves blocks matching the filter criteria.
// Uses filtered iteration to avoid loading all blocks into memory before filtering.
// Note: Sorting is still done in memory since BadgerDB uses lexicographical key order.
func (r *BlockRepo) ListFiltered(filter BlockFilter) ([]*model.Block, error) {
	// Build filter function from criteria
	filterFunc := func(b *model.Block) bool {
		// Apply project filter
		if filter.ProjectSID != "" && b.ProjectSID != filter.ProjectSID {
			return false
		}

		// Apply task filter
		if filter.TaskSID != "" && b.TaskSID != filter.TaskSID {
			return false
		}

		// Apply tag filter
		if filter.Tag != "" && !b.HasTag(filter.Tag) {
			return false
		}

		// Apply time range filters
		if !filter.StartAfter.IsZero() && b.TimestampStart.Before(filter.StartAfter) {
			return false
		}

		blockEnd := b.TimestampEnd
		if blockEnd.IsZero() {
			blockEnd = time.Now()
		}
		if !filter.EndBefore.IsZero() && blockEnd.After(filter.EndBefore) {
			return false
		}

		return true
	}

	// Use filtered iteration - can't apply limit here since we need to sort first
	filtered, err := GetFilteredByPrefix(r.db, model.PrefixBlock+":", func() *model.Block {
		return &model.Block{}
	}, filterFunc, 0)
	if err != nil {
		return nil, err
	}

	// Sort by start time (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].TimestampStart.After(filtered[j].TimestampStart)
	})

	// Apply limit after sorting
	if filter.Limit > 0 && len(filtered) > filter.Limit {
		filtered = filtered[:filter.Limit]
	}

	return filtered, nil
}

// TotalDuration calculates the total duration of given blocks.
func TotalDuration(blocks []*model.Block) time.Duration {
	var total time.Duration
	for _, b := range blocks {
		total += b.Duration()
	}
	return total
}

// AggregateByProject groups blocks by project and calculates totals.
type ProjectAggregate struct {
	ProjectSID string
	Duration   time.Duration
	BlockCount int
}

// AggregateByProject aggregates blocks by project.
func AggregateByProject(blocks []*model.Block) []ProjectAggregate {
	agg := make(map[string]*ProjectAggregate)

	for _, b := range blocks {
		if _, ok := agg[b.ProjectSID]; !ok {
			agg[b.ProjectSID] = &ProjectAggregate{
				ProjectSID: b.ProjectSID,
			}
		}
		agg[b.ProjectSID].Duration += b.Duration()
		agg[b.ProjectSID].BlockCount++
	}

	var result []ProjectAggregate
	for _, a := range agg {
		result = append(result, *a)
	}

	// Sort by duration (highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Duration > result[j].Duration
	})

	return result
}
