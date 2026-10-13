// Package grouptest measures proxy groups the way mihomo's own health check
// does, so the delays the app shows are the ones url-test groups pick by.
package grouptest

import "sync"

// Order lists the groups under roots children-first, each once, so a parent is
// measured through members that have already re-picked.
func Order(roots []string, children func(string) []string) []string {
	seen := map[string]bool{}
	var out []string
	var visit func(string)
	visit = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		for _, child := range children(name) {
			visit(child)
		}
		out = append(out, name)
	}
	for _, root := range roots {
		visit(root)
	}
	return out
}

// Run runs every group of order at once. await(child) blocks until that
// child's run is done, so a parent tests a nested group only after it re-picked;
// a child not earlier in order (a cycle) is not waited for.
func Run(order []string, run func(name string, await func(child string))) {
	done := make(map[string]chan struct{}, len(order))
	index := make(map[string]int, len(order))
	for i, name := range order {
		done[name] = make(chan struct{})
		index[name] = i
	}
	var wg sync.WaitGroup
	for i, name := range order {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(done[name])
			run(name, func(child string) {
				if j, ok := index[child]; ok && j < i {
					<-done[child]
				}
			})
		}()
	}
	wg.Wait()
}

// Fastest returns the member with the lowest delay; a delay <= 0 is a failure
// and a tie goes to the earlier member.
func Fastest(members []string, delays map[string]int) (string, bool) {
	best, bestDelay := "", 0
	for _, name := range members {
		d := delays[name]
		if d > 0 && (best == "" || d < bestDelay) {
			best, bestDelay = name, d
		}
	}
	return best, best != ""
}

// Seater is the part of mihomo's url-test group Seat needs.
type Seater interface {
	ForceSet(string)
	Now() string
}

// Seat makes name the group's pick without pinning it. A url-test group keeps
// its last pick until another beats it by more than the tolerance, so pinning,
// reading and unpinning leaves name in place with the group picking for itself.
func Seat(g Seater, name string) {
	g.ForceSet(name)
	g.Now()
	g.ForceSet("")
}

// Leaf follows a group's current pick through nested groups down to the node
// that carries the traffic, with the test URL of the innermost group. step
// reports a group's pick and URL, and false for a plain node; a group with no
// single pick (load-balance) has no leaf.
func Leaf(name string, step func(string) (now, url string, isGroup bool)) (string, string, bool) {
	seen := map[string]bool{}
	cur, url := name, ""
	for {
		now, groupURL, isGroup := step(cur)
		if !isGroup {
			return cur, url, cur != name
		}
		if now == "" || seen[cur] {
			return "", "", false
		}
		seen[cur] = true
		cur, url = now, groupURL
	}
}
