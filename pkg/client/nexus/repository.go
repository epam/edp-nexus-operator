package nexus

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

var ErrNotFound = errors.New("not found")

type ClientConfig struct {
	BaseURL  string
	UserName string
	Password string
}

type RepoClient struct {
	config ClientConfig
}

func NewRepoClient(config ClientConfig) *RepoClient {
	return &RepoClient{config: config}
}

func (s *RepoClient) Get(ctx context.Context, id, format, repoType string) (interface{}, error) {
	res := map[string]interface{}{}

	resp, err := s.r(ctx).
		SetPathParams(map[string]string{
			"id":     id,
			"format": format,
			"type":   repoType,
		}).
		SetResult(&res).
		Get("/service/rest/v1/repositories/{format}/{type}/{id}")

	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return nil, fmt.Errorf("repository %s %w: %s", id, ErrNotFound, resp.String())
		}

		return nil, fmt.Errorf("failed to get repository: %s", resp.String())
	}

	return res, nil
}

func (s *RepoClient) Create(ctx context.Context, format, repoType string, data interface{}) error {
	resp, err := s.r(ctx).
		SetPathParams(map[string]string{
			"format": format,
			"type":   repoType,
		}).
		SetBody(data).
		Post("/service/rest/v1/repositories/{format}/{type}")

	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to create repository: %s", resp.String())
	}

	return nil
}

func (s *RepoClient) Update(ctx context.Context, id, format, repoType string, data interface{}) error {
	resp, err := s.r(ctx).
		SetPathParams(map[string]string{
			"id":     id,
			"format": format,
			"type":   repoType,
		}).
		SetBody(data).
		Put("/service/rest/v1/repositories/{format}/{type}/{id}")

	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("failed to update repository: %s", resp.String())
	}

	return nil
}

func (s *RepoClient) Delete(ctx context.Context, id string) error {
	resp, err := s.r(ctx).
		SetPathParams(map[string]string{
			"id": id,
		}).
		Delete("/service/rest/v1/repositories/{id}")

	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return fmt.Errorf("repository %s %w: %s", id, ErrNotFound, resp.String())
		}

		return fmt.Errorf("failed to delete repository: %s", resp.String())
	}

	return nil
}

func (s *RepoClient) r(ctx context.Context) *resty.Request {
	return resty.New().
		SetBaseURL(s.config.BaseURL).
		SetBasicAuth(s.config.UserName, s.config.Password).
		R().
		ForceContentType("application/json").
		SetContext(ctx)
}
