package model

// UndoAction represents the type of action that can be undone.
type UndoAction string

const (
	UndoActionStart  UndoAction = "start"
	UndoActionStop   UndoAction = "stop"
	UndoActionDelete UndoAction = "delete"
)

// KeyUndo is the database key for the undo state.
const KeyUndo = "undo"

// UndoState stores information about the last action that can be undone.
type UndoState struct {
	Key           string     `json:"key"`
	Action        UndoAction `json:"action"`
	BlockKey      string     `json:"block_key"`
	BlockSnapshot *Block     `json:"block_snapshot,omitempty"` // Full block data for restore
}

// SetKey sets the database key for this undo state.
func (u *UndoState) SetKey(key string) {
	u.Key = key
}

// GetKey returns the database key for this undo state.
func (u *UndoState) GetKey() string {
	return u.Key
}

// NewUndoState creates a new undo state for the given action.
func NewUndoState(action UndoAction, blockKey string, snapshot *Block) *UndoState {
	return &UndoState{
		Key:           KeyUndo,
		Action:        action,
		BlockKey:      blockKey,
		BlockSnapshot: snapshot,
	}
}
