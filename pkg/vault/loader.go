package vault

import (
	"log"
	"os"
)

type VaultConfig struct {
	Enabled bool
	Address string
	Token   string
	Mount   string
}

func LoadVaultConfigFromEnv() VaultConfig {
	return VaultConfig{
		Enabled: os.Getenv("VAULT_ENABLED") == "true",
		Address: getEnvOrDefault("VAULT_ADDR", "http://vault:8200"),
		Token:   os.Getenv("VAULT_TOKEN"),
		Mount:   getEnvOrDefault("VAULT_MOUNT", "secret"),
	}
}

type SecretLoader struct {
	client      *Client
	servicePath string
	environment string
}

func NewSecretLoader(serviceName string) *SecretLoader {
	cfg := LoadVaultConfigFromEnv()

	loader := &SecretLoader{
		servicePath: "creafly/" + getEnvOrDefault("ENVIRONMENT", "production") + "/" + serviceName,
		environment: getEnvOrDefault("ENVIRONMENT", "production"),
	}

	if !cfg.Enabled {
		log.Println("Vault is disabled, using environment variables for secrets")
		return loader
	}

	if cfg.Token == "" {
		log.Println("VAULT_TOKEN not set, using environment variables for secrets")
		return loader
	}

	client, err := New(Config{
		Address: cfg.Address,
		Token:   cfg.Token,
		Mount:   cfg.Mount,
	})
	if err != nil {
		log.Printf("Failed to create Vault client: %v, using environment variables", err)
		return loader
	}

	if err := client.Health(); err != nil {
		log.Printf("Vault health check failed: %v, using environment variables", err)
		return loader
	}

	loader.client = client
	log.Printf("Vault client initialized, loading secrets from %s", loader.servicePath)
	return loader
}

func (l *SecretLoader) GetSecret(vaultKey, envKey, defaultValue string) string {
	if l.client != nil {
		value, err := l.client.GetSecretValue(l.servicePath, vaultKey)
		if err != nil {
			log.Printf("Failed to get %s from Vault path %s: %v, falling back to env", vaultKey, l.servicePath, err)
		} else if value != "" {
			return value
		} else {
			log.Printf("Secret %s not found in Vault path %s, falling back to env", vaultKey, l.servicePath)
		}
	}

	if value := os.Getenv(envKey); value != "" {
		return value
	}

	return defaultValue
}

func (l *SecretLoader) GetSecretRequired(vaultKey, envKey string) string {
	value := l.GetSecret(vaultKey, envKey, "")
	if value == "" {
		log.Fatalf("Required secret %s (env: %s) is not set", vaultKey, envKey)
	}
	return value
}

func (l *SecretLoader) IsVaultEnabled() bool {
	return l.client != nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
