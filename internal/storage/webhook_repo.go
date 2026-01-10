package storage

import (
	"time"

	"github.com/manav03panchal/humantime/internal/model"
)

// WebhookRepo provides operations for Webhook entities.
type WebhookRepo struct {
	db *DB
}

// NewWebhookRepo creates a new webhook repository.
func NewWebhookRepo(db *DB) *WebhookRepo {
	return &WebhookRepo{db: db}
}

// Create creates a new webhook.
func (r *WebhookRepo) Create(webhook *model.Webhook) error {
	if webhook.Key == "" {
		webhook.Key = model.GenerateWebhookKey(webhook.Name)
	}
	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = time.Now()
	}
	return r.db.Set(webhook)
}

// Get retrieves a webhook by name.
func (r *WebhookRepo) Get(name string) (*model.Webhook, error) {
	webhook := &model.Webhook{}
	key := model.GenerateWebhookKey(name)
	if err := r.db.Get(key, webhook); err != nil {
		return nil, err
	}
	return webhook, nil
}

// GetByKey retrieves a webhook by full key.
func (r *WebhookRepo) GetByKey(key string) (*model.Webhook, error) {
	webhook := &model.Webhook{}
	if err := r.db.Get(key, webhook); err != nil {
		return nil, err
	}
	return webhook, nil
}

// List retrieves all webhooks.
func (r *WebhookRepo) List() ([]*model.Webhook, error) {
	return GetAllByPrefix(r.db, model.PrefixWebhook+":", func() *model.Webhook {
		return &model.Webhook{}
	})
}

// ListEnabled retrieves all enabled webhooks.
func (r *WebhookRepo) ListEnabled() ([]*model.Webhook, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}

	var enabled []*model.Webhook
	for _, wh := range all {
		if wh.IsEnabled() {
			enabled = append(enabled, wh)
		}
	}
	return enabled, nil
}

// Update updates an existing webhook.
func (r *WebhookRepo) Update(webhook *model.Webhook) error {
	return r.db.Set(webhook)
}

// Delete removes a webhook by name.
func (r *WebhookRepo) Delete(name string) error {
	key := model.GenerateWebhookKey(name)
	return r.db.Delete(key)
}

// Enable enables a webhook.
func (r *WebhookRepo) Enable(name string) error {
	webhook, err := r.Get(name)
	if err != nil {
		return err
	}
	webhook.Enabled = true
	return r.db.Set(webhook)
}

// Disable disables a webhook.
func (r *WebhookRepo) Disable(name string) error {
	webhook, err := r.Get(name)
	if err != nil {
		return err
	}
	webhook.Enabled = false
	return r.db.Set(webhook)
}

// UpdateLastUsed updates the last used timestamp and optionally the last error.
func (r *WebhookRepo) UpdateLastUsed(name string, lastErr error) error {
	webhook, err := r.Get(name)
	if err != nil {
		return err
	}

	webhook.LastUsed = time.Now()
	if lastErr != nil {
		webhook.LastError = lastErr.Error()
	} else {
		webhook.LastError = ""
	}

	return r.db.Set(webhook)
}

// Exists checks if a webhook with the given name exists.
func (r *WebhookRepo) Exists(name string) (bool, error) {
	key := model.GenerateWebhookKey(name)
	return r.db.Exists(key)
}
