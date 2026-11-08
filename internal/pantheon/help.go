package pantheon

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type commandHelpEntry struct {
	Group   string
	Command string
	Purpose string
	Args    string
	Mutates string
	Control string
}

var pantheonCommands = []commandHelpEntry{
	{Group: "STATUS / INSPECTION", Command: "pantheon status", Purpose: "show runtime state and configured local capabilities", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon system inventory", Purpose: "show the secret-free Kernel 20 inventory contract for operators and the GUI", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show <area>", Purpose: "inspect Intake, Present, processors, orders, Apollo, Archimedes, Future, Janus, Mercury, or packages", Args: "area", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show janus", Purpose: "inspect temporal subjects and transition readiness", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show order <id>", Purpose: "inspect one durable order and its evidence", Args: "id", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show apollo report <id>", Purpose: "read one durable cited Apollo report", Args: "id", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show future proposal <id>", Purpose: "inspect Future authority and execution state", Args: "id", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show organism", Purpose: "inspect whole-animal integration traces", Mutates: "no"},
	{Group: "APHRODITE / PROJECTS", Command: "pantheon show aphrodite", Purpose: "inspect Aphrodite projects, Project Fame fixtures, research requests, and workshop boundaries", Mutates: "no"},
	{Group: "APHRODITE / PROJECTS", Command: "pantheon aphrodite phase <project-id> ...", Purpose: "advance or pause a project lifecycle phase", Args: "project-id --phase <phase> [--state <state>]", Mutates: "yes"},
	{Group: "APHRODITE / PROJECTS", Command: "pantheon aphrodite artifact <project-id> ...", Purpose: "write a candidate artifact inside the bounded project workspace", Args: "project-id --kind <kind> --title <title> --content <text>", Mutates: "yes"},
	{Group: "APHRODITE / PROJECTS", Command: "pantheon aphrodite research <project-id> ...", Purpose: "record a research request for Apollo review", Args: "project-id --question <text> [--rationale <text>] [--brief]", Mutates: "yes"},
	{Group: "APHRODITE / PROJECTS", Command: "pantheon aphrodite fame-fixture <project-id> ...", Purpose: "copy a transcript or Rumor artifact into Fame without changing the source", Args: "project-id --title <title> [--transcript <text>|--rumor-source <id>]", Mutates: "yes"},
	{Group: "ECHO / VOICES", Command: "pantheon show echo", Purpose: "inspect local diarization runs, anonymous speakers, and voice-profile state", Mutates: "no"},
	{Group: "ECHO / VOICES", Command: "pantheon echo voices", Purpose: "list human voice profiles without assigning anonymous speakers", Mutates: "no"},
	{Group: "ECHO / VOICES", Command: "pantheon echo voice add <name>", Purpose: "create a local voice profile awaiting a sample", Args: "name", Mutates: "yes"},
	{Group: "ECHO / VOICES", Command: "pantheon echo voice enroll <voice-id> <audio-path> ...", Purpose: "record a hashed local sample reference for a voice profile", Args: "voice-id audio-path [--start seconds --end seconds]", Mutates: "yes"},
	{Group: "ECHO / VOICES", Command: "pantheon echo import <audio-path> <diarization.tsv>", Purpose: "import a completed local Echo diarization run and keep speaker labels anonymous", Args: "audio-path diarization.tsv [--speakers count]", Mutates: "yes"},
	{Group: "FAME / PEOPLE", Command: "pantheon show fame", Purpose: "inspect durable people, conversations, identity evidence, observations, and corrections", Mutates: "no"},
	{Group: "FAME / PEOPLE", Command: "pantheon fame person add <name>", Purpose: "create a provisional local person record", Args: "name", Mutates: "yes"},
	{Group: "FAME / PEOPLE", Command: "pantheon fame conversation <echo-run-id>", Purpose: "create a provisional conversation boundary and preserve Echo speaker attribution links", Args: "echo-run-id", Mutates: "yes"},
	{Group: "FAME / PEOPLE", Command: "pantheon fame process <echo-run-id> <speaker> <person-id>", Purpose: "run the bounded Kernel 25 path: confirm one speaker, extract cited interaction observations, and preserve corrections", Args: "echo-run-id speaker person-id", Mutates: "yes"},
	{Group: "FAME / PEOPLE", Command: "pantheon fame extract <conversation-id>", Purpose: "run bounded local interaction extraction with exact transcript citations", Args: "conversation-id", Mutates: "yes"},
	{Group: "FAME / PEOPLE", Command: "pantheon fame correct <target-type> <kind> <statement>", Purpose: "record a human correction without silently relabeling a recording", Args: "target-type kind statement", Mutates: "yes"},
	{Group: "CONTROL", Command: "pantheon prometheus protocol --pause", Purpose: "pause autonomous activity for inspection", Mutates: "yes", Control: "local control token"},
	{Group: "CONTROL", Command: "pantheon prometheus protocol --resume", Purpose: "resume after an explicit inspection pause", Mutates: "yes", Control: "local control token"},
	{Group: "CONTROL", Command: "pantheon prometheus protocol --status", Purpose: "inspect the Prometheus pause state", Mutates: "no"},
	{Group: "INTAKE / PRESENT", Command: "pantheon intake scan", Purpose: "inventory local Intake and record deterministic occurrences", Mutates: "yes"},
	{Group: "INTAKE / PRESENT", Command: "pantheon present --new", Purpose: "normalize all newly observed occurrences", Mutates: "yes"},
	{Group: "INTAKE / PRESENT", Command: "pantheon present --occurrence <id>", Purpose: "faithfully normalize one occurrence and verify it", Args: "id", Mutates: "yes"},
	{Group: "APOLLO", Command: "pantheon order --recipient apollo ...", Purpose: "create a sealed local Apollo reasoning order", Mutates: "yes", Control: "local control token"},
	{Group: "ACTIONS", Command: "pantheon show actions", Purpose: "inspect Apollo-derived calendar, email, and clarification drafts", Mutates: "no"},
	{Group: "ACTIONS", Command: "pantheon actions accept <id>", Purpose: "accept one reviewable action into Future; no Mercury execution is started", Args: "id", Mutates: "yes", Control: "local control token"},
	{Group: "ARCHIMEDES", Command: "pantheon show archimedes", Purpose: "inspect local-only frontier referrals, exchanges, privacy incidents, and retention receipts", Mutates: "no"},
	{Group: "ARCHIMEDES", Command: "pantheon archimedes refer --reason <text> --question <text>", Purpose: "record a needs_frontier_reasoning referral without sending anything", Mutates: "yes", Control: "local control token"},
	{Group: "ARCHIMEDES", Command: "pantheon archimedes referrals|exchanges", Purpose: "inspect durable Archimedes referrals or exchange receipts", Mutates: "no"},
	{Group: "RUMOR / AUDIO", Command: "pantheon show rumor", Purpose: "inspect Rumor watcher, queue, and recent audio source state", Mutates: "no"},
	{Group: "RUMOR / AUDIO", Command: "pantheon show rumor queue", Purpose: "inspect due Rumor transcription work", Mutates: "no"},
	{Group: "RUMOR / AUDIO", Command: "pantheon show rumor source <id>", Purpose: "inspect one Rumor source, attempts, artifacts, and Intake handoff", Args: "id", Mutates: "no"},
	{Group: "RUMOR / AUDIO", Command: "pantheon rumor scan", Purpose: "scan configured audio paths and process due Rumor work", Mutates: "yes", Control: "local control token"},
	{Group: "RUMOR / AUDIO", Command: "pantheon rumor transcribe <audio-path>", Purpose: "explicitly enqueue and transcribe one local audio source", Args: "path", Mutates: "yes", Control: "local control token"},
	{Group: "RUMOR / AUDIO", Command: "pantheon rumor retry <source-id>", Purpose: "manually retry one failed or ignored source", Args: "id", Mutates: "yes", Control: "local control token"},
	{Group: "RUMOR / AUDIO", Command: "pantheon rumor ignore <source-id>", Purpose: "paper-jam an unwanted source without deleting its raw audio", Args: "id", Mutates: "yes", Control: "local control token"},
	{Group: "REPORTS / CONTINUITY", Command: "pantheon report assemble --date <YYYY-MM-DD>", Purpose: "assemble a replay-safe Vault evidence manifest for a daily continuity or event report", Args: "[--kind event|daily] [--source Vault-relative-path]", Mutates: "yes", Control: "local control token"},
	{Group: "REPORTS / CONTINUITY", Command: "pantheon report assemblies", Purpose: "inspect durable report-assembly receipts and exact source sets", Mutates: "no"},
	{Group: "WEAVE / LINKING", Command: "pantheon weave-links crawl", Purpose: "crawl user-authored Nexus chunks into a separate resumable link graph; agent chunks and Vault notes are untouched", Args: "[--batch-size n --max-links-per-chunk n]", Mutates: "yes"},
	{Group: "WEAVE / LINKING", Command: "pantheon weave-links materialize-shards", Purpose: "materialize user-only chunks as provenance-preserving Obsidian shard notes with crawler links", Args: "[--graph PATH --root Vault-relative-directory]", Mutates: "yes"},
	{Group: "MAIL / INTELLIGENCE", Command: "pantheon mail check", Purpose: "run one explicit bounded Gmail intake and Apollo-understanding round", Mutates: "yes", Control: "local control token plus Bifrost use approval"},
	{Group: "MAIL / INTELLIGENCE", Command: "pantheon show mail", Purpose: "inspect recorded mail understanding, backlog, filters, and action receipts", Mutates: "no"},
	{Group: "MAIL / INTELLIGENCE", Command: "pantheon show mail messages", Purpose: "list recorded mail messages without contacting the mailbox", Mutates: "no"},
	{Group: "ZODIAC / SCHEDULES", Command: "pantheon show zodiac", Purpose: "inspect schedules, due instances, projections, providers, and Needs You", Mutates: "no"},
	{Group: "ZODIAC / SCHEDULES", Command: "pantheon zodiac round", Purpose: "evaluate persisted schedules against Father Time", Mutates: "yes", Control: "local control token"},
	{Group: "ZODIAC / SCHEDULES", Command: "pantheon zodiac today", Purpose: "show durable due instances for the local operating day", Mutates: "no"},
	{Group: "ZODIAC / SCHEDULES", Command: "pantheon zodiac upcoming", Purpose: "show bounded upcoming due instances", Mutates: "no"},
	{Group: "ZODIAC / SCHEDULES", Command: "pantheon zodiac needs-you", Purpose: "show ambiguous, conflicted, blocked, and authority-bound schedule exceptions", Mutates: "no"},
	{Group: "ZODIAC / SCHEDULES", Command: "pantheon zodiac schedule ...", Purpose: "declare one explicit schedule or obligation", Mutates: "yes", Control: "local control token"},
	{Group: "ZEUS / SANITATION", Command: "pantheon show zeus", Purpose: "inspect configured machine-owned roots and the latest sanitation report", Mutates: "no"},
	{Group: "ZEUS / SANITATION", Command: "pantheon zeus inventory", Purpose: "record a deterministic inventory of configured machine-owned roots", Mutates: "yes"},
	{Group: "ZEUS / SANITATION", Command: "pantheon zeus dry-run", Purpose: "evaluate exact cache candidates and report recoverable quarantine actions without deleting", Mutates: "yes"},
	{Group: "SESHAT / PLUGINS", Command: "pantheon help --codes", Purpose: "list current subject codes, fields, capabilities, and status", Mutates: "no"},
	{Group: "SESHAT / PLUGINS", Command: "pantheon plugin help", Purpose: "show the local package-extension workflow", Mutates: "no"},
	{Group: "SESHAT / PLUGINS", Command: "pantheon plugin validate <path>", Purpose: "validate a manifest without registering it", Args: "path", Mutates: "no"},
	{Group: "SESHAT / PLUGINS", Command: "pantheon plugin register <path>", Purpose: "register a validated package manifest", Args: "path", Mutates: "yes", Control: "local control token"},
	{Group: "BIFROST / CAPABILITIES", Command: "pantheon bifrost status", Purpose: "inspect governed external capabilities and mail crossing", Mutates: "no"},
	{Group: "BIFROST / CAPABILITIES", Command: "pantheon bifrost list", Purpose: "list registered external capabilities and approval state", Mutates: "no"},
	{Group: "BIFROST / CAPABILITIES", Command: "pantheon bifrost show <capability>", Purpose: "inspect one capability's source, access, and approvals", Args: "capability", Mutates: "no"},
	{Group: "BIFROST / CAPABILITIES", Command: "pantheon bifrost approvals", Purpose: "inspect separate download, installation, registration, and use receipts", Mutates: "no"},
	{Group: "BIFROST / CAPABILITIES", Command: "pantheon bifrost enable <capability>", Purpose: "enable an already installed and fully approved capability", Args: "capability", Mutates: "yes", Control: "local control token"},
	{Group: "BIFROST / CAPABILITIES", Command: "pantheon bifrost disable <capability>", Purpose: "fail closed for a capability without removing its record", Args: "capability", Mutates: "yes", Control: "local control token"},
	{Group: "BIFROST / MAIL", Command: "pantheon bifrost mail status", Purpose: "inspect bounded mail crossing state and migration boundary", Mutates: "no"},
	{Group: "BIFROST / MAIL", Command: "pantheon bifrost mail intake", Purpose: "retrieve bounded new mail and write normalized Intake artifacts", Mutates: "yes", Control: "local control token plus Bifrost use approval"},
	{Group: "JANUS", Command: "pantheon janus round", Purpose: "apply deterministic temporal transitions", Mutates: "yes", Control: "local control token"},
	{Group: "FUTURE", Command: "pantheon future round", Purpose: "resolve proposals against existing authority and Janus state", Mutates: "yes", Control: "local control token"},
	{Group: "MERCURY", Command: "pantheon mercury round", Purpose: "issue Go and execute sealed exact orders", Mutates: "yes", Control: "local control token"},
	{Group: "INTEGRATION", Command: "pantheon organism run ...", Purpose: "connect Present, Apollo, Seshat, Janus, Future, Mercury, and evidence", Mutates: "yes", Control: "local control token"},
	{Group: "AUTHORITY / GRANTS", Command: "pantheon grants", Purpose: "inspect grants and their current state", Mutates: "no"},
	{Group: "AUTHORITY / GRANTS", Command: "pantheon authority check <capability> <scope>", Purpose: "check effective authority without creating it", Args: "capability scope", Mutates: "no"},
	{Group: "HELP", Command: "pantheon help", Purpose: "show concise operator guidance", Mutates: "no"},
	{Group: "HELP", Command: "pantheon help --commands", Purpose: "list discoverable commands and safety properties", Mutates: "no"},
	{Group: "HELP", Command: "pantheon help --codes", Purpose: "list Seshat subject vocabulary", Mutates: "no"},
}

func runHelpCommand(args []string) error {
	if len(args) == 0 {
		fmt.Print(commandHelpText())
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: pantheon help [--commands|--codes]")
	}
	switch args[0] {
	case "--commands", "commands":
		fmt.Print(commandDiscoveryText())
		return nil
	case "--codes", "codes":
		return printLocalCodeHelp()
	default:
		return fmt.Errorf("usage: pantheon help [--commands|--codes]")
	}
}

func commandHelpText() string {
	return "Pantheon is a local-first personal AI control system. Start here:\n\n" +
		"  pantheon status                         check whether the daemon is alive\n" +
		"  pantheon help --commands                discover commands and safety boundaries\n" +
		"  pantheon help --codes                   discover Seshat subject vocabulary\n" +
		"  pantheon prometheus protocol --pause    stop activity for inspection\n" +
		"  pantheon show future                    inspect authority administration\n" +
		"  pantheon show mercury                   inspect sealed-order execution\n\n" +
		"All reasoning is local by default. Registration, authority, and execution are separate boundaries.\n"
}

func commandDiscoveryText() string {
	entries := append([]commandHelpEntry{}, pantheonCommands...)
	groups := map[string][]commandHelpEntry{}
	var order []string
	for _, entry := range entries {
		if _, exists := groups[entry.Group]; !exists {
			order = append(order, entry.Group)
		}
		groups[entry.Group] = append(groups[entry.Group], entry)
	}
	var b strings.Builder
	b.WriteString("Pantheon commands (local metadata; no model required)\n")
	for _, group := range order {
		fmt.Fprintf(&b, "\n%s\n", group)
		for _, entry := range groups[group] {
			fmt.Fprintf(&b, "  %-46s %s\n", entry.Command, entry.Purpose)
			if entry.Args != "" || entry.Control != "" {
				fmt.Fprintf(&b, "    arguments: %-28s changes state: %-3s", entry.Args, entry.Mutates)
				if entry.Control != "" {
					fmt.Fprintf(&b, "  boundary: %s", entry.Control)
				}
				b.WriteByte('\n')
			} else {
				fmt.Fprintf(&b, "    changes state: %s\n", entry.Mutates)
			}
		}
	}
	return b.String()
}

func printLocalCodeHelp() error {
	opts, err := defaultDaemonOptions()
	if err != nil {
		return err
	}
	registryDir := filepath.Join(filepath.Dir(opts.ConfigPath), "plugins", "registered")
	if _, err := os.Stat(registryDir); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No registered Seshat packages found at %s.\n", registryDir)
			return nil
		}
		return err
	}
	registry, err := newSeshatRegistry(registryDir)
	if err != nil {
		return fmt.Errorf("load Seshat code index: %w", err)
	}
	fmt.Print(seshatCodeHelp(registry.packagesInspection()))
	return nil
}

func seshatCodeHelp(packages []seshatPackage) string {
	packages = append([]seshatPackage{}, packages...)
	sort.Slice(packages, func(i, j int) bool { return packages[i].SubjectCode < packages[j].SubjectCode })
	var b strings.Builder
	b.WriteString("Seshat subject codes (package form, not authority)\n")
	for _, pkg := range packages {
		if !packageEnabled(pkg) {
			continue
		}
		status := "disabled"
		if pkg.AdapterAvailable {
			status = "ready"
		} else {
			status = "adapter unavailable"
		}
		fmt.Fprintf(&b, "\n%s\n  %s\n  Requires: %s\n", pkg.SubjectCode, pkg.Description, strings.Join(pkg.RequiredFields, ", "))
		if len(pkg.OptionalFields) > 0 {
			fmt.Fprintf(&b, "  Optional: %s\n", strings.Join(pkg.OptionalFields, ", "))
		}
		fmt.Fprintf(&b, "  Capability: %s\n  Status: %s\n", pkg.Capability, status)
	}
	return b.String()
}
