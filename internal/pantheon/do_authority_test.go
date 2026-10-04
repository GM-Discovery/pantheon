package pantheon

import "testing"

// `pantheon do` is advertised in the README as routing a request and then
// autonomously executing low-risk capabilities "while refusing automatic
// execution of capabilities that have not yet been authorized." These
// tests pin that sentence as behavior.
//
// Before this, POST /v1/input enforced `capability != "reason"` -> refuse,
// a single hardcoded allowlist entry rather than a rule over declared risk
// and real grants, and there was no `do` verb at all.
func doTestDaemon(t *testing.T) *daemon {
	t.Helper()
	d := testKernel9Daemon(t, testKernel9Config(t, t.TempDir()))
	if err := initializeAuthoritySchema(d.db); err != nil {
		t.Fatal(err)
	}
	d.capabilities = capabilityConfig{Capabilities: []capability{
		{Name: "reason", Type: "model", Risk: "low"},
		{Name: "notify", Type: "deterministic", Risk: "low"},
		{Name: "hermes", Type: "delegated", Risk: "medium"},
		{Name: "email.read", Type: "delegated", Risk: "low"},
		{Name: "self.change", Type: "deterministic", Risk: "high"},
	}}
	return d
}

func TestAutonomousAuthorityTiers(t *testing.T) {
	d := doTestDaemon(t)

	tests := []struct {
		capability string
		want       bool
		why        string
	}{
		{"reason", true, "low-risk model reasoning is the baseline autonomous capability"},
		{"notify", true, "low-risk deterministic work has a Pantheon-defined effect"},

		// Delegated capabilities hand work to an agent that picks its own
		// actions, so the declared risk tier does not bound the blast
		// radius. email.read is low risk and must STILL be refused --
		// this is the case a naive `risk == "low"` check gets wrong.
		{"email.read", false, "delegated capability, low risk, still needs a grant"},
		{"hermes", false, "delegated and medium risk"},
		{"self.change", false, "high risk"},
	}

	for _, tt := range tests {
		t.Run(tt.capability, func(t *testing.T) {
			cap, found := d.capabilities.capabilityByName(tt.capability)
			if !found {
				t.Fatalf("capability %q not declared in fixture", tt.capability)
			}
			allowed, basis, grantID := d.autonomousAuthority(cap)
			if allowed != tt.want {
				t.Fatalf("autonomousAuthority(%s) = %v (basis %q), want %v -- %s",
					tt.capability, allowed, basis, tt.want, tt.why)
			}
			if basis == "" {
				t.Fatal("every decision must carry a basis, so the audit log explains itself")
			}
			if grantID != "" {
				t.Fatalf("no grant exists in this fixture, got grant id %q", grantID)
			}
		})
	}
}

// A human-issued grant must be able to authorize something the risk tier
// refuses -- otherwise the grant system is decorative and the tier rule is
// the real policy.
func TestAutonomousAuthorityHonorsExplicitGrant(t *testing.T) {
	d := doTestDaemon(t)
	cap, _ := d.capabilities.capabilityByName("hermes")

	if allowed, _, _ := d.autonomousAuthority(cap); allowed {
		t.Fatal("hermes must be refused before any grant exists")
	}

	if _, err := d.db.Exec(`INSERT INTO capability_grants
		(id, capability, scope, intent_summary, human_approved, status, created_at, holder, granted_by)
		VALUES ('grant-do-test', 'hermes', '', 'test grant', 1, 'active', 'now', 'person', 'person')`); err != nil {
		t.Fatal(err)
	}

	allowed, basis, grantID := d.autonomousAuthority(cap)
	if !allowed {
		t.Fatalf("an active human-approved grant must authorize hermes, basis=%q", basis)
	}
	if grantID != "grant-do-test" {
		t.Fatalf("grant id = %q, want grant-do-test -- the caller needs it to consume single-use grants", grantID)
	}

	// Revocation must take effect without restarting anything.
	if _, err := d.db.Exec(`UPDATE capability_grants SET status = 'revoked' WHERE id = 'grant-do-test'`); err != nil {
		t.Fatal(err)
	}
	if allowed, _, _ := d.autonomousAuthority(cap); allowed {
		t.Fatal("a revoked grant must stop authorizing immediately")
	}
}

// A capability may be declared and authorized and still have nothing behind
// it. That must read as a refusal, never a silent success -- the failure
// mode this guards against is Pantheon reporting that it did something it
// has no executor for.
func TestExecuteCapabilityRefusesUndefinedExecutor(t *testing.T) {
	d := doTestDaemon(t)
	cap, _ := d.capabilities.capabilityByName("email.read")

	result, err := d.executeCapability(cap, "anything")
	if err == nil {
		t.Fatalf("expected a refusal for a capability with no executor, got result %q", result)
	}
	if result != "" {
		t.Fatalf("a refused execution must return no result, got %q", result)
	}
}

// The registry is the authority on what exists: a model routing to a
// capability Pantheon does not declare must not be guessed at.
func TestCapabilityByNameRejectsUndeclared(t *testing.T) {
	d := doTestDaemon(t)
	if _, found := d.capabilities.capabilityByName("shell.admin"); found {
		t.Fatal("shell.admin is not declared in the fixture and must not resolve")
	}
	if cap, found := d.capabilities.capabilityByName("  NOTIFY  "); !found || cap.Name != "notify" {
		t.Fatal("lookup must tolerate whitespace and case from a model's JSON")
	}
}
