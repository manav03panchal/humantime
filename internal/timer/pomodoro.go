// Package timer provides timer functionality for Humantime.
package timer

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

// PomodoroConfig holds the configuration for a Pomodoro session.
type PomodoroConfig struct {
	WorkDuration      time.Duration
	BreakDuration     time.Duration
	LongBreakDuration time.Duration
	SessionsBeforeLong int
	TotalSessions     int // 0 means infinite
}

// DefaultPomodoroConfig returns the default Pomodoro configuration.
func DefaultPomodoroConfig() PomodoroConfig {
	return PomodoroConfig{
		WorkDuration:      25 * time.Minute,
		BreakDuration:     5 * time.Minute,
		LongBreakDuration: 15 * time.Minute,
		SessionsBeforeLong: 4,
		TotalSessions:     4, // 4 work sessions by default
	}
}

// PomodoroState represents the current state of the pomodoro timer.
type PomodoroState struct {
	CurrentSession    int
	CurrentType       SessionType
	Remaining         time.Duration
	TotalDuration     time.Duration
	Paused            bool
	TotalWorkTime     time.Duration
	WorkSessionsDone  int
	StartTime         time.Time
	PauseTime         time.Time
	InterruptedAt     *time.Time // Set if user quits early
}

// PomodoroEvent represents events from the pomodoro timer.
type PomodoroEvent int

const (
	EventTick PomodoroEvent = iota
	EventSessionComplete
	EventAllComplete
	EventPaused
	EventResumed
	EventSkipped
	EventQuit
)

// PomodoroCallback is called when events occur.
type PomodoroCallback func(event PomodoroEvent, state PomodoroState)

// Pomodoro manages a pomodoro timer session.
type Pomodoro struct {
	config   PomodoroConfig
	state    PomodoroState
	display  *CountdownDisplay
	callback PomodoroCallback

	mu       sync.RWMutex
	cancelFn context.CancelFunc
	done     chan struct{}

	// Control channels
	pauseCh  chan struct{}
	skipCh   chan struct{}
	quitCh   chan struct{}
}

// NewPomodoro creates a new Pomodoro timer.
func NewPomodoro(config PomodoroConfig) *Pomodoro {
	return &Pomodoro{
		config:  config,
		display: NewCountdownDisplay(),
		state: PomodoroState{
			CurrentSession:  1,
			CurrentType:     SessionWork,
			Remaining:       config.WorkDuration,
			TotalDuration:   config.WorkDuration,
		},
		done:    make(chan struct{}),
		pauseCh: make(chan struct{}, 1),
		skipCh:  make(chan struct{}, 1),
		quitCh:  make(chan struct{}, 1),
	}
}

// SetCallback sets the event callback.
func (p *Pomodoro) SetCallback(cb PomodoroCallback) {
	p.callback = cb
}

// SetDisplay sets the countdown display.
func (p *Pomodoro) SetDisplay(display *CountdownDisplay) {
	p.display = display
}

// GetState returns a copy of the current state.
func (p *Pomodoro) GetState() PomodoroState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// Pause pauses the timer.
func (p *Pomodoro) Pause() {
	select {
	case p.pauseCh <- struct{}{}:
	default:
	}
}

// Skip skips the current session.
func (p *Pomodoro) Skip() {
	select {
	case p.skipCh <- struct{}{}:
	default:
	}
}

// Quit quits the pomodoro timer.
func (p *Pomodoro) Quit() {
	select {
	case p.quitCh <- struct{}{}:
	default:
	}
}

