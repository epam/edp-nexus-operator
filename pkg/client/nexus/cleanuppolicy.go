package nexus

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type NexusCleanupPolicy struct {
	Name                    string  `json:"name"`
	Format                  string  `json:"format"`
	Notes                   string  `json:"notes"`
	CriteriaReleaseType     *string `json:"criteriaReleaseType"`
	CriteriaLastDownloaded  *int    `json:"criteriaLastDownloaded"`
	CriteriaLastBlobUpdated *int    `json:"criteriaLastBlobUpdated"`
	CriteriaAssetRegex      *string `json:"criteriaAssetRegex"`
}

type NexusCleanupPolicyClient struct {
	config ClientConfig
}

func NewNexusCleanupPolicyClient(config ClientConfig) *NexusCleanupPolicyClient {
	return &NexusCleanupPolicyClient{config: config}
}

func (s *NexusCleanupPolicyClient) Get(ctx context.Context, name string) (*NexusCleanupPolicy, error) {
	res := &NexusCleanupPolicy{}

	resp, err := s.r(ctx).
		SetPathParams(map[string]string{
			"name": name,
		}).
		SetResult(res).
		Get("/service/rest/internal/cleanup-policies/{name}")

	if err != nil {
		return nil, fmt.Errorf("failed to get cleanup policy: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("failed to get cleanup policy: %s", resp.String())
	}

	return res, nil
}

func (s *NexusCleanupPolicyClient) Create(ctx context.Context, policy *NexusCleanupPolicy) error {
	resp, err := s.r(ctx).
		SetBody(policy).
		Post("/service/rest/internal/cleanup-policies")

	if err != nil {
		return fmt.Errorf("failed to create cleanup policy: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to create cleanup policy: %s", resp.String())
	}

	return nil
}

func (s *NexusCleanupPolicyClient) Update(ctx context.Context, name string, policy *NexusCleanupPolicy) error {
	resp, err := s.r(ctx).
		SetPathParams(map[string]string{
			"name": name,
		}).
		SetBody(policy).
		Put("/service/rest/internal/cleanup-policies/{name}")

	if err != nil {
		return fmt.Errorf("failed to update cleanup policy: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to update cleanup policy: %s", resp.String())
	}

	return nil
}

func (s *NexusCleanupPolicyClient) Delete(ctx context.Context, name string) error {
	resp, err := s.r(ctx).
		SetPathParams(map[string]string{
			"name": name,
		}).
		Delete("/service/rest/internal/cleanup-policies/{name}")

	if err != nil {
		return fmt.Errorf("failed to delete cleanup policy: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}

		return fmt.Errorf("failed to delete cleanup policy: %s", resp.String())
	}

	return nil
}

func (s *NexusCleanupPolicyClient) r(ctx context.Context) *resty.Request {
	return resty.New().
		SetBaseURL(s.config.BaseURL).
		SetBasicAuth(s.config.UserName, s.config.Password).
		R().
		ForceContentType("application/json").
		SetContext(ctx)
}
