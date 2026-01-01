package vault

import (
	"fmt"
	"sync"

	"github.com/hashicorp/vault/api"
)

type Config struct {
	Address string
	Token   string
	Mount   string
}

type Client struct {
	client *api.Client
	mount  string
	mu     sync.RWMutex
}

var (
	once     sync.Once
	instance *Client
)

func New(cfg Config) (*Client, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("vault address is required")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("vault token is required")
	}
	if cfg.Mount == "" {
		cfg.Mount = "secret"
	}

	config := api.DefaultConfig()
	config.Address = cfg.Address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(cfg.Token)

	return &Client{
		client: client,
		mount:  cfg.Mount,
	}, nil
}

func InitGlobal(cfg Config) error {
	var initErr error
	once.Do(func() {
		var err error
		instance, err = New(cfg)
		if err != nil {
			initErr = err
		}
	})
	return initErr
}

func GetInstance() *Client {
	return instance
}

func (c *Client) GetSecret(path string) (map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fullPath := fmt.Sprintf("%s/data/%s", c.mount, path)

	secret, err := c.client.Logical().Read(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no secret found at path: %s", path)
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid secret format at path: %s", path)
	}

	return data, nil
}

func (c *Client) GetSecretValue(path, key string) (string, error) {
	data, err := c.GetSecret(path)
	if err != nil {
		return "", err
	}

	value, ok := data[key]
	if !ok {
		return "", nil
	}

	strValue, ok := value.(string)
	if !ok {
		return fmt.Sprintf("%v", value), nil
	}

	return strValue, nil
}

func (c *Client) GetSecretValueOrDefault(path, key, defaultValue string) string {
	value, err := c.GetSecretValue(path, key)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}

func (c *Client) Health() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	health, err := c.client.Sys().Health()
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if !health.Initialized {
		return fmt.Errorf("vault is not initialized")
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	return nil
}

func (c *Client) WriteSecret(path string, data map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fullPath := fmt.Sprintf("%s/data/%s", c.mount, path)

	_, err := c.client.Logical().Write(fullPath, map[string]interface{}{
		"data": data,
	})
	if err != nil {
		return fmt.Errorf("failed to write secret to %s: %w", path, err)
	}

	return nil
}
