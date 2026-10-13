package selection

import (
	"errors"
	"testing"
)

type fakeSelector struct {
	set      []string
	forceSet []string
	setErr   error
}

func (f *fakeSelector) Set(name string) error {
	f.set = append(f.set, name)
	return f.setErr
}

func (f *fakeSelector) ForceSet(name string) {
	f.forceSet = append(f.forceSet, name)
}

func TestApplyPinsByName(t *testing.T) {
	sel := &fakeSelector{}
	if err := Apply(sel, "node-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sel.set) != 1 || sel.set[0] != "node-1" {
		t.Fatalf("Set calls = %v, want [node-1]", sel.set)
	}
	if len(sel.forceSet) != 0 {
		t.Fatalf("ForceSet calls = %v, want none", sel.forceSet)
	}
}

// Regression: an unpin routed through Set() is refused with "proxy not exist",
// which left the pin in place and the group stuck on the pinned node.
func TestApplyUnpinsOnEmptyName(t *testing.T) {
	sel := &fakeSelector{setErr: errors.New("proxy not exist")}
	if err := Apply(sel, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sel.set) != 0 {
		t.Fatalf("Set calls = %v, want none", sel.set)
	}
	if len(sel.forceSet) != 1 || sel.forceSet[0] != "" {
		t.Fatalf("ForceSet calls = %v, want [\"\"]", sel.forceSet)
	}
}

func TestApplyReportsRefusal(t *testing.T) {
	want := errors.New("proxy not exist")
	sel := &fakeSelector{setErr: want}
	if err := Apply(sel, "absent"); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
