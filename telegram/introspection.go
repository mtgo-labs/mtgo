package telegram

// LoadSnapshot is a read-only runtime snapshot of client load for operator
// observability (FR-020). All fields are populated from atomic reads with zero
// hot-path allocation. Safe for concurrent use.
type LoadSnapshot struct {
	OutboundHighDepth int
	OutboundLowDepth  int
	InFlightRPCs      int64
	ContainersSent    int64
	MessagesPacked    int64
	ThrottleLevel     ThrottleLevel
	OverloadActive    bool
	Connection        ConnectionSnapshot
}

// LoadSnapshot returns an aggregated view of the client's internal load state.
// When production-hardening features are disabled (all config zero), most fields
// will be zero — this is expected (backward compat, Principle IV).
func (c *Client) LoadSnapshot() LoadSnapshot {
	snap := LoadSnapshot{}

	c.mu.RLock()
	sess := c.session
	oc := c.overloadController
	c.mu.RUnlock()

	// Outbound batcher metrics (from session).
	if sess != nil {
		os := sess.BatcherSnapshot()
		snap.OutboundHighDepth = os.HighDepth
		snap.OutboundLowDepth = os.LowDepth
		snap.ContainersSent = os.ContainersSent
		snap.MessagesPacked = os.MessagesPacked
	}

	// Overload controller.
	if oc != nil {
		os := oc.Snapshot()
		snap.InFlightRPCs = os.InFlight
		snap.ThrottleLevel = os.ThrottleLevel
		snap.OverloadActive = os.OverloadActive
	}
	snap.Connection = c.ConnectionSnapshot()

	return snap
}

// ConnectionSnapshot returns transport and connection lifecycle diagnostics.
func (c *Client) ConnectionSnapshot() ConnectionSnapshot {
	return c.connMetrics.Snapshot()
}
