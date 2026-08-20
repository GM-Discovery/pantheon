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

package pantheon

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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

func Run(args []string) {
	if len(args) > 1 && args[1] == "daemon" {
		runDaemon()
		return
	}
	if len(args) < 2 {
		printWelcome()
		return
	}
	if err := runClient(args[1:]); err != nil {
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
	stopRumor := d.startRumorWatcher()
	defer stopRumor()
	stopZodiac := d.startZodiacWatcher()
	defer stopZodiac()
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
	case "help":
		return runHelpCommand(args[1:])
	case "show":
		if len(args) == 2 && args[1] == "codes" {
			return printLocalCodeHelp()
		}
		if len(args) == 2 && args[1] == "organism" {
			return clientGET("/v1/organism")
		}
		if len(args) == 2 && args[1] == "aphrodite" {
			return clientGET("/v1/aphrodite")
		}
		if len(args) == 2 && args[1] == "echo" {
			return clientGET("/v1/echo")
		}
		if len(args) == 3 && args[1] == "rumor" && args[2] == "queue" {
			return clientGET("/v1/rumor/queue")
		}
		if len(args) == 4 && args[1] == "rumor" && args[2] == "source" {
			return clientGET("/v1/rumor/sources/" + url.PathEscape(args[3]))
		}
		if len(args) == 3 && args[1] == "order" {
			return clientGET("/v1/orders/" + url.PathEscape(args[2]))
		}
		if len(args) == 3 && args[1] == "grant" {
			return clientGET("/v1/grants/" + url.PathEscape(args[2]))
		}
		if len(args) == 3 && args[1] == "apollo" && args[2] == "reports" {
			return clientGET("/v1/apollo/reports")
		}
		if len(args) == 3 && args[1] == "future" && args[2] == "proposals" {
			return clientGET("/v1/future/proposals")
		}
		if len(args) == 2 && args[1] == "mail" {
			return clientGET("/v1/mail")
		}
		if len(args) == 3 && args[1] == "mail" && args[2] == "messages" {
			return clientGET("/v1/mail/messages")
		}
		if len(args) == 3 && args[1] == "mail" && args[2] == "rules" {
			return clientGET("/v1/mail/rules")
		}
		if len(args) == 2 && args[1] == "zodiac" {
			return clientGET("/v1/zodiac")
		}
		if len(args) == 4 && args[1] == "apollo" && args[2] == "report" {
			return clientGET("/v1/apollo/reports/" + url.PathEscape(args[3]))
		}
		if len(args) == 4 && args[1] == "future" && args[2] == "proposal" {
			return clientGET("/v1/future/proposals/" + url.PathEscape(args[3]))
		}
		if len(args) == 3 && args[1] == "package" {
			return clientGET("/v1/seshat/packages/" + url.PathEscape(args[2]))
		}
		if len(args) == 3 && args[1] == "transition" {
			return clientGET("/v1/janus/transitions/" + url.PathEscape(args[2]))
		}
		if len(args) != 2 {
			return fmt.Errorf("usage: pantheon show [config|intake|father-time|present|processors|rumor|rumor queue|rumor source <id>|mail|mail messages|mail rules|zodiac|aphrodite|echo|orders|order <id>|grant <id>|plugins|packages|codes|organism|package <subject-code>|janus|transitions|transition <id>|mercury|apollo|future|bifrost|future proposals|future proposal <id>]")
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
		case "rumor":
			return clientGET("/v1/rumor")
		case "orders":
			return clientGET("/v1/orders")
		case "mercury":
			return clientGET("/v1/mercury")
		case "janus":
			return clientGET("/v1/janus")
		case "transitions":
			return clientGET("/v1/janus/transitions")
		case "apollo":
			return clientGET("/v1/apollo")
		case "future":
			return clientGET("/v1/future")
		case "bifrost":
			return clientGET("/v1/bifrost")
		case "mail":
			return clientGET("/v1/mail")
		case "zodiac":
			return clientGET("/v1/zodiac")
		case "aphrodite":
			return clientGET("/v1/aphrodite")
		case "echo":
			return clientGET("/v1/echo")
		case "plugins":
			return clientGET("/v1/seshat/plugins")
		case "packages":
			return clientGET("/v1/seshat/packages")
		default:
			return fmt.Errorf("usage: pantheon show [config|intake|father-time|present|processors|rumor|rumor queue|rumor source <id>|mail|mail messages|mail rules|zodiac|aphrodite|echo|orders|order <id>|grant <id>|plugins|packages|codes|organism|package <subject-code>|janus|transitions|transition <id>|mercury|apollo|future|bifrost|future proposals|future proposal <id>]")
		}
	case "intake":
		if len(args) != 2 || args[1] != "scan" {
			return fmt.Errorf("usage: pantheon intake scan")
		}
		return clientPOST("/v1/intake/scan", nil)
	case "config":
		return runConfigCommand(args[1:])
	case "plugin":
		return runPluginCommand(args[1:])
	case "bifrost":
		return runBifrostCommand(args[1:])
	case "present":
		return runPresentCommand(args[1:])
	case "rumor":
		return runRumorCommand(args[1:])
	case "mail":
		return runMailCommand(args[1:])
	case "zodiac":
		return runZodiacCommand(args[1:])
	case "order":
		return runOrderCommand(args[1:])
	case "mercury":
		if len(args) != 2 || args[1] != "round" {
			return fmt.Errorf("usage: pantheon mercury round")
		}
		return clientPOSTWithControl("/v1/mercury/round", nil)
	case "janus":
		if len(args) != 2 || args[1] != "round" {
			return fmt.Errorf("usage: pantheon janus round")
		}
		return clientPOSTWithControl("/v1/janus/round", nil)
	case "future":
		if len(args) != 2 || args[1] != "round" {
			return fmt.Errorf("usage: pantheon future round")
		}
		return clientPOSTWithControl("/v1/future/round", nil)
	case "organism":
		return runOrganismCommand(args[1:])
	case "aphrodite":
		return runAphroditeCommand(args[1:])
	case "echo":
		return runEchoCommand(args[1:])
	case "/echo":
		return runEchoCommand(args[1:])
	case "prometheus":
		return runPrometheusCommand(args[1:])
	case "status":
		return clientGET("/v1/status")
	case "system":
		if len(args) != 2 || args[1] != "inventory" {
			return fmt.Errorf("usage: pantheon system inventory")
		}
		return clientGET("/v1/system/inventory")
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
	case "authority":
		if len(args) != 4 || args[1] != "check" {
			return fmt.Errorf("usage: pantheon authority check <capability> <scope>")
		}
		return clientGET("/v1/authority/check?capability=" + url.QueryEscape(args[2]) + "&scope=" + url.QueryEscape(args[3]))
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

func runAphroditeCommand(args []string) error {
	if len(args) == 0 || args[0] == "status" {
		if len(args) > 1 {
			return fmt.Errorf("usage: pantheon aphrodite [status|project|phase|artifact|research|fame-fixture]")
		}
		return clientGET("/v1/aphrodite")
	}
	switch args[0] {
	case "project":
		if len(args) != 2 {
			return fmt.Errorf("usage: pantheon aphrodite project <id>")
		}
		return clientGET("/v1/aphrodite/projects/" + url.PathEscape(args[1]))
	case "phase":
		if len(args) < 2 {
			return fmt.Errorf("usage: pantheon aphrodite phase <project-id> --phase <phase> [--state <state>] [--result <text>]")
		}
		request := aphroditePhaseRequest{}
		for i := 2; i < len(args); i++ {
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			switch args[i] {
			case "--phase":
				request.Phase = args[i+1]
			case "--state":
				request.State = args[i+1]
			case "--result":
				request.LastResult = args[i+1]
			default:
				return fmt.Errorf("unknown Aphrodite phase option %q", args[i])
			}
			i++
		}
		return clientPOST("/v1/aphrodite/projects/"+url.PathEscape(args[1])+"/phase", request)
	case "artifact":
		if len(args) < 2 {
			return fmt.Errorf("usage: pantheon aphrodite artifact <project-id> --kind <kind> --title <title> --content <text>")
		}
		request := aphroditeArtifactRequest{}
		for i := 2; i < len(args); i++ {
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			switch args[i] {
			case "--kind":
				request.Kind = args[i+1]
			case "--title":
				request.Title = args[i+1]
			case "--content":
				request.Content = args[i+1]
			case "--status":
				request.Status = args[i+1]
			default:
				return fmt.Errorf("unknown Aphrodite artifact option %q", args[i])
			}
			i++
		}
		return clientPOST("/v1/aphrodite/projects/"+url.PathEscape(args[1])+"/artifacts", request)
	case "research":
		if len(args) < 2 {
			return fmt.Errorf("usage: pantheon aphrodite research <project-id> --question <text> [--rationale <text>] [--brief]")
		}
		request := aphroditeResearchRequestRequest{}
		for i := 2; i < len(args); i++ {
			if args[i] == "--brief" {
				request.SurfaceBrief = true
				continue
			}
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			switch args[i] {
			case "--question":
				request.Question = args[i+1]
			case "--rationale":
				request.Rationale = args[i+1]
			default:
				return fmt.Errorf("unknown Aphrodite research option %q", args[i])
			}
			i++
		}
		return clientPOST("/v1/aphrodite/projects/"+url.PathEscape(args[1])+"/research-requests", request)
	case "fame-fixture":
		if len(args) < 2 {
			return fmt.Errorf("usage: pantheon aphrodite fame-fixture <project-id> --title <title> [--transcript <text>] [--rumor-source <id>]")
		}
		request := aphroditeFixtureRequest{}
		for i := 2; i < len(args); i++ {
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			switch args[i] {
			case "--title":
				request.Title = args[i+1]
			case "--transcript":
				request.Transcript = args[i+1]
			case "--rumor-source":
				request.RumorSourceID = args[i+1]
			case "--start":
				request.CitationStart = args[i+1]
			case "--end":
				request.CitationEnd = args[i+1]
			default:
				return fmt.Errorf("unknown Fame fixture option %q", args[i])
			}
			i++
		}
		return clientPOST("/v1/aphrodite/projects/"+url.PathEscape(args[1])+"/fame/fixtures", request)
	default:
		return fmt.Errorf("usage: pantheon aphrodite [status|project|phase|artifact|research|fame-fixture]")
	}
}

func runEchoCommand(args []string) error {
	if len(args) == 0 || args[0] == "voices" || args[0] == "status" {
		if len(args) > 1 {
			return fmt.Errorf("usage: pantheon echo [voices|voice add|voice enroll|import|confirm]")
		}
		return clientGET("/v1/echo")
	}
	switch args[0] {
	case "voice":
		if len(args) >= 2 && args[1] == "add" {
			if len(args) != 3 {
				return fmt.Errorf("usage: pantheon echo voice add <name>")
			}
			return clientPOST("/v1/echo/voices", echoVoiceRequest{Name: args[2]})
		}
		if len(args) >= 2 && args[1] == "enroll" {
			if len(args) < 4 {
				return fmt.Errorf("usage: pantheon echo voice enroll <voice-id> <audio-path> [--start seconds --end seconds]")
			}
			req := echoEnrollRequest{SourcePath: args[3]}
			for i := 4; i < len(args); i++ {
				if i+1 >= len(args) {
					return fmt.Errorf("missing value for %s", args[i])
				}
				value, err := strconv.ParseFloat(args[i+1], 64)
				if err != nil {
					return fmt.Errorf("invalid seconds %q", args[i+1])
				}
				switch args[i] {
				case "--start":
					req.StartSeconds = value
				case "--end":
					req.EndSeconds = value
				default:
					return fmt.Errorf("unknown option %s", args[i])
				}
				i++
			}
			return clientPOST("/v1/echo/voices/"+url.PathEscape(args[2])+"/samples", req)
		}
		return fmt.Errorf("usage: pantheon echo voice [add <name>|enroll <voice-id> <audio-path>]")
	case "import":
		if len(args) < 3 {
			return fmt.Errorf("usage: pantheon echo import <audio-path> <diarization.tsv> [--speakers count]")
		}
		req := echoImportRequest{SourcePath: args[1], DiarizationPath: args[2]}
		for i := 3; i < len(args); i++ {
			if i+1 >= len(args) || args[i] != "--speakers" {
				return fmt.Errorf("usage: pantheon echo import <audio-path> <diarization.tsv> [--speakers count]")
			}
			count, err := strconv.Atoi(args[i+1])
			if err != nil || count < 1 {
				return fmt.Errorf("invalid speaker count")
			}
			req.SpeakerCount = count
			i++
		}
		return clientPOST("/v1/echo/runs/import", req)
	case "confirm":
		if len(args) != 4 {
			return fmt.Errorf("usage: pantheon echo confirm <voice-id> <run-id> <anonymous-speaker>")
		}
		return clientPOST("/v1/echo/voices/"+url.PathEscape(args[1])+"/confirm", echoConfirmRequest{RunID: args[2], Speaker: args[3]})
	default:
		return fmt.Errorf("usage: pantheon echo [voices|voice add|voice enroll|import|confirm]")
	}
}

func runMailCommand(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: pantheon mail [check|status|messages|rules|actions]")
	}
	switch args[0] {
	case "check":
		return clientPOSTWithControl("/v1/mail/round", nil)
	case "status":
		return clientGET("/v1/mail")
	case "messages":
		return clientGET("/v1/mail/messages")
	case "rules":
		return clientGET("/v1/mail/rules")
	case "actions":
		return clientGET("/v1/mail/actions")
	default:
		return fmt.Errorf("usage: pantheon mail [check|status|messages|rules|actions]")
	}
}

func runZodiacCommand(args []string) error {
	if len(args) == 0 || args[0] == "status" {
		if len(args) > 1 {
			return fmt.Errorf("usage: pantheon zodiac [status|round|today|upcoming|needs-you|schedule ...]")
		}
		return clientGET("/v1/zodiac")
	}
	switch args[0] {
	case "round":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon zodiac round")
		}
		return clientPOSTWithControl("/v1/zodiac/round", nil)
	case "today":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon zodiac today")
		}
		return clientGET("/v1/zodiac/today")
	case "upcoming":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon zodiac upcoming")
		}
		return clientGET("/v1/zodiac/upcoming")
	case "needs-you":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon zodiac needs-you")
		}
		return clientGET("/v1/zodiac/needs-you")
	case "schedule":
		request := zodiacScheduleRequest{}
		for i := 1; i < len(args); i++ {
			if !strings.HasPrefix(args[i], "--") || i+1 >= len(args) {
				return fmt.Errorf("usage: pantheon zodiac schedule --name <name> --kind <kind> --start <RFC3339|YYYY-MM-DD> --action <action> [--recurrence <RRULE>] [--timezone <zone>] [--catch-up <policy>] [--projection <policy>] [--source <source>]")
			}
			value := args[i+1]
			switch args[i] {
			case "--name":
				request.Name = value
			case "--kind":
				request.Kind = value
			case "--start":
				request.StartAt = value
			case "--action":
				request.Action = value
			case "--recurrence":
				request.Recurrence = value
			case "--timezone":
				request.Timezone = value
			case "--catch-up":
				request.CatchUpPolicy = value
			case "--projection":
				request.ProjectionPolicy = value
			case "--source":
				request.Source = value
			case "--priority":
				request.Priority = value
			case "--dependency":
				request.Dependency = json.RawMessage(value)
			case "--payload":
				request.Payload = json.RawMessage(value)
			default:
				return fmt.Errorf("unknown zodiac schedule option %q", args[i])
			}
			i++
		}
		return clientPOSTWithControl("/v1/zodiac/schedules", request)
	default:
		return fmt.Errorf("usage: pantheon zodiac [status|round|today|upcoming|needs-you|schedule ...]")
	}
}

