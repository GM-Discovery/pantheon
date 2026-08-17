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
	{Group: "STATUS / INSPECTION", Command: "pantheon show <area>", Purpose: "inspect Intake, Present, processors, orders, Apollo, Future, Janus, Mercury, or packages", Args: "area", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show janus", Purpose: "inspect temporal subjects and transition readiness", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show order <id>", Purpose: "inspect one durable order and its evidence", Args: "id", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show apollo report <id>", Purpose: "read one durable cited Apollo report", Args: "id", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show future proposal <id>", Purpose: "inspect Future authority and execution state", Args: "id", Mutates: "no"},
	{Group: "STATUS / INSPECTION", Command: "pantheon show organism", Purpose: "inspect whole-animal integration traces", Mutates: "no"},
	{Group: "CONTROL", Command: "pantheon prometheus protocol --pause", Purpose: "pause autonomous activity for inspection", Mutates: "yes", Control: "local control token"},
	{Group: "CONTROL", Command: "pantheon prometheus protocol --resume", Purpose: "resume after an explicit inspection pause", Mutates: "yes", Control: "local control token"},
	{Group: "CONTROL", Command: "pantheon prometheus protocol --status", Purpose: "inspect the Prometheus pause state", Mutates: "no"},
	{Group: "INTAKE / PRESENT", Command: "pantheon intake scan", Purpose: "inventory local Intake and record deterministic occurrences", Mutates: "yes"},
	{Group: "INTAKE / PRESENT", Command: "pantheon present --new", Purpose: "normalize all newly observed occurrences", Mutates: "yes"},
	{Group: "INTAKE / PRESENT", Command: "pantheon present --occurrence <id>", Purpose: "faithfully normalize one occurrence and verify it", Args: "id", Mutates: "yes"},
	{Group: "APOLLO", Command: "pantheon order --recipient apollo ...", Purpose: "create a sealed local Apollo reasoning order", Mutates: "yes", Control: "local control token"},
	{Group: "SESHAT / PLUGINS", Command: "pantheon help --codes", Purpose: "list current subject codes, fields, capabilities, and status", Mutates: "no"},
	{Group: "SESHAT / PLUGINS", Command: "pantheon plugin help", Purpose: "show the local package-extension workflow", Mutates: "no"},
	{Group: "SESHAT / PLUGINS", Command: "pantheon plugin validate <path>", Purpose: "validate a manifest without registering it", Args: "path", Mutates: "no"},
	{Group: "SESHAT / PLUGINS", Command: "pantheon plugin register <path>", Purpose: "register a validated package manifest", Args: "path", Mutates: "yes", Control: "local control token"},
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
