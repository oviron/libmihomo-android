// Package selection decides how a proxy-group selection reaches the core.
package selection

// Selector is the part of mihomo's outboundgroup.SelectAble this package needs.
type Selector interface {
	Set(string) error
	ForceSet(string)
}

// Apply pins a group to one of its members, or clears the pin when name is
// empty so a url-test/fallback group returns to picking for itself.
//
// Set() resolves the name among the members and rejects "", so an unpin has to
// go through ForceSet — the same split mihomo uses in its own REST handlers.
func Apply(sel Selector, name string) error {
	if name == "" {
		sel.ForceSet("")
		return nil
	}
	return sel.Set(name)
}