func runBifrostCommand(args []string) error {
	if len(args) == 0 || args[0] == "status" {
		if len(args) > 1 {
			return fmt.Errorf("usage: pantheon bifrost [status|list|show <capability>|approvals|invocations|register ...|enable <capability>|disable <capability>|mail status|mail intake]")
		}
		return clientGET("/v1/bifrost/status")
	}
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon bifrost list")
		}
		return clientGET("/v1/bifrost/capabilities")
	case "show":
		if len(args) != 2 {
			return fmt.Errorf("usage: pantheon bifrost show <capability>")
		}
		return clientGET("/v1/bifrost/capabilities/" + url.PathEscape(args[1]))
	case "approvals":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon bifrost approvals")
		}
		return clientGET("/v1/bifrost/approvals")
	case "invocations":
		if len(args) != 1 {
			return fmt.Errorf("usage: pantheon bifrost invocations")
		}
		return clientGET("/v1/bifrost/invocations")
	case "enable", "disable":
		if len(args) != 2 {
			return fmt.Errorf("usage: pantheon bifrost %s <capability>", args[0])
		}
		path := "/v1/bifrost/capabilities/" + url.PathEscape(args[1]) + "/" + args[0]
		return clientPOSTWithControl(path, nil)
	case "register":
		return runBifrostRegisterCommand(args[1:])
	case "mail":
		if len(args) == 2 && args[1] == "status" {
			return clientGET("/v1/bifrost/mail/status")
		}
		if len(args) < 2 || args[1] != "intake" {
			return fmt.Errorf("usage: pantheon bifrost mail [status|intake [--capability <id>] [--account <name>] [--mailbox <name>] [--max <n>] [--bootstrap]]")
		}
		request := bifrostMailIntakeRequest{}
		for i := 2; i < len(args); i++ {
			if args[i] == "--bootstrap" {
				request.Bootstrap = true
				continue
			}
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			value := args[i+1]
			switch args[i] {
			case "--capability":
				request.CapabilityID = value
			case "--account":
				request.Account = value
			case "--mailbox":
				request.Mailbox = value
			case "--max":
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid --max: %w", err)
				}
				request.MaxMessages = parsed
			default:
				return fmt.Errorf("unknown Bifrost mail option %q", args[i])
			}
			i++
		}
		return clientPOSTWithControl("/v1/bifrost/mail/intake", request)
	default:
		return fmt.Errorf("usage: pantheon bifrost [status|list|show <capability>|approvals|invocations|register ...|enable <capability>|disable <capability>|mail status|mail intake]")
	}
}

