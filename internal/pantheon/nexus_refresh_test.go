package pantheon

import "testing"

func TestDiffFingerprintStatesDetectsNewChangedRemoved(t *testing.T) {
	old := weaveFingerprintState{Files: map[string]string{"same": "1", "changed": "1", "removed": "1"}}
	now := weaveFingerprintState{Files: map[string]string{"same": "1", "changed": "2", "new": "1"}}
	d := diffFingerprintStates(old, now)
	if len(d.New) != 1 || len(d.Changed) != 1 || len(d.Removed) != 1 {
		t.Fatalf("%#v", d)
	}
}
