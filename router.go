package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type routeDecision struct {
	Capability string  `json:"capability"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

type routerRequest struct {
	Model    string         `json:"model"`
	Messages []message      `json:"messages"`
	Stream   bool           `json:"stream"`
	Format   map[string]any `json:"format"`
}

type routerResponse struct {
	Message message `json:"message"`
}

func routeRequest(cfg config, caps capabilityConfig, input string) (routeDecision, error) {
	var names []string

	for _, c := range caps.Capabilities {
		names = append(names, fmt.Sprintf(
			"%s (type=%s, risk=%s)",
			c.Name,
			c.Type,
			c.Risk,
		))
	}

	system := `You are Pantheon's routing engine.

Choose exactly one available capability for the user's request.

Available capabilities:
` + strings.Join(names, "\n") + `

Definitions:
- reason: answer or think using the reasoning model.
- notify: send a notification to the user.
- hermes: delegate work requiring tools, external services, files, email, calendar, scheduled work, or other execution.

Return ONLY valid JSON in this exact shape:
{"capability":"name","reason":"brief explanation","confidence":0.0}

Do not execute the request. Only route it.`

	req := routerRequest{
		Model: cfg.Model,
		Messages: []message{
			{
				Role:    "system",
				Content: system,
			},
			{
				Role:    "user",
				Content: input,
			},
		},
		Stream: false,
		Format: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"capability": map[string]any{
					"type": "string",
					"enum": []string{"reason", "notify", "hermes"},
				},
				"reason": map[string]any{
					"type": "string",
				},
				"confidence": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
			},
			"required":             []string{"capability", "reason", "confidence"},
			"additionalProperties": false,
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return routeDecision{}, err
	}

	resp, err := http.Post(
		cfg.OllamaURL,
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return routeDecision{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return routeDecision{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return routeDecision{}, fmt.Errorf(
			"Ollama returned %s: %s",
			resp.Status,
			string(body),
		)
	}

	var result routerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return routeDecision{}, err
	}

	var decision routeDecision
	if err := json.Unmarshal(
		[]byte(strings.TrimSpace(result.Message.Content)),
		&decision,
	); err != nil {
		return routeDecision{}, fmt.Errorf(
			"router returned invalid JSON: %s",
			result.Message.Content,
		)
	}

	// Deterministic policy gate:
	// the model may only select capabilities that actually exist.
	valid := false
	for _, c := range caps.Capabilities {
		if c.Name == decision.Capability {
			valid = true
			break
		}
	}

	if !valid {
		return routeDecision{}, fmt.Errorf(
			"router selected unknown capability %q",
			decision.Capability,
		)
	}

	return decision, nil
}
