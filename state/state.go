package state

import (
	"sync"
	"time"

	"mihomo-tray/mihomo"
	"mihomo-tray/registry"
)

// AppState holds the current detected state of the system and mihomo kernel.
type AppState struct {
	MihomoRunning bool   // mihomo.exe process is alive and API is reachable
	MihomoMode    string // current mode: "rule", "global", "direct", "unknown", ""
	SystemProxy   bool   // Windows system proxy is enabled
	TunMode       bool   // TUN mode is active
}

// Poller periodically checks mihomo and system proxy state.
type Poller struct {
	mu     sync.RWMutex
	state  AppState
	client *mihomo.Client
	subs   []chan AppState
	muSubs sync.Mutex
	stopCh chan struct{}
}

// NewPoller creates a Poller and returns it.
func NewPoller(client *mihomo.Client) *Poller {
	return &Poller{
		client: client,
		stopCh: make(chan struct{}),
	}
}

// Subscribe returns a channel that receives state updates every poll cycle.
func (p *Poller) Subscribe() <-chan AppState {
	ch := make(chan AppState, 1)
	p.muSubs.Lock()
	p.subs = append(p.subs, ch)
	p.muSubs.Unlock()
	return ch
}

// Start begins the background polling loop. Call once.
func (p *Poller) Start() {
	// Do an immediate poll so we have data right away.
	p.poll()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.poll()
			case <-p.stopCh:
				return
			}
		}
	}()
}

// Stop terminates the polling loop.
func (p *Poller) Stop() {
	close(p.stopCh)
}

// GetState returns a copy of the current state (thread-safe).
func (p *Poller) GetState() AppState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

func (p *Poller) poll() {
	s := AppState{}

	// Check mihomo process and API.
	s.MihomoRunning = mihomo.IsRunning()
	if s.MihomoRunning {
		// Verify API is actually reachable.
		if p.client.IsAlive() {
			s.MihomoMode = p.client.GetMode()
			s.TunMode = p.client.HasTUN()
		} else {
			// Process exists but API not ready yet — still consider running
			// but mode is unknown.
			s.MihomoMode = "unknown"
		}
	}

	// Check system proxy.
	s.SystemProxy = registry.IsSystemProxyEnabled()

	p.mu.Lock()
	p.state = s
	p.mu.Unlock()

	// Notify subscribers.
	p.muSubs.Lock()
	for _, ch := range p.subs {
		select {
		case ch <- s:
		default:
			// Drop if subscriber is not reading fast enough.
		}
	}
	p.muSubs.Unlock()
}
