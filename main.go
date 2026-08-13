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
	"os/exec"
	"path/filepath"
	"strings"
)

type config struct {
	Model         string `json:"model,omitempty"` // Backwards-compatible alias for LocalModel.
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

func loadConfig() (config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return config{}, err
	}

	path := filepath.Join(home, "pantheon-runtime", "config", "pantheon.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}

	if cfg.LocalModel == "" {
		cfg.LocalModel = cfg.Model
	}
	if cfg.Model == "" {
		cfg.Model = cfg.LocalModel
	}

	return cfg, nil
}

func callHermes(prompt string) (string, error) {
	cmd := exec.Command(
		"/home/brick/.local/bin/hermes",
		"--safe-mode",
		"-z",
		prompt,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Hermes failed: %w\n%s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func notify(topic, title, text string) error {
	cmd := exec.Command(
		"/home/brick/.local/bin/pantheon-notify",
		topic,
		title,
		text,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("notification failed: %w\n%s", err, string(output))
	}

	return nil
}

func loadCapabilities() (capabilityConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return capabilityConfig{}, err
	}

	path := filepath.Join(home, "pantheon-runtime", "config", "capabilities.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return capabilityConfig{}, err
	}

	var cfg capabilityConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return capabilityConfig{}, err
	}

	return cfg, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println(`usage:
  pantheon "your request"
  pantheon status
  pantheon past "what should I search for?"
  pantheon hermes "request"
  pantheon notify "message"`)
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not load Pantheon config: %v\n", err)
		os.Exit(1)
	}

	capabilities, err := loadCapabilities()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not load Pantheon capabilities: %v\n", err)
		os.Exit(1)
	}

	// Deterministic runtime status.
	if os.Args[1] == "status" {
		fmt.Println("Pantheon Runtime")
		fmt.Println("Local model:", cfg.LocalModel)
		fmt.Println("Frontier model:", cfg.FrontierModel)
		fmt.Println("Ollama:", cfg.OllamaURL)
		fmt.Println("Vault:", cfg.VaultPath)
		fmt.Println("Roadmap:", cfg.RoadmapPath)
		fmt.Println("Capabilities:")
		for _, c := range capabilities.Capabilities {
			fmt.Printf("  - %s [%s, risk=%s]\n", c.Name, c.Type, c.Risk)
		}
		return
	}

	// Past has a single authority: searching the configured vault. It never
	// writes to the vault or invokes another daemon.
	if os.Args[1] == "past" {
		if len(os.Args) < 3 {
			fmt.Println(`usage: pantheon past "what should I search for?"`)
			os.Exit(1)
		}

		result, err := searchPast(cfg.VaultPath, strings.Join(os.Args[2:], " "))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Past could not search the vault: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(result)
		return
	}

	if os.Args[1] == "route" {
		if len(os.Args) < 3 {
			fmt.Println(`usage: pantheon route "request"`)
			os.Exit(1)
		}

		input := strings.Join(os.Args[2:], " ")

		decision, err := routeRequest(cfg, capabilities, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "routing failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Capability:", decision.Capability)
		fmt.Println("Confidence:", decision.Confidence)
		fmt.Println("Reason:", decision.Reason)
		return
	}

	if os.Args[1] == "do" {
		if len(os.Args) < 3 {
			fmt.Println(`usage: pantheon do "request"`)
			os.Exit(1)
		}

		input := strings.Join(os.Args[2:], " ")

		decision, err := routeRequest(cfg, capabilities, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "routing failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Pantheon routed to: %s\n", decision.Capability)

		switch decision.Capability {
		case "notify":
			if err := notify(
				"pantheon-alerts",
				"Pantheon",
				input,
			); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Println("Notification sent.")
			return

		case "hermes":
			fmt.Println("Hermes selected, but automatic Hermes execution is not yet authorized.")
			fmt.Println(`Use: pantheon hermes "request"`)
			return

		case "reason":
			// Fall through into Pantheon's normal reasoning path below.
			os.Args = append([]string{os.Args[0]}, strings.Fields(input)...)

		default:
			fmt.Fprintf(os.Stderr, "unsupported capability: %s\n", decision.Capability)
			os.Exit(1)
		}
	}

	// Explicit delegation to Hermes.
	if os.Args[1] == "hermes" {
		if len(os.Args) < 3 {
			fmt.Println(`usage: pantheon hermes "request"`)
			os.Exit(1)
		}

		prompt := strings.Join(os.Args[2:], " ")

		fmt.Println("Pantheon → Hermes")

		result, err := callHermes(prompt)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println(result)
		return
	}

	// Send a notification through Pantheon's ntfy service.
	if os.Args[1] == "notify" {
		if len(os.Args) < 3 {
			fmt.Println(`usage: pantheon notify "message"`)
			os.Exit(1)
		}

		text := strings.Join(os.Args[2:], " ")

		err := notify(
			"pantheon-alerts",
			"Pantheon",
			text,
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("Notification sent.")
		return
	}

	// Normal Pantheon reasoning path.
	userInput := strings.Join(os.Args[1:], " ")

	roadmap, err := os.ReadFile(cfg.RoadmapPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read Pantheon Roadmap: %v\n", err)
		os.Exit(1)
	}

	systemPrompt := fmt.Sprintf(`You are the reasoning engine currently serving Pantheon.

	Pantheon is larger than you. You are a replaceable component inside its runtime.

	Known runtime facts:
	- Current reasoning model: %s
	- Ollama endpoint: %s
	- Pantheon Roadmap: %s

	These runtime facts are authoritative. Do not contradict them or invent alternatives.

	Use the supplied Pantheon Roadmap as governing context. Follow its objective, value hierarchy, non-goals, and operating principles. Do not invent facts not supported by the provided context. Be concise and useful.`,
		cfg.Model,
		cfg.OllamaURL,
		cfg.RoadmapPath,
	)

	reqBody := ollamaRequest{
		Model: cfg.Model,
		Messages: []message{
			{
				Role:    "system",
				Content: systemPrompt + "\n\nPANTHEON ROADMAP:\n" + string(roadmap),
			},
			{
				Role:    "user",
				Content: userInput,
			},
		},
		Stream: true,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not encode request: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.Post(cfg.OllamaURL, "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not reach Ollama: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Ollama returned %s:\n%s\n", resp.Status, body)
		os.Exit(1)
	}

	decoder := json.NewDecoder(resp.Body)

	for {
		var chunk ollamaResponse

		err := decoder.Decode(&chunk)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "\ncould not decode Ollama stream: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(chunk.Message.Content)

		if chunk.Done {
			break
		}
	}

	fmt.Println()
}
