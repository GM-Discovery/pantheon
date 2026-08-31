package pantheon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const omiStorageSyncScript = "omi_storage_sync.py"

type omiSyncNotificationState struct {
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

func (d *daemon) omiStorageSyncScriptPath() (string, error) {
	candidates := []string{
		filepath.Join(filepath.Dir(os.Args[0]), "..", "scripts", omiStorageSyncScript),
		filepath.Join("scripts", omiStorageSyncScript),
	}
	for _, candidate := range candidates {
		path, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, statErr := os.Stat(path); statErr == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("Omi storage sync script %q was not found", omiStorageSyncScript)
}

func trimOmiSyncOutput(output []byte) string {
	const maxOutput = 12000
	value := strings.TrimSpace(string(output))
	if len(value) <= maxOutput {
		return value
	}
	return value[:maxOutput] + "\n[Omi sync output truncated]"
}

func (d *daemon) runOmiStorageSync(ctx context.Context) (string, error) {
	script, err := d.omiStorageSyncScriptPath()
	if err != nil {
		return "", err
	}
	command := []string{
		script,
		"--device", d.cfg.OmiDevice,
		"--adapter", d.cfg.OmiAdapter,
		"--address-type", d.cfg.OmiAddressType,
		"--output-dir", d.cfg.OmiOutputDir,
		"--packet-count", fmt.Sprintf("%d", d.cfg.OmiPacketCount),
		"--max-batches", fmt.Sprintf("%d", d.cfg.OmiMaxBatches),
	}
	process := exec.CommandContext(ctx, "python3", command...)
	process.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	output, runErr := process.CombinedOutput()
	result := trimOmiSyncOutput(output)
	if runErr != nil {
		if result == "" {
			result = runErr.Error()
		}
		return result, fmt.Errorf("Omi storage sync: %w: %s", runErr, result)
	}
	return result, nil
}

func (d *daemon) omiSyncNotificationStatePath() string {
	if strings.TrimSpace(d.cfg.OmiOutputDir) == "" {
		return ""
	}
	return filepath.Join(d.cfg.OmiOutputDir, ".omi-sync-notification-state.json")
}

func (d *daemon) readOmiSyncNotificationState() omiSyncNotificationState {
	path := d.omiSyncNotificationStatePath()
	if path == "" {
		return omiSyncNotificationState{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return omiSyncNotificationState{}
	}
	var state omiSyncNotificationState
	if json.Unmarshal(data, &state) != nil {
		return omiSyncNotificationState{}
	}
	return state
}

func (d *daemon) writeOmiSyncNotificationState(state omiSyncNotificationState) error {
	path := d.omiSyncNotificationStatePath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	partial := path + ".partial"
	if err := os.WriteFile(partial, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(partial, path)
}

func (d *daemon) notifyOmiSyncTransition(state, detail string) {
	previous := d.readOmiSyncNotificationState()
	if previous.State == state && previous.Detail == detail {
		return
	}
	shouldNotify := state == "failed" || previous.State == "failed"
	if !shouldNotify {
		if err := d.writeOmiSyncNotificationState(omiSyncNotificationState{State: state, Detail: detail}); err != nil {
			d.audit("omi.notification", d.cfg.OmiDevice, "notify", "", "failed", "", "could not persist notification state: "+err.Error(), 0)
		}
		return
	}
	title := "Omi storage failure"
	message := "Omi storage sync failed: " + detail + ". Pantheon will retry automatically; inspect the Omi receipt in the configured Omi folder."
	if state != "failed" {
		title = "Omi storage recovered"
		message = "Omi storage sync recovered and is running again."
	}
	callCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(callCtx, notifyCommand, "pantheon-alerts", title, message).CombinedOutput()
	if err != nil {
		d.audit("omi.notification", d.cfg.OmiDevice, "notify", "", "failed", "", fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(output))), 0)
		return
	}
	if err := d.writeOmiSyncNotificationState(omiSyncNotificationState{State: state, Detail: detail}); err != nil {
		d.audit("omi.notification", d.cfg.OmiDevice, "notify", "", "failed", "", "notification sent but state could not be persisted: "+err.Error(), 0)
		return
	}
	d.audit("omi.notification", d.cfg.OmiDevice, "notify", "", "completed", "", "Omi sync transition notification sent", 0)
}

func omiSyncFailureDetail(result string, fallback error) string {
	var receipt struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(result), &receipt) == nil && strings.TrimSpace(receipt.Error) != "" {
		return strings.TrimSpace(receipt.Error)
	}
	if fallback != nil {
		return strings.TrimSpace(fallback.Error())
	}
	return "unknown Omi storage sync failure"
}

func (d *daemon) startOmiStorageWatcher() func() {
	if !d.cfg.OmiEnabled {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Duration(d.cfg.OmiSyncIntervalSeconds) * time.Second
	if interval < time.Minute {
		interval = 15 * time.Minute
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		run := func() {
			paused, pauseErr := d.paused()
			if pauseErr != nil {
				d.audit("omi.watcher", d.cfg.OmiDevice, "omi.storage.sync", "", "failed", "", pauseErr.Error(), 0)
				return
			}
			if paused {
				d.audit("omi.watcher", d.cfg.OmiDevice, "omi.storage.sync", "", "blocked", "", "Prometheus is paused; Omi storage sync was skipped", 0)
				return
			}
			started := time.Now()
			result, err := d.runOmiStorageSync(ctx)
			if err != nil {
				d.audit("omi.watcher", d.cfg.OmiDevice, "omi.storage.sync", "", "failed", result, err.Error(), time.Since(started))
				d.notifyOmiSyncTransition("failed", omiSyncFailureDetail(result, err))
				return
			}
			d.audit("omi.watcher", d.cfg.OmiDevice, "omi.storage.sync", "", "completed", result, "", time.Since(started))
			d.notifyOmiSyncTransition("completed", "completed")
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	return func() {
		cancel()
		wg.Wait()
	}
}
