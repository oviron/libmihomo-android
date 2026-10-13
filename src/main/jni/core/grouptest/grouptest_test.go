package grouptest

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func tree(edges map[string][]string) func(string) []string {
	return func(name string) []string { return edges[name] }
}

func TestOrderMeasuresChildrenBeforeTheirParent(t *testing.T) {
	children := tree(map[string][]string{
		"VPN":   {"qde", "Blanc"},
		"qde":   nil,
		"Blanc": nil,
	})
	got := Order([]string{"VPN"}, children)
	want := []string{"qde", "Blanc", "VPN"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Order = %v, want %v", got, want)
	}
}

func TestOrderVisitsASharedGroupOnce(t *testing.T) {
	children := tree(map[string][]string{
		"VPN": {"qde"},
		"qde": nil,
	})
	got := Order([]string{"VPN", "qde"}, children)
	want := []string{"qde", "VPN"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Order = %v, want %v", got, want)
	}
}

func TestOrderSurvivesACycle(t *testing.T) {
	children := tree(map[string][]string{"A": {"B"}, "B": {"A"}})
	got := Order([]string{"A"}, children)
	want := []string{"B", "A"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Order = %v, want %v", got, want)
	}
}

func TestFastestPicksTheLowestDelay(t *testing.T) {
	got, ok := Fastest(
		[]string{"Blanc", "qde", "qde-ru"},
		map[string]int{"Blanc": 103, "qde": 55, "qde-ru": 113},
	)
	if !ok || got != "qde" {
		t.Fatalf("Fastest = %q, %v; want qde, true", got, ok)
	}
}

func TestFastestSkipsFailures(t *testing.T) {
	got, ok := Fastest(
		[]string{"dead", "slow"},
		map[string]int{"dead": -1, "slow": 300},
	)
	if !ok || got != "slow" {
		t.Fatalf("Fastest = %q, %v; want slow, true", got, ok)
	}
}

func TestFastestBreaksATieByMemberOrder(t *testing.T) {
	got, _ := Fastest([]string{"b", "a"}, map[string]int{"a": 50, "b": 50})
	if got != "b" {
		t.Fatalf("Fastest = %q, want b", got)
	}
}

func TestFastestFindsNothingWhenAllFailed(t *testing.T) {
	if got, ok := Fastest([]string{"x"}, map[string]int{"x": -1}); ok {
		t.Fatalf("Fastest = %q, true; want false", got)
	}
}

type fakeGroup struct {
	calls []string
	now   string
}

func (f *fakeGroup) ForceSet(name string) {
	f.calls = append(f.calls, "set:"+name)
	if name != "" {
		f.now = name
	}
}

func (f *fakeGroup) Now() string {
	f.calls = append(f.calls, "now")
	return f.now
}

// Seat must read the group between the two ForceSet calls: mihomo only adopts
// a pick when the group is read, and the unpin alone would leave the old pick.
func TestSeatAdoptsThePickThenLetsGo(t *testing.T) {
	g := &fakeGroup{now: "Blanc"}
	Seat(g, "qde")
	want := []string{"set:qde", "now", "set:"}
	if !reflect.DeepEqual(g.calls, want) {
		t.Fatalf("calls = %v, want %v", g.calls, want)
	}
}

func groups(g map[string][2]string) func(string) (string, string, bool) {
	return func(name string) (string, string, bool) {
		v, ok := g[name]
		return v[0], v[1], ok
	}
}

// The app shows a group's tile with the delay of the node it resolves to, so a
// parent's measurement has to land on that node and the URL of its own group.
func TestLeafFollowsNestedPicksToTheNode(t *testing.T) {
	step := groups(map[string][2]string{
		"VPN": {"qde", "https://a"},
		"qde": {"qd-grpc", "https://b"},
	})
	leaf, url, ok := Leaf("qde", step)
	if !ok || leaf != "qd-grpc" || url != "https://b" {
		t.Fatalf("Leaf = %q, %q, %v; want qd-grpc, https://b, true", leaf, url, ok)
	}
}

func TestLeafOfANodeIsNothing(t *testing.T) {
	if _, _, ok := Leaf("qd-grpc", groups(nil)); ok {
		t.Fatal("Leaf of a plain node reported ok")
	}
}

func TestLeafStopsOnACycle(t *testing.T) {
	step := groups(map[string][2]string{"A": {"B", "u"}, "B": {"A", "u"}})
	if _, _, ok := Leaf("A", step); ok {
		t.Fatal("Leaf of a cycle reported ok")
	}
}

func TestLeafOfALoadBalanceIsNothing(t *testing.T) {
	step := groups(map[string][2]string{"lb": {"", "u"}})
	if _, _, ok := Leaf("lb", step); ok {
		t.Fatal("Leaf of a group with no pick reported ok")
	}
}

func runWithin(t *testing.T, order []string, run func(string, func(string))) {
	t.Helper()
	finished := make(chan struct{})
	go func() {
		Run(order, run)
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not finish")
	}
}

func TestRunParentWaitsForItsChild(t *testing.T) {
	var mu sync.Mutex
	var events []string
	record := func(e string) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	}
	runWithin(t, []string{"child", "parent"}, func(name string, await func(string)) {
		if name == "child" {
			time.Sleep(50 * time.Millisecond)
			record("child done")
			return
		}
		await("child")
		record("parent tests child")
	})
	if len(events) != 2 || events[0] != "child done" {
		t.Fatalf("events = %v, want the child first", events)
	}
}

// Unrelated groups must not queue behind each other: each one here only
// finishes once the other has started.
func TestRunSiblingsRunTogether(t *testing.T) {
	started := map[string]chan struct{}{"a": make(chan struct{}), "b": make(chan struct{})}
	other := map[string]string{"a": "b", "b": "a"}
	runWithin(t, []string{"a", "b"}, func(name string, _ func(string)) {
		close(started[name])
		<-started[other[name]]
	})
}

func TestRunCycleDoesNotDeadlock(t *testing.T) {
	runWithin(t, []string{"a", "b"}, func(name string, await func(string)) {
		if name == "a" {
			await("b")
		} else {
			await("a")
		}
	})
}
