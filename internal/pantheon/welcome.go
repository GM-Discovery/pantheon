package pantheon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const welcomeWidth = 78

var ansiEscape = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

type welcomeState struct {
	Runtime string
	Model   string
	Vault   string
	Paused  bool
}

func printWelcome() {
	state := welcomeState{Runtime: "daemon unavailable; metadata help remains available", Model: "local model configured by runtime", Vault: "Vault path available through `pantheon status`"}
	client := &http.Client{Timeout: 300 * time.Millisecond}
	resp, err := client.Get(daemonURL + "/v1/status")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode < 300 {
			var status statusResponse
			if json.NewDecoder(resp.Body).Decode(&status) == nil {
				state.Runtime = "daemon alive at " + status.ListenAddress
				state.Model = status.LocalModel
				state.Vault = status.VaultPath
				state.Paused = status.Paused
			}
		}
	}
	fmt.Println(renderWelcome(state))
}

func renderWelcome(state welcomeState) string {
	cyan, red, green, white, dim, reset := "\033[1;96m", "\033[1;91m", "\033[1;92m", "\033[1;97m", "\033[2m", "\033[0m"
	if os.Getenv("NO_COLOR") != "" {
		cyan, red, green, white, dim, reset = "", "", "", "", "", ""
	}
	stop := "STOP"
	if state.Paused {
		stop = "PAUSED"
	}
	lines := []string{
		"",
		centerWelcome(cyan + "P A N T H E O N" + reset),
		centerWelcome(dim + "LOCAL INTELLIGENCE · BOUND IN SERVICE" + reset),
		"",
		centerWelcome("╭────────────╮"),
		centerWelcome("│  " + red + "■ " + stop + reset + "    │"),
		centerWelcome("╰────────────╯"),
		"",
		centerWelcome(white + "IN CASE OF EMERGENCY" + reset),
		"",
		"  " + dim + "Prometheus Protocol" + reset,
		"  " + green + "pantheon prometheus protocol --pause" + reset,
		"  Pauses Pantheon and autonomous activity for inspection.",
		"  Need commands?  pantheon help --commands",
		"",
		"  Runtime: " + state.Runtime,
		"  Model:   " + state.Model,
		"  Vault:   " + state.Vault,
		"",
		"  Start with: pantheon help",
		"",
	}
	var b strings.Builder
	b.WriteString("╭" + strings.Repeat("─", welcomeWidth-2) + "╮\n")
	for _, line := range lines {
		b.WriteString(welcomeBoxLine(line))
		b.WriteByte('\n')
	}
	b.WriteString("╰" + strings.Repeat("─", welcomeWidth-2) + "╯")
	return b.String()
}

func centerWelcome(value string) string {
	max := welcomeWidth - 4
	value = fitWelcome(value, max)
	padding := max - visibleWidth(value)
	left := padding / 2
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", padding-left)
}

func welcomeBoxLine(value string) string {
	inner := welcomeWidth - 4
	value = fitWelcome(value, inner)
	return "│ " + value + strings.Repeat(" ", inner-visibleWidth(value)) + " │"
}

func visibleWidth(value string) int {
	return len([]rune(ansiEscape.ReplaceAllString(value, "")))
}

func fitWelcome(value string, max int) string {
	if visibleWidth(value) <= max {
		return value
	}
	plain := ansiEscape.ReplaceAllString(value, "")
	runes := []rune(plain)
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}
