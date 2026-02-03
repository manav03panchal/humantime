package storage

import (
	"github.com/manav03panchal/humantime/internal/model"
)

// ActiveBlockRepo provides operations for the ActiveBlock singleton.
type ActiveBlockRepo struct {
	db *DB
}

// NewActiveBlockRepo creates a new active block repository.
func NewActiveBlockRepo(db *DB) *ActiveBlockRepo {
	return &ActiveBlockRepo{db: db}
}

// Get retrieves the active block state.
func (r *ActiveBlockRepo) Get() (*model.ActiveBlock, error) {
	active := model.NewActiveBlock()
	err := r.db.Get(model.KeyActiveBlock, active)
	if err != nil {
		if IsErrKeyNotFound(err) {
			// Return empty active block if not found
			return active, nil
		}
		return nil, err
	}
	return active, nil
}

// Save persists the active block state.
func (r *ActiveBlockRepo) Save(active *model.ActiveBlock) error {
	active.Key = model.KeyActiveBlock
	return r.db.Set(active)
}

// GetActiveBlock retrieves the currently active block, if any.
func (r *ActiveBlockRepo) GetActiveBlock(blockRepo *BlockRepo) (*model.Block, error) {
	active, err := r.Get()
	if err != nil {
		return nil, err
	}

	if !active.IsTracking() {
		return nil, nil
	}

	return blockRepo.Get(active.ActiveBlockKey)
}

// SetActiveBlock sets the given block as active.
func (r *ActiveBlockRepo) SetActiveBlock(block *model.Block) error {
	active, err := r.Get()
	if err != nil {
		return err
	}

	active.SetActive(block.Key)
	return r.Save(active)
}

// ClearActiveBlock clears the active block.
func (r *ActiveBlockRepo) ClearActiveBlock() error {
	active, err := r.Get()
	if err != nil {
		return err
	}

	active.ClearActive()
	return r.Save(active)
}

// GetPreviousBlock retrieves the previous block (for resume functionality).
func (r *ActiveBlockRepo) GetPreviousBlock(blockRepo *BlockRepo) (*model.Block, error) {
	active, err := r.Get()
	if err != nil {
		return nil, err
	}

	if active.PreviousBlockKey == "" {
		return nil, nil
	}

	return blockRepo.Get(active.PreviousBlockKey)
}

// SetActive sets the given block key as active (convenience method).
func (r *ActiveBlockRepo) SetActive(blockKey string) error {
	active, err := r.Get()
	if err != nil {
		return err
	}

	active.SetActive(blockKey)
	return r.Save(active)
}

// ClearActive clears the active block (convenience method).
func (r *ActiveBlockRepo) ClearActive() error {
	return r.ClearActiveBlock()
}