func runBifrostRegisterCommand(args []string) error {
	request := bifrostRegistrationRequest{Type: bifrostCapabilityTypeCommand, Installed: true}
	for i := 0; i < len(args); i++ {
		if args[i] == "--download-approved" {
			request.DownloadApproved = true
			continue
		}
		if args[i] == "--installation-approved" {
			request.InstallationApproved = true
			continue
		}
		if args[i] == "--use-approved" {
			request.UseApproved = true
			continue
		}
		if args[i] == "--auto-update-approved" {
			request.AutoUpdateApproved = true
			continue
		}
		if args[i] == "--network" {
			request.NetworkAccess = true
			continue
		}
		if args[i] == "--sends-data-off-machine" {
			request.SendsDataOffMachine = true
			continue
		}
		if args[i] == "--authentication-required" {
			request.AuthenticationRequired = true
			continue
		}
		if i+1 >= len(args) {
			return fmt.Errorf("missing value for %s", args[i])
		}
		value := args[i+1]
		switch args[i] {
		case "--id":
			request.CapabilityID = value
		case "--name":
			request.Name = value
		case "--type":
			request.Type = value
		case "--source":
			request.Source = value
		case "--version":
			request.Version = value
		case "--description":
			request.Description = value
		case "--invocation":
			request.Invocation = value
		case "--operation":
			request.Operations = append(request.Operations, value)
		case "--resource":
			request.Resources = append(request.Resources, value)
		case "--component":
			request.PantheonComponents = append(request.PantheonComponents, value)
		case "--prohibited-operation":
			request.ProhibitedOperations = append(request.ProhibitedOperations, value)
		default:
			return fmt.Errorf("unknown Bifrost register option %q", args[i])
		}
		i++
	}
	if request.Name == "" || request.Source == "" || request.Version == "" {
		return fmt.Errorf("usage: pantheon bifrost register --name <name> --source <source> --version <version> [--id <id>] [--type <type>] [--description <text>] [--invocation <text>] [--operation <name>] [--prohibited-operation <name>] [--resource <scope>] [--component <name>] [--network] [--sends-data-off-machine] [--authentication-required] [--download-approved] [--installation-approved] [--use-approved] [--auto-update-approved]")
	}
	return clientPOSTWithControl("/v1/bifrost/capabilities", request)
}

