// Value Function (Conceptual)
//
// V(person) ≈ (A × R × C) / P
//
// A = Agency (ability to act and be accountable)
// R = Relational continuity (history + context across time)
// C = Capacity for correction (ability to responsibly override)
// P = Replaceability (→ 0 for true identity)
//
// As P → 0, value → ∞.
//
// A person is not a user account.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ollamaRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}
type ollamaResponse struct {
	Message message `json:"message"`
	Done    bool    `json:"done"`
}
type capability struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
	Model   string `json:"model,omitempty"`
	Risk    string `json:"risk"`
}
type capabilityConfig struct {
	Capabilities []capability `json:"capabilities"`
}

const daemonURL = "http://127.0.0.1:8765"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		runDaemon()
		return
	}
	if len(os.Args) < 2 {
		printWelcome()
		return
	}
	if err := runClient(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runDaemon() {
	opts, err := defaultDaemonOptions()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d, err := newDaemon(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Pantheon daemon startup failed:", err)
		os.Exit(1)
	}
	defer d.Close()
	server := &http.Server{Addr: defaultListenAddress, Handler: d.Handler(), ReadHeaderTimeout: 5 * time.Second}
	fmt.Println("Pantheon daemon listening on", defaultListenAddress)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runClient(args []string) error {
	command := args[0]
	switch command {
	case "show":
		if len(args) != 2 {
			return fmt.Errorf("usage: pantheon show [config|intake|father-time|present|processors]")
		}
		switch args[1] {
		case "config":
			return runShowConfig()
		case "intake":
			return clientGET("/v1/intake")
		case "father-time":
			return clientGET("/v1/father-time")
		case "present":
			return clientGET("/v1/present")
		case "processors":
			return clientGET("/v1/processors")
		default:
			return fmt.Errorf("usage: pantheon show [config|intake|father-time|present|processors]")
		}
	case "intake":
		if len(args) != 2 || args[1] != "scan" {
			return fmt.Errorf("usage: pantheon intake scan")
		}
		return clientPOST("/v1/intake/scan", nil)
	case "config":
		return runConfigCommand(args[1:])
	case "present":
		return runPresentCommand(args[1:])
	case "prometheus":
		return runPrometheusCommand(args[1:])
	case "status":
		return clientGET("/v1/status")
	case "route":
		if len(args) < 2 {
			return fmt.Errorf("usage: pantheon route \"request\"")
		}
		return clientPOST("/v1/route", inputRequest{Input: strings.Join(args[1:], " ")})
	case "notify":
		if len(args) < 2 {
			return fmt.Errorf("usage: pantheon notify \"message\"")
		}
		return clientPOST("/v1/notify", inputRequest{Input: strings.Join(args[1:], " ")})
	case "pause":
		return clientPOST("/v1/pause", nil)
	case "resume":
		return clientPOST("/v1/resume", nil)
	case "grants":
		return clientGET("/v1/grants")
	case "grant":
		if len(args) != 4 {
			return fmt.Errorf("usage: pantheon grant <capability> <scope> <intent-summary>")
		}
		return clientPOSTWithControl("/v1/grants", grantRequest{Capability: args[1], Scope: args[2], IntentSummary: args[3], HumanApproved: true})
	case "weave-review":
		if len(args) < 2 {
			return fmt.Errorf("usage: pantheon weave-review <subject>")
		}
		return clientPOSTMarkdown("/v1/weave/review", weaveRetrieveRequest{Query: strings.Join(args[1:], " ")})
	case "weave-refresh":
		return clientPOSTWithControl("/v1/weave/refresh", nil)
	case "hermes", "past":
		return fmt.Errorf("pantheon %s remains a direct command during this daemon milestone; it is not yet daemon-routed", command)
	default:
		return clientPOST("/v1/input", inputRequest{Input: strings.Join(args, " ")})
	}
}

func runPresentCommand(args []string) error {
	if len(args) == 1 && args[0] == "--new" {
		return clientPOST("/v1/present", presentRequest{ProcessNew: true})
	}
	if len(args) == 2 && args[0] == "--occurrence" {
		if strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("usage: pantheon present --occurrence <occurrence-id>")
		}
		return clientPOST("/v1/present", presentRequest{OccurrenceID: args[1]})
	}
	if len(args) == 0 || (len(args) > 0 && (args[0] == "--new" || args[0] == "--occurrence")) {
		return fmt.Errorf("usage: pantheon present \"request\" | pantheon present --occurrence <occurrence-id> | pantheon present --new")
	}
	return clientPOST("/v1/present", presentRequest{Request: strings.Join(args, " ")})
}

func runShowConfig() error {
	path, err := configPathFromDefaults()
	if err != nil {
		return err
	}
	resolution, err := validateConfigFile(path)
	if err != nil {
		return fmt.Errorf("show config: %w", err)
	}
	fmt.Print(renderResolvedConfig(path, resolution))
	return nil
}

func runConfigCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pantheon config [validate|path|migrate [--apply]]")
	}
	path, err := configPathFromDefaults()
	if err != nil {
		return err
	}
	switch args[0] {
	case "path":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon config path")
		}
		fmt.Println(path)
		return nil
	case "validate":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon config validate")
		}
		resolution, err := validateConfigFile(path)
		if err != nil {
			return fmt.Errorf("configuration invalid: %w", err)
		}
		if resolution.Legacy {
			fmt.Printf("Configuration is structurally valid as legacy flat configuration.\nRun `pantheon config migrate` to preview the current schema.\n")
		} else {
			fmt.Printf("Configuration is structurally valid (schema_version %d).\n", resolution.SchemaVersion)
		}
		return nil
	case "migrate":
		apply := false
		if len(args) == 2 {
			if args[1] != "--apply" {
				return fmt.Errorf("usage: pantheon config migrate [--apply]")
			}
			apply = true
		} else if len(args) != 1 {
			return fmt.Errorf("usage: pantheon config migrate [--apply]")
		}
		result, err := migrateConfig(path, apply)
		if err != nil {
			return err
		}
		fmt.Print(formatMigrationResult(result, apply))
		return nil
	default:
		return fmt.Errorf("unknown config command %q; expected validate, path, or migrate", args[0])
	}
}

