package core

// NSHConfig represents the complete SSH config file
type NSHConfig struct {
	Blocks        []NSHBlock
	GroupOrder    []string
	PinnedAliases []string
}

// Hosts returns all hosts (excluding Match/Include/Comment/Blank blocks)
func (c *NSHConfig) Hosts() []NSHHost {
	var hosts []NSHHost
	for _, b := range c.Blocks {
		if b.Type == BlockHost && b.Host != nil {
			hosts = append(hosts, *b.Host)
		}
	}
	return hosts
}

// Groups returns all unique group names in appearance order
func (c *NSHConfig) Groups() []string {
	seen := map[string]bool{}
	var result []string
	for _, h := range c.Hosts() {
		if !seen[h.Group] {
			seen[h.Group] = true
			result = append(result, h.Group)
		}
	}
	return result
}

// SortedGroups returns groups sorted by GroupOrder, then by appearance
func (c *NSHConfig) SortedGroups() []string {
	allGroups := c.Groups()
	if len(c.GroupOrder) == 0 {
		return allGroups
	}

	seen := map[string]bool{}
	var result []string

	allSet := map[string]bool{}
	for _, g := range allGroups {
		allSet[g] = true
	}

	for _, g := range c.GroupOrder {
		if allSet[g] && !seen[g] {
			seen[g] = true
			result = append(result, g)
		}
	}
	for _, g := range allGroups {
		if !seen[g] {
			seen[g] = true
			result = append(result, g)
		}
	}
	return result
}

// HostsInGroup returns hosts in the specified group, sorted by order
func (c *NSHConfig) HostsInGroup(group string) []NSHHost {
	var hosts []NSHHost
	for _, h := range c.Hosts() {
		if h.Group == group {
			hosts = append(hosts, h)
		}
	}
	// Stable sort by order
	for i := 1; i < len(hosts); i++ {
		for j := i; j > 0 && hosts[j].Order < hosts[j-1].Order; j-- {
			hosts[j], hosts[j-1] = hosts[j-1], hosts[j]
		}
	}
	return hosts
}

// IsPinned returns whether the given alias is pinned
func (c *NSHConfig) IsPinned(alias string) bool {
	for _, a := range c.PinnedAliases {
		if a == alias {
			return true
		}
	}
	return false
}

// PinnedHosts returns pinned hosts in pinned order
func (c *NSHConfig) PinnedHosts() []NSHHost {
	var result []NSHHost
	for _, alias := range c.PinnedAliases {
		if h := c.HostByAlias(alias); h != nil {
			result = append(result, *h)
		}
	}
	return result
}

// HostByAlias finds a host by its alias
func (c *NSHConfig) HostByAlias(alias string) *NSHHost {
	for _, b := range c.Blocks {
		if b.Type == BlockHost && b.Host != nil && b.Host.Alias == alias {
			h := *b.Host
			return &h
		}
	}
	return nil
}