func runOrganismCommand(args []string) error {
	if len(args) == 0 || args[0] != "run" {
		return fmt.Errorf("usage: pantheon organism run --request <text> --package <subject-code> --scope <scope> --missive <Seshat missive> [--occurrence <id>] [--janus-subject <id>]")
	}
	request := organismRunRequest{}
	for i := 1; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") || i+1 >= len(args) {
			return fmt.Errorf("usage: pantheon organism run --request <text> --package <subject-code> --scope <scope> --missive <Seshat missive> [--occurrence <id>] [--janus-subject <id>]")
		}
		value := args[i+1]
		switch args[i] {
		case "--request":
			request.RequestText = value
		case "--package":
			request.PackageCode = value
		case "--scope":
			request.Scope = value
		case "--missive":
			request.Missive = value
		case "--occurrence":
			request.OccurrenceID = value
		case "--janus-subject":
			request.JanusSubject = value
		case "--purpose":
			request.Purpose = value
		default:
			return fmt.Errorf("unknown organism option %q", args[i])
		}
		i++
	}
	return clientPOSTWithControl("/v1/organism/run", request)
}

func runPluginCommand(args []string) error {
	if len(args) == 1 && args[0] == "help" {
		return clientGET("/v1/seshat/help")
	}
	if len(args) == 1 && args[0] == "schema" {
		return clientGET("/v1/seshat/schema")
	}
	if len(args) == 2 && args[0] == "validate" {
		return clientPOST("/v1/seshat/plugins/validate", seshatPathRequest{Path: args[1]})
	}
	if len(args) == 2 && args[0] == "register" {
		return clientPOSTWithControl("/v1/seshat/plugins/register", seshatPathRequest{Path: args[1]})
	}
	return fmt.Errorf("usage: pantheon plugin [help|schema|validate <path>|register <path>]")
}

