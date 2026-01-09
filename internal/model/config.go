package model

// Config holds application configuration (singleton).
type Config struct {
	Key     string `json:"key"`
	UserKey string `json:"user_key" validate:"required"`
}

// SetKey sets the database key for this config.
func (c *Config) SetKey(key string) {
	c.Key = key
}

// GetKey returns the database key for this config.
func (c *Config) GetKey() string {
	return c.Key
}

// NewConfig creates a new config with the given user key.
func NewConfig(userKey string) *Config {
	return &Config{
		Key:     KeyConfig,
		UserKey: userKey,
	}
}
