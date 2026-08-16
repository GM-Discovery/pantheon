package pantheon

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const prometheusPauseMarkerName = "prometheus.pause"

type serviceStatus struct {
	State     string
	Available bool
}

type serviceController interface {
	status(context.Context) (serviceStatus, error)
	start(context.Context) error
	stop(context.Context) error
}

type systemdUserService struct {
	name string
}

func (s systemdUserService) status(ctx context.Context) (serviceStatus, error) {
	output, err := exec.CommandContext(ctx, "systemctl", "--user", "is-active", s.name).CombinedOutput()
	state := strings.TrimSpace(string(output))
	if err == nil {
		if state == "" {
			state = "active"
		}
		return serviceStatus{State: state, Available: true}, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && knownSystemdServiceState(state) {
		return serviceStatus{State: state, Available: true}, nil
	}
	return serviceStatus{}, fmt.Errorf("systemd user service inspection unavailable: %w", err)
}

func knownSystemdServiceState(state string) bool {
	switch state {
	case "active", "inactive", "failed", "activating", "deactivating", "reloading":
		return true
	default:
		return false
	}
}

func (s systemdUserService) start(ctx context.Context) error {
	return runSystemctl(ctx, "start", s.name)
}

func (s systemdUserService) stop(ctx context.Context) error {
	return runSystemctl(ctx, "stop", s.name)
}

func runSystemctl(ctx context.Context, action, service string) error {
	output, err := exec.CommandContext(ctx, "systemctl", "--user", action, service).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("systemctl --user %s %s failed: %s", action, service, message)
	}
	return nil
}

func pauseMarkerPathForDatabase(databasePath string) string {
	return filepath.Join(filepath.Dir(databasePath), prometheusPauseMarkerName)
}

func pauseMarkerExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writePauseMarker(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".prometheus-pause-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := fmt.Fprintf(tmp, "paused_at=%s\n", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func removePauseMarker(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func setPersistedPauseState(databasePath string, paused bool) error {
	if _, err := os.Stat(databasePath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return err
	}
	defer db.Close()
	value := "false"
	if paused {
		value = "true"
	}
	_, err = db.Exec(`UPDATE runtime_state SET value = ?, updated_at = ? WHERE key = 'paused'`, value, time.Now().Format(time.RFC3339Nano))
	return err
}

type protocolDaemonState struct {
	Reachable bool
	Paused    *bool
	Error     error
}

func readProtocolDaemonState(baseURL string) protocolDaemonState {
	client := &http.Client{Timeout: 2 * time.Second}
	request, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/v1/status", nil)
	if err != nil {
		return protocolDaemonState{Error: err}
	}
	response, err := client.Do(request)
	if err != nil {
		return protocolDaemonState{Error: err}
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return protocolDaemonState{Error: fmt.Errorf("daemon returned %s: %s", response.Status, strings.TrimSpace(string(body)))}
	}
	var status statusResponse
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		return protocolDaemonState{Error: err}
	}
	return protocolDaemonState{Reachable: true, Paused: &status.Paused}
}

func postProtocolDaemon(baseURL, path string) error {
	return postProtocolDaemonWithTimeout(baseURL, path, 5*time.Second)
}

