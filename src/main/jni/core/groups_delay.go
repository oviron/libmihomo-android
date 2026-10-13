//go:build android && cgo

package main

import (
	"encoding/json"
	"sync"

	"github.com/metacubex/mihomo/adapter/outboundgroup"
	"github.com/metacubex/mihomo/common/utils"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/tunnel"
	"github.com/oviron/libmihomo-android/grouptest"
)

// GroupDelay is one measured member, keyed the way the app keys its delays.
type GroupDelay struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Value int    `json:"value"`
}

type groupState struct {
	TestURL string `json:"testUrl"`
	Fixed   string `json:"fixed"`
}

// handleTestGroups tests the named groups and their nested groups at once, a
// parent testing a nested group only after it re-picked, and seats each
// unpinned url-test group on its fastest member, skipping the tolerance.
func handleTestGroups(names []string, defaultURL string, timeoutMs int) string {
	// Groups live in tunnel.Proxies(); a provider node of the same name must not hide one.
	proxies := tunnel.Proxies()
	groupOf := func(name string) (outboundgroup.ProxyGroup, bool) {
		p, ok := proxies[name]
		if !ok {
			return nil, false
		}
		g, ok := p.Adapter().(outboundgroup.ProxyGroup)
		return g, ok
	}
	children := func(name string) []string {
		g, ok := groupOf(name)
		if !ok {
			return nil
		}
		var out []string
		for _, m := range g.Proxies() {
			if _, ok := groupOf(m.Name()); ok {
				out = append(out, m.Name())
			}
		}
		return out
	}

	urlOf := func(state groupState) string {
		if state.TestURL != "" {
			return state.TestURL
		}
		return defaultURL
	}
	step := func(name string) (string, string, bool) {
		g, ok := groupOf(name)
		if !ok {
			return "", "", false
		}
		return g.Now(), urlOf(stateOf(g)), true
	}

	var mu sync.Mutex
	results := []GroupDelay{}
	grouptest.Run(grouptest.Order(names, children), func(name string, await func(string)) {
		g, ok := groupOf(name)
		if !ok {
			return
		}
		state := stateOf(g)
		url := urlOf(state)
		members := g.Proxies()
		expected, _ := utils.NewUnsignedRanges[uint16](expectedStatusOf(name))
		delays := testMembers(members, url, expected, timeoutMs, await)
		// mihomo picks by its shared alive flag, which a reply outside
		// expected-status clears; showing and seating by it keeps us in step.
		for _, m := range members {
			if !m.AliveForTestUrl(url) {
				delays[m.Name()] = -1
			}
		}
		memberNames := make([]string, len(members))
		found := make([]GroupDelay, 0, 2*len(members))
		for i, m := range members {
			memberNames[i] = m.Name()
			found = append(found, GroupDelay{Name: m.Name(), URL: url, Value: delays[m.Name()]})
		}
		if ut, ok := g.(*outboundgroup.URLTest); ok {
			// The pin is read again here: the user may have pinned during the test.
			pickMu.Lock()
			if best, ok := grouptest.Fastest(memberNames, delays); ok && stateOf(g).Fixed == "" {
				grouptest.Seat(ut, best)
			}
			pickMu.Unlock()
		}
		// A nested group's tile shows the node it resolves to: give that node
		// the delay this group picked by, so the screen and the pick agree.
		for _, m := range members {
			if leaf, leafURL, ok := grouptest.Leaf(m.Name(), step); ok {
				found = append(found, GroupDelay{Name: leaf, URL: leafURL, Value: delays[m.Name()]})
			}
		}
		mu.Lock()
		results = append(results, found...)
		mu.Unlock()
		// Each group reports as it finishes; a parent waits on its slowest member.
		sendMessage(Message{Type: GroupDelayMessage, Data: found})
	})
	data, err := json.Marshal(results)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func stateOf(g outboundgroup.ProxyGroup) groupState {
	var state groupState
	if data, err := g.MarshalJSON(); err == nil {
		_ = json.Unmarshal(data, &state)
	}
	return state
}

// testMembers tests a nested group only once await says it has re-picked; the
// plain members start at once.
func testMembers(members []C.Proxy, url string, expected utils.IntRanges[uint16], timeoutMs int, await func(string)) map[string]int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	out := make(map[string]int, len(members))
	for _, m := range members {
		wg.Add(1)
		go func(m C.Proxy) {
			defer wg.Done()
			await(m.Name())
			delay := testProxy(m, url, expected, timeoutMs)
			mu.Lock()
			out[m.Name()] = delay
			mu.Unlock()
		}(m)
	}
	wg.Wait()
	return out
}
