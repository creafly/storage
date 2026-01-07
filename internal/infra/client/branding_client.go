package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type LogoByFileIDResponse struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenantId"`
	FileID   uuid.UUID `json:"fileId"`
}

type BrandingClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewBrandingClient(baseURL string) *BrandingClient {
	return &BrandingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *BrandingClient) GetLogoByFileID(ctx context.Context, tenantID, fileID uuid.UUID) (*LogoByFileIDResponse, error) {
	url := fmt.Sprintf("%s/api/v1/internal/logos/by-file/%s", c.baseURL, fileID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("X-Service-Name", "storage")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call branding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("branding service returned status %d", resp.StatusCode)
	}

	var logoResp LogoByFileIDResponse
	if err := json.NewDecoder(resp.Body).Decode(&logoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &logoResp, nil
}