func postProtocolDaemonWithTimeout(baseURL, path string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return fmt.Errorf("daemon returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func protocolService() serviceController {
	return systemdUserService{name: "pantheon"}
}

func protocolContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func runPrometheusProtocol(action string) (string, error) {
	options, err := defaultDaemonOptions()
	if err != nil {
		return "", err
	}
	return runPrometheusProtocolWith(action, protocolService(), daemonURL, pauseMarkerPathForDatabase(options.DatabasePath), options.DatabasePath)
}

func runPrometheusProtocolWith(action string, service serviceController, baseURL, markerPath, databasePath string) (string, error) {
	switch action {
	case "status":
		return protocolStatus(service, baseURL, markerPath), nil
	case "pause":
		return protocolPause(service, baseURL, markerPath)
	case "resume":
		return protocolResume(service, baseURL, markerPath, databasePath)
	case "hard-stop":
		return protocolHardStop(service, baseURL)
	default:
		return "", fmt.Errorf("unknown Prometheus Protocol action %q; expected status, pause, resume, or hard-stop", action)
	}
}

func protocolStatus(service serviceController, baseURL, markerPath string) string {
	ctx, cancel := protocolContext()
	defer cancel()
	serviceState, serviceErr := service.status(ctx)
	daemonState := readProtocolDaemonState(baseURL)
	markerPaused := pauseMarkerExists(markerPath)

	serviceText := serviceState.State
	if serviceErr != nil {
		serviceText = "unavailable"
	}
	daemonText := "unreachable"
	if daemonState.Reachable {
		daemonText = "reachable"
	}
	pausedText := "unknown"
	if daemonState.Paused != nil {
		pausedText = fmt.Sprintf("%t", *daemonState.Paused)
	} else if markerPaused {
		pausedText = "true"
	}
	controlText := "healthy"
	switch {
	case serviceErr != nil:
		controlText = "systemd unavailable"
	case daemonState.Reachable && daemonState.Paused != nil && markerPaused != *daemonState.Paused:
		controlText = "inconsistent"
	case serviceState.State != "active" && !daemonState.Reachable:
		controlText = "hard-stopped"
	case !daemonState.Reachable:
		controlText = "daemon unavailable"
	}

	var b strings.Builder
	b.WriteString("Prometheus Protocol\n\n")
	fmt.Fprintf(&b, "service: %s\n", serviceText)
	fmt.Fprintf(&b, "daemon: %s\n", daemonText)
	fmt.Fprintf(&b, "paused: %s\n", pausedText)
	fmt.Fprintf(&b, "control: %s\n", controlText)
	if serviceErr != nil {
		fmt.Fprintf(&b, "service detail: %v\n", serviceErr)
	}
	if daemonState.Error != nil && !daemonState.Reachable {
		fmt.Fprintf(&b, "daemon detail: %v\n", daemonState.Error)
	}
	return b.String()
}

func protocolPause(service serviceController, baseURL, markerPath string) (string, error) {
	if err := writePauseMarker(markerPath); err != nil {
		return "", fmt.Errorf("Prometheus pause failed: %w", err)
	}
	daemonState := readProtocolDaemonState(baseURL)
	if daemonState.Reachable {
		if err := postProtocolDaemon(baseURL, "/v1/pause"); err != nil {
			return "", fmt.Errorf("Prometheus pause is persisted locally, but daemon synchronization failed: %w", err)
		}
		return "Prometheus Protocol paused Pantheon.\nNo new reasoning, external execution, or queued replay will occur.\n", nil
	}
	return "Prometheus Protocol pause persisted locally.\nThe daemon was unavailable; new work will remain paused when it returns.\n", nil
}

func protocolResume(service serviceController, baseURL, markerPath, databasePath string) (string, error) {
	daemonState := readProtocolDaemonState(baseURL)
	if daemonState.Reachable {
		if err := postProtocolDaemon(baseURL, "/v1/resume"); err != nil {
			return "", fmt.Errorf("Prometheus resume failed: %w", err)
		}
		if err := removePauseMarker(markerPath); err != nil {
			return "", err
		}
		return "Prometheus Protocol resumed Pantheon.\nSkipped work will not replay automatically.\n", nil
	}

	ctx, cancel := protocolContext()
	defer cancel()
	serviceState, serviceErr := service.status(ctx)
	if serviceErr != nil {
		return "", fmt.Errorf("daemon unavailable and systemd control is unavailable: %w", serviceErr)
	}
	if err := setPersistedPauseState(databasePath, false); err != nil {
		return "", fmt.Errorf("clear persisted pause state: %w", err)
	}
	if err := removePauseMarker(markerPath); err != nil {
		return "", err
	}
	if serviceState.State != "active" {
		if err := service.start(ctx); err != nil {
			_ = writePauseMarker(markerPath)
			_ = setPersistedPauseState(databasePath, true)
			return "", err
		}
	}
	return "Prometheus Protocol resumed Pantheon through the user service.\nSkipped work will not replay automatically.\n", nil
}

func protocolHardStop(service serviceController, baseURL string) (string, error) {
	ctx, cancel := protocolContext()
	defer cancel()
	// Auditing is best effort. The service-manager operation remains direct so
	// an unavailable or unhealthy daemon cannot prevent emergency control.
	_ = postProtocolDaemonWithTimeout(baseURL, "/v1/prometheus/hard-stop", 250*time.Millisecond)
	if err := service.stop(ctx); err != nil {
		return "", fmt.Errorf("Prometheus hard-stop failed: %w", err)
	}
	return "Prometheus Protocol hard-stopped the Pantheon user service.\nVault, Weave, configuration, audit history, and human source material were not removed.\n", nil
}
