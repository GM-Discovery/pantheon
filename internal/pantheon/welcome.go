package pantheon

import (
	"fmt"
	"os"
)

func printWelcome() {
	cyan := "\033[1;96m"
	red := "\033[1;91m"
	green := "\033[1;92m"
	white := "\033[1;97m"
	dim := "\033[2m"
	reset := "\033[0m"

	if os.Getenv("NO_COLOR") != "" {
		cyan, red, green, white, dim, reset = "", "", "", "", "", ""
	}

	fmt.Printf(`
╭──────────────────────────────────────────────────────────────────────────────╮
│                                                                              │
│                          %sP A N T H E O N%s                                 │
│                   %sLOCAL INTELLIGENCE · BOUND IN SERVICE%s                  │
│                                                                              │
│                              ╭────────────╮                                  │
│                              │  %s■ STOP%s    │                                  │
│                              ╰────────────╯                                  │
│                                                                              │
│                         %sIN CASE OF EMERGENCY%s                              │
│                                                                              │
│  %sbash%s                                                                        │
│  %spantheon prometheus protocol --pause%s                                    │
│                                                                              │
│        Pauses Pantheon and autonomous activity for inspection.               │
│                   You do not need to run it now.                             │
│                                                                              │
╰──────────────────────────────────────────────────────────────────────────────╯

Available commands:
  pantheon status                         inspect daemon state
  pantheon intake scan                    inventory Intake locally
  pantheon present "request"              prepare and verify an Intake source
  pantheon show intake                    inspect current Intake ledger
  pantheon show present                   inspect Present missives
  pantheon show processors                inspect local processor readiness
  pantheon order "request"                record a trusted durable order
  pantheon mercury round                  run one deterministic Mercury round
  pantheon show orders                    inspect durable orders
  pantheon show mercury                   inspect Mercury readiness and backlog
  pantheon show apollo                    inspect Apollo readiness and reports
  pantheon show father-time               inspect Father Time chronology
  pantheon prometheus protocol --pause    pause Pantheon for inspection

`,
		cyan, reset,
		dim, reset,
		red, reset,
		white, reset,
		dim, reset,
		green, reset,
	)
}
