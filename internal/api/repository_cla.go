package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RepositoryCLA is an available agreement, not the repository's current configuration.
type RepositoryCLA struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func ListRepositoryCLAs(client *Client, owner, repo string) ([]RepositoryCLA, error) {
	var response struct {
		Data *[]RepositoryCLA `json:"data"`
	}
	err := client.doJSONRequest(http.MethodGet, policyPath(owner, repo, "clas"), nil, "", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: true}, &response)
	if err != nil {
		return nil, fmt.Errorf("list repository CLAs: %w", err)
	}
	if response.Data == nil {
		return nil, fmt.Errorf("list repository CLAs: %w: missing data array", ErrInvalidResponse)
	}
	for _, cla := range *response.Data {
		if strings.TrimSpace(cla.ID) == "" || strings.TrimSpace(cla.Name) == "" {
			return nil, fmt.Errorf("list repository CLAs: %w: missing agreement identity", ErrInvalidResponse)
		}
	}
	return *response.Data, nil
}

// SetRepositoryCLA requires an ID; callers cannot accidentally disable CLA by omitting it.
func SetRepositoryCLA(client *Client, owner, repo, id string, enforce *bool) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("CLA ID must not be empty")
	}
	return updateRepositoryCLA(client, owner, repo, struct {
		ID      string `json:"cla_id"`
		Enforce *bool  `json:"enforce_check,omitempty"`
	}{id, enforce})
}

func DisableRepositoryCLA(client *Client, owner, repo string) error {
	return updateRepositoryCLA(client, owner, repo, struct{}{})
}

func updateRepositoryCLA(client *Client, owner, repo string, request any) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode repository CLA configuration: %w", err)
	}
	// Do not retry or replay redirects: a concurrent administrator may change the configuration.
	httpClient := *metadataHTTPClient(client.httpClient)
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	err = client.doJSONRequestContext(client.requestContext(), &httpClient, http.MethodPut,
		policyPath(owner, repo, "cla"), bytes.NewReader(payload), "application/json", "application/json",
		RequestPolicy{AllowedStatuses: []int{http.StatusOK}, CanRetry: false}, nil)
	if err != nil {
		return fmt.Errorf("update repository CLA configuration (remote state may have changed; check AtomGit before retrying): %w", err)
	}
	return nil
}
