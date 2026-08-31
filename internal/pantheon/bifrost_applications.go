package pantheon

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// bifrostApplicationConfiguration is connection metadata owned by Bifrost.
// It deliberately contains no credential value; credential_env is only the
// name of the local secret variable a capability may use at invocation time.
type bifrostApplicationConfiguration struct {
	Application   string `json:"application"`
	Endpoint      string `json:"endpoint"`
	Route         string `json:"route"`
	Model         string `json:"model"`
	CredentialEnv string `json:"credential_env,omitempty"`
	ConfigPath    string `json:"config_path,omitempty"`
	UpdatedAt     string `json:"updated_at"`
}

type bifrostApplicationConfigurationRequest struct {
	Application   string `json:"application"`
	Endpoint      string `json:"endpoint"`
	Route         string `json:"route"`
	Model         string `json:"model"`
	CredentialEnv string `json:"credential_env,omitempty"`
	ConfigPath    string `json:"config_path,omitempty"`
}

func validateBifrostApplicationConfiguration(request bifrostApplicationConfigurationRequest) error {
	request.Application = strings.ToLower(strings.TrimSpace(request.Application))
	if request.Application != "switchyard" {
		return fmt.Errorf("unsupported Bifrost application %q", request.Application)
	}
	request.Endpoint = strings.TrimSpace(request.Endpoint)
	endpoint, err := url.Parse(request.Endpoint)
	if err != nil || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return errors.New("Bifrost application endpoint must be a credential-free URL without query or fragment")
	}
	if endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && localApolloHosts[strings.ToLower(endpoint.Hostname())]) {
		return errors.New("Bifrost application endpoint must use HTTPS; plain HTTP is local-test-only")
	}
	if strings.TrimSpace(request.Route) == "" {
		return errors.New("Bifrost application route is required")
	}
	if len(request.Route) > 300 || len(request.Model) > 300 || len(request.ConfigPath) > 1000 {
		return errors.New("Bifrost application configuration exceeds its bounds")
	}
	if strings.TrimSpace(request.Model) == "" {
		return errors.New("Bifrost application model is required")
	}
	if request.CredentialEnv != "" && !validArchimedesCredentialEnvName(request.CredentialEnv) {
		return errors.New("Bifrost application credential_env must be an environment variable name")
	}
	if strings.IndexByte(request.ConfigPath, 0) >= 0 {
		return errors.New("Bifrost application config_path contains a NUL byte")
	}
	return nil
}

func (d *daemon) loadBifrostApplicationConfiguration(capabilityID string) (*bifrostApplicationConfiguration, error) {
	var configuration bifrostApplicationConfiguration
	err := d.db.QueryRow(`SELECT application, endpoint, route, model, COALESCE(credential_env, ''), COALESCE(config_path, ''), updated_at FROM bifrost_application_configurations WHERE capability_id = ?`, capabilityID).Scan(&configuration.Application, &configuration.Endpoint, &configuration.Route, &configuration.Model, &configuration.CredentialEnv, &configuration.ConfigPath, &configuration.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &configuration, nil
}

func (d *daemon) saveBifrostApplicationConfiguration(capabilityID string, request bifrostApplicationConfigurationRequest) (*bifrostApplicationConfiguration, error) {
	capabilityID = strings.TrimSpace(capabilityID)
	if capabilityID == "" {
		return nil, errors.New("Bifrost capability ID is required")
	}
	if _, err := d.loadBifrostCapability(capabilityID); err != nil {
		return nil, err
	}
	request.Application = strings.ToLower(strings.TrimSpace(request.Application))
	request.Endpoint = strings.TrimSpace(request.Endpoint)
	request.Route = strings.TrimSpace(request.Route)
	request.Model = strings.TrimSpace(request.Model)
	request.CredentialEnv = strings.TrimSpace(request.CredentialEnv)
	request.ConfigPath = strings.TrimSpace(request.ConfigPath)
	if err := validateBifrostApplicationConfiguration(request); err != nil {
		return nil, err
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := d.db.Exec(`INSERT INTO bifrost_application_configurations (capability_id, application, endpoint, route, model, credential_env, config_path, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(capability_id) DO UPDATE SET application = excluded.application, endpoint = excluded.endpoint, route = excluded.route, model = excluded.model, credential_env = excluded.credential_env, config_path = excluded.config_path, updated_at = excluded.updated_at`, capabilityID, request.Application, request.Endpoint, request.Route, request.Model, nullableString(request.CredentialEnv), nullableString(request.ConfigPath), updatedAt)
	if err != nil {
		return nil, err
	}
	return d.loadBifrostApplicationConfiguration(capabilityID)
}

func (d *daemon) handleBifrostApplicationConfiguration(w http.ResponseWriter, r *http.Request) {
	if !d.bifrostControlRequired(w, r) {
		return
	}
	var request bifrostApplicationConfigurationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, 400, err)
		return
	}
	configuration, err := d.saveBifrostApplicationConfiguration(r.PathValue("id"), request)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, errors.New("Bifrost capability not found"))
		return
	}
	if err != nil {
		writeError(w, 400, err)
		return
	}
	writeJSON(w, 200, configuration)
}

func (d *daemon) handleGUIBifrostApplicationConfiguration(w http.ResponseWriter, r *http.Request) {
	if !d.guiControlRequired(w, r) {
		return
	}
	var request bifrostApplicationConfigurationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, 400, err)
		return
	}
	capabilityID := r.PathValue("id")
	configuration, err := d.saveBifrostApplicationConfiguration(capabilityID, request)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, errors.New("Bifrost capability not found"))
		return
	}
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": bifrostSafeError(err.Error())})
		return
	}
	_ = d.audit("POST /v1/gui/bifrost/capabilities/"+capabilityID+"/configuration", capabilityID, "bifrost.configure", "", "requested", "Bifrost application configuration changed", "", 0)
	writeJSON(w, 200, configuration)
}
