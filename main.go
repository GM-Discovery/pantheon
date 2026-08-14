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

type config struct {
	Model         string `json:"model,omitempty"`
	OllamaURL     string `json:"ollama_url,omitempty"`
	LocalModel    string `json:"local_model,omitempty"`
	FrontierModel string `json:"frontier_model,omitempty"`
	VaultPath     string `json:"vault_path,omitempty"`
	RoadmapPath   string `json:"roadmap_path,omitempty"`
}

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
		fmt.Fprintln(os.Stderr, "usage: pantheon [status|route|notify|hermes|past] \"request\"")
		os.Exit(1)
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
	case "hermes", "past":
		return fmt.Errorf("pantheon %s remains a direct command during this daemon milestone; it is not yet daemon-routed", command)
	default:
		return clientPOST("/v1/input", inputRequest{Input: strings.Join(args, " ")})
	}
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
