package storage

import (
	"github.com/google/uuid"
	"github.com/manav03panchal/humantime/internal/model"
)

// ConfigRepo provides operations for the Config singleton.
type ConfigRepo struct {
	db *DB
}

// NewConfigRepo creates a new config repository.
func NewConfigRepo(db *DB) *ConfigRepo {
	return &ConfigRepo{db: db}
}

// Get retrieves the config, creating it if it doesn't exist.
func (r *ConfigRepo) Get() (*model.Config, error) {
	config := &model.Config{}
	err := r.db.Get(model.KeyConfig, config)
	if err == nil {
		return config, nil
	}

	if !IsErrKeyNotFound(err) {
		return nil, err
	}

	// Create new config with generated user key
	userKey, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	config = model.NewConfig(userKey.String())
	if err := r.db.Set(config); err != nil {
		return nil, err
	}

	return config, nil
}

// Update updates the config.
func (r *ConfigRepo) Update(config *model.Config) error {
	return r.db.Set(config)
}

// ActiveBlockRepo provides operations for the ActiveBlock singleton.
type ActiveBlockRepo struct {
	db *DB
}

// NewActiveBlockRepo creates a new active block repository.
func NewActiveBlockRepo(db *DB) *ActiveBlockRepo {
	return &ActiveBlockRepo{db: db}
}

// Get retrieves the active block state, creating it if it doesn't exist.
func (r *ActiveBlockRepo) Get() (*model.ActiveBlock, error) {
	active := &model.ActiveBlock{}
	err := r.db.Get(model.KeyActiveBlock, active)
	if err == nil {
		return active, nil
	}

	if !IsErrKeyNotFound(err) {
		return nil, err
	}

	// Create new active block record
	active = model.NewActiveBlock()
	if err := r.db.Set(active); err != nil {
		return nil, err
	}

	return active, nil
}

// Update updates the active block state.
func (r *ActiveBlockRepo) Update(active *model.ActiveBlock) error {
	return r.db.Set(active)
}

// SetActive sets the active block key.
func (r *ActiveBlockRepo) SetActive(blockKey string) error {
	active, err := r.Get()
	if err != nil {
		return err
	}

	active.SetActive(blockKey)
	return r.Update(active)
}

// ClearActive clears the active block.
func (r *ActiveBlockRepo) ClearActive() error {
	active, err := r.Get()
	if err != nil {
		return err
	}

	active.ClearActive()
	return r.Update(active)
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

// GetPreviousBlock retrieves the previously tracked block, if any.
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

// NotifyConfigRepo provides operations for the NotifyConfig singleton.
type NotifyConfigRepo struct {
	db *DB
}

// NewNotifyConfigRepo creates a new notify config repository.
func NewNotifyConfigRepo(db *DB) *NotifyConfigRepo {
	return &NotifyConfigRepo{db: db}
}

// Get retrieves the notify config, returning defaults if not set.
func (r *NotifyConfigRepo) Get() (*model.NotifyConfig, error) {
	config := &model.NotifyConfig{}
	err := r.db.GetRaw(model.KeyNotifyConfig, config)
	if err == nil {
		return config, nil
	}

	if !IsErrKeyNotFound(err) {
		return nil, err
	}

	// Return default config (don't persist until explicitly set)
	return model.DefaultNotifyConfig(), nil
}

// Set stores the notify config.
func (r *NotifyConfigRepo) Set(config *model.NotifyConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	return r.db.SetRaw(model.KeyNotifyConfig, config)
}

// Update updates a specific field of the notify config.
func (r *NotifyConfigRepo) Update(config *model.NotifyConfig) error {
	return r.Set(config)
}
