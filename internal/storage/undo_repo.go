package storage

import (
	"github.com/manav03panchal/humantime/internal/model"
)

// UndoRepo provides operations for UndoState entities.
type UndoRepo struct {
	db *DB
}

// NewUndoRepo creates a new undo repository.
func NewUndoRepo(db *DB) *UndoRepo {
	return &UndoRepo{db: db}
}

// Get retrieves the current undo state.
func (r *UndoRepo) Get() (*model.UndoState, error) {
	state := &model.UndoState{}
	if err := r.db.Get(model.KeyUndo, state); err != nil {
		if IsErrKeyNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return state, nil
}

// Set saves the undo state.
func (r *UndoRepo) Set(state *model.UndoState) error {
	state.Key = model.KeyUndo
	return r.db.Set(state)
}

// Clear removes the undo state.
func (r *UndoRepo) Clear() error {
	return r.db.Delete(model.KeyUndo)
}

// SaveUndoStart saves undo state for a start action.
func (r *UndoRepo) SaveUndoStart(blockKey string) error {
	state := model.NewUndoState(model.UndoActionStart, blockKey, nil)
	return r.Set(state)
}

// SaveUndoStop saves undo state for a stop action.
func (r *UndoRepo) SaveUndoStop(block *model.Block) error {
	state := model.NewUndoState(model.UndoActionStop, block.Key, block)
	return r.Set(state)
}

// SaveUndoDelete saves undo state for a delete action with full block snapshot.
func (r *UndoRepo) SaveUndoDelete(block *model.Block) error {
	// Create a copy of the block for the snapshot
	snapshot := &model.Block{
		Key:            block.Key,
		OwnerKey:       block.OwnerKey,
		ProjectSID:     block.ProjectSID,
		TaskSID:        block.TaskSID,
		Note:           block.Note,
		Tags:           block.Tags,
		TimestampStart: block.TimestampStart,
		TimestampEnd:   block.TimestampEnd,
	}
	state := model.NewUndoState(model.UndoActionDelete, block.Key, snapshot)
	return r.Set(state)
}