// Run starts the pomodoro timer and blocks until complete or quit.
func (p *Pomodoro) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancelFn = cancel
	defer cancel()

	// Set up raw terminal mode for keyboard input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Handle OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Start keyboard listener
	go p.listenKeyboard(ctx)

	// Main timer loop
	totalSessions := p.config.TotalSessions
	if totalSessions == 0 {
		totalSessions = 1000 // Effectively infinite
	}

	for p.state.CurrentSession <= totalSessions {
		// Determine session type and duration
		p.mu.Lock()
		if p.state.CurrentType == SessionWork {
			p.state.Remaining = p.config.WorkDuration
			p.state.TotalDuration = p.config.WorkDuration
			p.state.StartTime = time.Now()
		} else if p.state.CurrentType == SessionLongBreak {
			p.state.Remaining = p.config.LongBreakDuration
			p.state.TotalDuration = p.config.LongBreakDuration
		} else {
			p.state.Remaining = p.config.BreakDuration
			p.state.TotalDuration = p.config.BreakDuration
		}
		p.mu.Unlock()

		// Run the current session
		result := p.runSession(ctx, sigCh)

		switch result {
		case EventQuit:
			// User quit - record interruption
			p.mu.Lock()
			now := time.Now()
			p.state.InterruptedAt = &now
			if p.state.CurrentType == SessionWork {
				// Calculate partial work time
				elapsed := p.state.TotalDuration - p.state.Remaining
				p.state.TotalWorkTime += elapsed
			}
			p.mu.Unlock()
			if p.callback != nil {
				p.callback(EventQuit, p.GetState())
			}
			return nil

		case EventSkipped:
			// Skip to next session
			if p.callback != nil {
				p.callback(EventSkipped, p.GetState())
			}
			p.advanceSession()

		case EventSessionComplete:
			// Session completed normally
			if p.callback != nil {
				p.callback(EventSessionComplete, p.GetState())
			}

			p.mu.Lock()
			if p.state.CurrentType == SessionWork {
				p.state.TotalWorkTime += p.config.WorkDuration
				p.state.WorkSessionsDone++
			}
			p.mu.Unlock()

			p.advanceSession()

			// Check if all sessions complete
			if p.state.CurrentSession > totalSessions {
				if p.callback != nil {
					p.callback(EventAllComplete, p.GetState())
				}
				// Show completion message
				p.display.ClearScreen()
				output := p.display.RenderAllComplete(p.state.TotalWorkTime, p.state.WorkSessionsDone)
				os.Stdout.WriteString(output + "\n")
				return nil
			}

			// Brief pause between sessions
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

// runSession runs a single pomodoro session.
func (p *Pomodoro) runSession(ctx context.Context, sigCh <-chan os.Signal) PomodoroEvent {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastUpdate := time.Now()

	for {
		select {
		case <-ctx.Done():
			return EventQuit

		case <-sigCh:
			return EventQuit

		case <-p.quitCh:
			return EventQuit

		case <-p.skipCh:
			return EventSkipped

		case <-p.pauseCh:
			p.mu.Lock()
			p.state.Paused = !p.state.Paused
			if p.state.Paused {
				p.state.PauseTime = time.Now()
			}
			paused := p.state.Paused
			p.mu.Unlock()

			if paused {
				if p.callback != nil {
					p.callback(EventPaused, p.GetState())
				}
			} else {
				if p.callback != nil {
					p.callback(EventResumed, p.GetState())
				}
			}
			p.render()

		case <-ticker.C:
			p.mu.Lock()
			if !p.state.Paused {
				elapsed := time.Since(lastUpdate)
				p.state.Remaining -= elapsed

				if p.state.Remaining <= 0 {
					p.state.Remaining = 0
					p.mu.Unlock()
					p.render()
					return EventSessionComplete
				}
			}
			p.mu.Unlock()

			lastUpdate = time.Now()

			// Send tick event
			if p.callback != nil {
				p.callback(EventTick, p.GetState())
			}

			p.render()
		}
	}
}

// advanceSession moves to the next session.
func (p *Pomodoro) advanceSession() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state.CurrentType == SessionWork {
		// After work, take a break
		if p.state.WorkSessionsDone > 0 && p.state.WorkSessionsDone%p.config.SessionsBeforeLong == 0 {
			p.state.CurrentType = SessionLongBreak
		} else {
			p.state.CurrentType = SessionBreak
		}
	} else {
		// After break, start work
		p.state.CurrentType = SessionWork
		p.state.CurrentSession++
	}
}

// render updates the display.
func (p *Pomodoro) render() {
	state := p.GetState()

	p.display.MoveCursorHome()
	p.display.ClearScreen()

	totalSessions := p.config.TotalSessions
	if totalSessions == 0 {
		totalSessions = state.CurrentSession // Show current as total for infinite
	}

	output := p.display.RenderTimer(
		state.Remaining,
		state.TotalDuration,
		state.CurrentType,
		state.CurrentSession,
		totalSessions,
		state.Paused,
	)

	os.Stdout.WriteString(output)
}

// listenKeyboard listens for keyboard input.
func (p *Pomodoro) listenKeyboard(ctx context.Context) {
	buf := make([]byte, 1)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Non-blocking read
			os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				continue
			}

			switch buf[0] {
			case ' ': // Space - pause/resume
				p.Pause()
			case 's', 'S': // S - skip
				p.Skip()
			case 'q', 'Q', 3: // Q or Ctrl+C - quit
				p.Quit()
			}
		}
	}
}

// CalculateWorkTime returns the total work time from a pomodoro session.
// If interrupted, it includes partial time.
func (p *Pomodoro) CalculateWorkTime() time.Duration {
	return p.GetState().TotalWorkTime
}

// WasInterrupted returns true if the session was quit early.
func (p *Pomodoro) WasInterrupted() bool {
	return p.GetState().InterruptedAt != nil
}
