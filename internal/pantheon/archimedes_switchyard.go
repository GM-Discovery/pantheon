package pantheon

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const archimedesProviderSwitchyard = "switchyard"

type archimedesTransportSettings struct {
	Provider           string
	Endpoint           string
	Route              string
	Model              string
	CredentialEnv      string
	CredentialRequired bool
	CapabilityID       string
}

func (d *daemon) switchyardTransportSettings() (archimedesTransportSettings, error) {
	capabilityID := strings.TrimSpace(d.cfg.ArchimedesBifrostCapabilityID)
	capability, err := d.loadBifrostCapability(capabilityID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return archimedesTransportSettings{}, fmt.Errorf("Switchyard Bifrost capability %q is not registered", capabilityID)
		}
		return archimedesTransportSettings{}, fmt.Errorf("load Switchyard Bifrost capability %q: %w", capabilityID, err)
	}
	if capability.ApplicationConfig == nil || capability.ApplicationConfig.Application != archimedesProviderSwitchyard {
		return archimedesTransportSettings{}, fmt.Errorf("Switchyard Bifrost capability %q has no application configuration", capabilityID)
	}
	configuration := capability.ApplicationConfig
	return archimedesTransportSettings{
		Provider: archimedesProviderSwitchyard, Endpoint: configuration.Endpoint, Route: configuration.Route,
		Model: configuration.Model, CredentialEnv: configuration.CredentialEnv,
		CredentialRequired: capability.AuthenticationRequired, CapabilityID: capabilityID,
	}, nil
}

func validateArchimedesTransportSettings(settings archimedesTransportSettings) error {
	endpoint, err := url.Parse(strings.TrimSpace(settings.Endpoint))
	if err != nil || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return errors.New("Switchyard endpoint is not a credential-free URL")
	}
	if endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && localApolloHosts[strings.ToLower(endpoint.Hostname())]) {
		return errors.New("Switchyard endpoint must use HTTPS; plain HTTP is local-test-only")
	}
	if strings.TrimSpace(settings.Route) == "" {
		return errors.New("Switchyard route is not configured by Bifrost")
	}
	if strings.TrimSpace(settings.Model) == "" {
		return errors.New("Switchyard model is not configured by Bifrost")
	}
	if settings.CredentialRequired && !validArchimedesCredentialEnvName(settings.CredentialEnv) {
		return errors.New("Switchyard credential environment name is invalid")
	}
	return nil
}

func (d *daemon) callArchimedesSwitchyard(ctx context.Context, exchange archimedesExchange, missive archimedesMissive) (archimedesProviderResponse, error) {
	settings, err := d.switchyardTransportSettings()
	if err != nil {
		return archimedesProviderResponse{}, err
	}
	if err := validateArchimedesTransportSettings(settings); err != nil {
		return archimedesProviderResponse{}, err
	}
	credential := ""
	if settings.CredentialRequired {
		credential = os.Getenv(strings.TrimSpace(settings.CredentialEnv))
		if strings.TrimSpace(credential) == "" {
			return archimedesProviderResponse{}, fmt.Errorf("Switchyard credential environment variable %s is not set", settings.CredentialEnv)
		}
	}
	missiveJSON, err := json.Marshal(missive)
	if err != nil {
		return archimedesProviderResponse{}, fmt.Errorf("marshal minimized Archimedes missive for Switchyard: %w", err)
	}
	requestBody, err := json.Marshal(archimedesChatRequest{
		Model: settings.Route,
		Messages: []archimedesChatMessage{
			{Role: "system", Content: "You are a frontier reasoning transport behind Pantheon's Archimedes boundary. Return JSON only: {\"answer\":\"...\",\"actions\":[{\"kind\":\"calendar_event|email_draft|other\",\"summary\":\"...\",\"payload\":{}}],\"uncertainties\":[]}. Actions are suggestions only. Never request raw identities, local paths, credentials, tools, or additional context."},
			{Role: "user", Content: string(missiveJSON)},
		},
		Stream: false,
	})
	if err != nil {
		return archimedesProviderResponse{}, fmt.Errorf("marshal Switchyard request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, settings.Endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return archimedesProviderResponse{}, fmt.Errorf("create Switchyard request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Pantheon-Component", "Archimedes")
	request.Header.Set("X-Pantheon-Exchange-ID", exchange.ExchangeID)
	request.Header.Set("X-Pantheon-Route", exchange.Route)
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	timeout := d.cfg.ArchimedesTimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	response, err := (&http.Client{Timeout: time.Duration(timeout) * time.Second}).Do(request)
	if err != nil {
		return archimedesProviderResponse{}, fmt.Errorf("Switchyard transport request failed: %w", err)
	}
	defer response.Body.Close()
	maxBytes := d.cfg.ArchimedesMaxResponseBytes
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, int64(maxBytes)+1))
	if err != nil {
		return archimedesProviderResponse{}, fmt.Errorf("read Switchyard response: %w", err)
	}
	if len(responseBody) > maxBytes {
		return archimedesProviderResponse{}, fmt.Errorf("Switchyard response exceeded %d bytes", maxBytes)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return archimedesProviderResponse{}, fmt.Errorf("Switchyard returned HTTP %d", response.StatusCode)
	}
	var decoded archimedesChatResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return archimedesProviderResponse{}, fmt.Errorf("decode Switchyard response: %w", err)
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return archimedesProviderResponse{}, errors.New("Switchyard response contained no answer")
	}
	model := strings.TrimSpace(decoded.Model)
	if model == "" {
		model = settings.Model
	}
	return archimedesProviderResponse{ExchangeID: exchange.ExchangeID, Turn: exchange.Turn, Provider: exchange.Provider, Model: model, Status: "complete", Answer: decoded.Choices[0].Message.Content}, nil
}
