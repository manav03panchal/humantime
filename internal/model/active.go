package model

// ActiveBlock is a singleton that tracks the currently active time block.
type ActiveBlock struct {
	Key              string `json:"key"`
	ActiveBlockKey   string `json:"active_block_key,omitempty"`
	PreviousBlockKey string `json:"previous_block_key,omitempty"`
}

// SetKey sets the database key for this active block record.
func (a *ActiveBlock) SetKey(key string) {
	a.Key = key
}

// GetKey returns the database key for this active block record.
func (a *ActiveBlock) GetKey() string {
	return a.Key
}

// IsTracking returns true if there is currently active tracking.
func (a *ActiveBlock) IsTracking() bool {
	return a.ActiveBlockKey != ""
}

// NewActiveBlock creates a new active block singleton.
func NewActiveBlock() *ActiveBlock {
	return &ActiveBlock{
		Key: KeyActiveBlock,
	}
}

// SetActive sets the active block key and moves current to previous.
func (a *ActiveBlock) SetActive(blockKey string) {
	if a.ActiveBlockKey != "" {
		a.PreviousBlockKey = a.ActiveBlockKey
	}
	a.ActiveBlockKey = blockKey
}

// ClearActive clears the active block and saves it as previous.
func (a *ActiveBlock) ClearActive() {
	if a.ActiveBlockKey != "" {
		a.PreviousBlockKey = a.ActiveBlockKey
	}
	a.ActiveBlockKey = ""
}