func runRumorCommand(args []string) error {
	if len(args) == 1 && args[0] == "scan" {
		return clientPOSTWithControl("/v1/rumor/scan", nil)
	}
	if len(args) == 2 && args[0] == "transcribe" {
		return clientPOSTWithControl("/v1/rumor/transcribe", rumorTranscribeRequest{Path: args[1]})
	}
	if len(args) == 2 && args[0] == "retry" {
		return clientPOSTWithControl("/v1/rumor/sources/"+url.PathEscape(args[1])+"/retry", nil)
	}
	if len(args) == 2 && args[0] == "ignore" {
		return clientPOSTWithControl("/v1/rumor/sources/"+url.PathEscape(args[1])+"/ignore", nil)
	}
	return fmt.Errorf("usage: pantheon rumor [scan|transcribe <audio-path>|retry <source-id>|ignore <source-id>]")
}

func runOrderCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pantheon order \"request\" | pantheon order --recipient <name> --capability <name> --scope <scope> \"request\" | pantheon order cancel <id>")
	}
	if args[0] == "cancel" {
		if len(args) != 2 || strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("usage: pantheon order cancel <id>")
		}
		return clientPOSTWithControl("/v1/orders/"+url.PathEscape(args[1])+"/cancel", nil)
	}
	request := orderRequest{}
	var text []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", args[i])
			}
			value := args[i+1]
			switch args[i] {
			case "--recipient":
				request.Recipient = value
			case "--capability":
				request.Capability = value
			case "--scope":
				request.Scope = value
			case "--parent":
				request.ParentOrderID = value
			case "--parent-seal":
				request.ParentSealDigest = value
			case "--purpose":
				request.Purpose = value
			default:
				return fmt.Errorf("unknown order option %q", args[i])
			}
			i++
			continue
		}
		text = append(text, args[i])
	}
	if len(text) == 0 {
		return errors.New("order request text is required")
	}
	request.RequestText = strings.Join(text, " ")
	return clientPOSTWithControl("/v1/orders", request)
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
	// Control-boundary operations may include a bounded local Apollo run. Keep
	// the client deadline aligned with the daemon's 30-minute integration
	// bound so a slow local model is not canceled by the CLI first.
	resp, err := (&http.Client{Timeout: 30 * time.Minute}).Do(req)
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
