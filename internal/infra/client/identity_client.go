package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type VerifyResponse struct {
	Valid    bool       `json:"valid"`
	UserID   uuid.UUID  `json:"userId,omitempty"`
	Email    string     `json:"email,omitempty"`
	TenantID *uuid.UUID `json:"tenantId,omitempty"`
	Error    string     `json:"error,omitempty"`
}

type ValidateTenantResponse struct {
	Valid    bool   `json:"valid"`
	IsMember bool   `json:"isMember"`
	TenantID string `json:"tenantId,omitempty"`
	UserID   string `json:"userId,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Error    string `json:"error,omitempty"`
}

type IdentityClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewIdentityClient(baseURL string) *IdentityClient {
	return &IdentityClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *IdentityClient) VerifyToken(ctx context.Context, accessToken string) (*VerifyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/auth/verify", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call identity service: %w", err)
	}
	defer resp.Body.Close()

	var verifyResp VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &verifyResp, nil
}

func (c *IdentityClient) ValidateTenantAccess(ctx context.Context, accessToken string, tenantID uuid.UUID, userID uuid.UUID) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/tenants/%s/validate", c.baseURL, tenantID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to call identity service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, nil
	}

	var validateResp ValidateTenantResponse
	if err := json.NewDecoder(resp.Body).Decode(&validateResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return validateResp.Valid && validateResp.IsMember, nil
}