func runPrometheusCommand(args []string) error {
	if len(args) != 2 || args[0] != "protocol" {
		return fmt.Errorf("usage: pantheon prometheus protocol [--status|--pause|--resume|--hard-stop]")
	}
	action := strings.TrimPrefix(args[1], "--")
	if action != "status" && action != "pause" && action != "resume" && action != "hard-stop" {
		return fmt.Errorf("usage: pantheon prometheus protocol [--status|--pause|--resume|--hard-stop]")
	}
	result, err := runPrometheusProtocol(action)
	if result != "" {
		fmt.Print(result)
	}
	return err
}

func clientGET(path string) error {
	resp, err := (&http.Client{Timeout: 90 * time.Second}).Get(daemonURL + path)
	if err != nil {
		return fmt.Errorf("Pantheon daemon unavailable at %s: %w", daemonURL, err)
	}
	defer resp.Body.Close()
	return printClientResponse(resp)
}

func clientPOSTWithControl(path string, value any) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	token, err := os.ReadFile(filepath.Join(home, "pantheon-runtime", "state", "control.token"))
	if err != nil {
		return fmt.Errorf("read local Pantheon control token: %w", err)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, daemonURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(controlTokenHeader, strings.TrimSpace(string(token)))
	resp, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("Pantheon daemon unavailable at %s: %w", daemonURL, err)
	}
	defer resp.Body.Close()
	return printClientResponse(resp)
}

func clientPOST(path string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Post(daemonURL+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("Pantheon daemon unavailable at %s: %w", daemonURL, err)
	}
	defer resp.Body.Close()
	return printClientResponse(resp)
}

func clientPOSTMarkdown(path string, value any) error { return clientPOST(path, value) }

func printClientResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("daemon returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var input inputResponse
	if json.Unmarshal(body, &input) == nil && input.Result != "" {
		fmt.Println(input.Result)
		return nil
	}
	var markdown struct {
		Markdown string `json:"markdown"`
	}
	if json.Unmarshal(body, &markdown) == nil && markdown.Markdown != "" {
		fmt.Println(markdown.Markdown)
		return nil
	}
	var result map[string]string
	if json.Unmarshal(body, &result) == nil && result["result"] != "" {
		fmt.Println(result["result"])
		return nil
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		fmt.Println(pretty.String())
		return nil
	}
	fmt.Println(string(body))
	return nil
}
