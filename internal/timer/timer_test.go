package timer

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// CountdownDisplay Tests
// =============================================================================

func TestNewCountdownDisplay(t *testing.T) {
	cd := NewCountdownDisplay()
	assert.NotNil(t, cd)
	assert.NotNil(t, cd.Writer)
	assert.True(t, cd.UseColor)
	assert.True(t, cd.ShowSeconds)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00"},
		{30 * time.Second, "00:30"},
		{1 * time.Minute, "01:00"},
		{1*time.Minute + 30*time.Second, "01:30"},
		{5 * time.Minute, "05:00"},
		{25 * time.Minute, "25:00"},
		{59*time.Minute + 59*time.Second, "59:59"},
		{1 * time.Hour, "01:00:00"},
		{1*time.Hour + 30*time.Minute, "01:30:00"},
		{2*time.Hour + 15*time.Minute + 30*time.Second, "02:15:30"},
		{-5 * time.Second, "00:00"}, // Negative treated as 0
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSessionTypeString(t *testing.T) {
	tests := []struct {
		sessionType SessionType
		expected    string
	}{
		{SessionWork, "WORK"},
		{SessionBreak, "BREAK"},
		{SessionLongBreak, "LONG BREAK"},
		{SessionType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.sessionType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountdownDisplayRenderTimer(t *testing.T) {
	t.Run("work_session_no_color", func(t *testing.T) {
		var buf bytes.Buffer
		cd := &CountdownDisplay{
			Writer:      &buf,
			UseColor:    false,
			ShowSeconds: true,
		}

		output := cd.RenderTimer(
			24*time.Minute+30*time.Second,
			25*time.Minute,
			SessionWork,
			1,
			4,
			false,
		)

		assert.Contains(t, output, "WORK")
		assert.Contains(t, output, "[1/4]")
		assert.Contains(t, output, "24:30")
		assert.Contains(t, output, "Press SPACE")
	})

	t.Run("paused", func(t *testing.T) {
		cd := &CountdownDisplay{
			UseColor: false,
		}

		output := cd.RenderTimer(
			10*time.Minute,
			25*time.Minute,
			SessionWork,
			1,
			4,
			true,
		)

		assert.Contains(t, output, "[PAUSED]")
	})

	t.Run("break_session", func(t *testing.T) {
		cd := &CountdownDisplay{
			UseColor: false,
		}

		output := cd.RenderTimer(
			3*time.Minute,
			5*time.Minute,
			SessionBreak,
			2,
			4,
			false,
		)

		assert.Contains(t, output, "BREAK")
	})

	t.Run("long_break_session", func(t *testing.T) {
		cd := &CountdownDisplay{
			UseColor: false,
		}

		output := cd.RenderTimer(
			10*time.Minute,
			15*time.Minute,
			SessionLongBreak,
			4,
			4,
			false,
		)

		assert.Contains(t, output, "LONG BREAK")
	})

	t.Run("with_color", func(t *testing.T) {
		cd := &CountdownDisplay{
			UseColor: true,
		}

		output := cd.RenderTimer(
			10*time.Minute,
			25*time.Minute,
			SessionWork,
			1,
			4,
			false,
		)

		// Should contain the text even with ANSI codes
		assert.Contains(t, output, "WORK")
		assert.Contains(t, output, "10:00")
	})
}

func TestCountdownDisplayRenderProgressBar(t *testing.T) {
	cd := &CountdownDisplay{}

	t.Run("zero_progress", func(t *testing.T) {
		bar := cd.renderProgressBar(0, 10)
		assert.Contains(t, bar, "0%")
	})

	t.Run("half_progress", func(t *testing.T) {
		bar := cd.renderProgressBar(0.5, 10)
		assert.Contains(t, bar, "50%")
	})

	t.Run("full_progress", func(t *testing.T) {
		bar := cd.renderProgressBar(1.0, 10)
		assert.Contains(t, bar, "100%")
	})

	t.Run("over_progress_capped", func(t *testing.T) {
		bar := cd.renderProgressBar(1.5, 10)
		// Bar width is capped at width, but percentage shows actual value
		assert.Contains(t, bar, "150%")
	})

	t.Run("negative_progress_capped", func(t *testing.T) {
		bar := cd.renderProgressBar(-0.5, 10)
		assert.Contains(t, bar, "0%")
	})
}

func TestCountdownDisplayClearScreen(t *testing.T) {
	var buf bytes.Buffer
	cd := &CountdownDisplay{Writer: &buf}

	cd.ClearScreen()
	assert.Contains(t, buf.String(), "\033[H\033[2J")
}

func TestCountdownDisplayMoveCursorHome(t *testing.T) {
	var buf bytes.Buffer
	cd := &CountdownDisplay{Writer: &buf}

	cd.MoveCursorHome()
	assert.Contains(t, buf.String(), "\033[H")
}

func TestCountdownDisplayRenderComplete(t *testing.T) {
	t.Run("work_complete_break_next", func(t *testing.T) {
		cd := &CountdownDisplay{UseColor: false}

		output := cd.RenderComplete(SessionWork, SessionBreak)
		assert.Contains(t, output, "WORK session complete!")
		assert.Contains(t, output, "BREAK")
	})

	t.Run("break_complete_work_next", func(t *testing.T) {
		cd := &CountdownDisplay{UseColor: false}

		output := cd.RenderComplete(SessionBreak, SessionWork)
		assert.Contains(t, output, "BREAK session complete!")
		assert.Contains(t, output, "WORK")
	})

	t.Run("same_type_no_next_message", func(t *testing.T) {
		cd := &CountdownDisplay{UseColor: false}

		output := cd.RenderComplete(SessionWork, SessionWork)
		assert.Contains(t, output, "WORK session complete!")
		assert.NotContains(t, output, "Starting")
	})

	t.Run("with_color", func(t *testing.T) {
		cd := &CountdownDisplay{UseColor: true}

		output := cd.RenderComplete(SessionWork, SessionBreak)
		assert.Contains(t, output, "session complete!")
	})
}

func TestCountdownDisplayRenderAllComplete(t *testing.T) {
	t.Run("no_color", func(t *testing.T) {
		cd := &CountdownDisplay{UseColor: false}

		output := cd.RenderAllComplete(100*time.Minute, 4)
		assert.Contains(t, output, "Pomodoro session complete!")
		assert.Contains(t, output, "Completed 4 work sessions")
		assert.Contains(t, output, "01:40:00")
	})

	t.Run("with_color", func(t *testing.T) {
		cd := &CountdownDisplay{UseColor: true}

		output := cd.RenderAllComplete(25*time.Minute, 1)
		assert.Contains(t, output, "Pomodoro session complete!")
	})
}

// =============================================================================
// PomodoroConfig Tests
// =============================================================================

func TestDefaultPomodoroConfig(t *testing.T) {
	config := DefaultPomodoroConfig()

	assert.Equal(t, 25*time.Minute, config.WorkDuration)
	assert.Equal(t, 5*time.Minute, config.BreakDuration)
	assert.Equal(t, 15*time.Minute, config.LongBreakDuration)
	assert.Equal(t, 4, config.SessionsBeforeLong)
	assert.Equal(t, 4, config.TotalSessions)
}

func TestPomodoroConfigStruct(t *testing.T) {
	config := PomodoroConfig{
		WorkDuration:       30 * time.Minute,
		BreakDuration:      10 * time.Minute,
		LongBreakDuration:  20 * time.Minute,
		SessionsBeforeLong: 3,
		TotalSessions:      6,
	}

	assert.Equal(t, 30*time.Minute, config.WorkDuration)
	assert.Equal(t, 10*time.Minute, config.BreakDuration)
	assert.Equal(t, 20*time.Minute, config.LongBreakDuration)
	assert.Equal(t, 3, config.SessionsBeforeLong)
	assert.Equal(t, 6, config.TotalSessions)
}

// =============================================================================
// Pomodoro Tests
// =============================================================================

func TestNewPomodoro(t *testing.T) {
	config := DefaultPomodoroConfig()
	p := NewPomodoro(config)

	assert.NotNil(t, p)
	assert.NotNil(t, p.display)
	assert.NotNil(t, p.done)
	assert.NotNil(t, p.pauseCh)
	assert.NotNil(t, p.skipCh)
	assert.NotNil(t, p.quitCh)

	state := p.GetState()
	assert.Equal(t, 1, state.CurrentSession)
	assert.Equal(t, SessionWork, state.CurrentType)
	assert.Equal(t, config.WorkDuration, state.Remaining)
	assert.Equal(t, config.WorkDuration, state.TotalDuration)
	assert.False(t, state.Paused)
}

func TestPomodoroSetCallback(t *testing.T) {
	p := NewPomodoro(DefaultPomodoroConfig())

	p.SetCallback(func(event PomodoroEvent, state PomodoroState) {
		// Callback is set
	})

	assert.NotNil(t, p.callback)
}

func TestPomodoroSetDisplay(t *testing.T) {
	p := NewPomodoro(DefaultPomodoroConfig())
	newDisplay := &CountdownDisplay{UseColor: false}

	p.SetDisplay(newDisplay)
	assert.Equal(t, newDisplay, p.display)
}

func TestPomodoroGetState(t *testing.T) {
	p := NewPomodoro(DefaultPomodoroConfig())
	state := p.GetState()

	assert.Equal(t, 1, state.CurrentSession)
	assert.Equal(t, SessionWork, state.CurrentType)
	assert.False(t, state.Paused)
}

func TestPomodoroPause(t *testing.T) {
	p := NewPomodoro(DefaultPomodoroConfig())

	// Should not block when called multiple times
	p.Pause()
	p.Pause()
	p.Pause()

	// Channel should have one message
	select {
	case <-p.pauseCh:
		// Expected
	default:
		t.Error("Expected message in pause channel")
	}
}

func TestPomodoroSkip(t *testing.T) {
	p := NewPomodoro(DefaultPomodoroConfig())

	p.Skip()

	select {
	case <-p.skipCh:
		// Expected
	default:
		t.Error("Expected message in skip channel")
	}
}

func TestPomodoroQuit(t *testing.T) {
	p := NewPomodoro(DefaultPomodoroConfig())

	p.Quit()

	select {
	case <-p.quitCh:
		// Expected
	default:
		t.Error("Expected message in quit channel")
	}
}

func TestPomodoroAdvanceSession(t *testing.T) {
	config := DefaultPomodoroConfig()
	config.SessionsBeforeLong = 2
	p := NewPomodoro(config)

	// Initially at work session 1
	assert.Equal(t, SessionWork, p.GetState().CurrentType)
	assert.Equal(t, 1, p.GetState().CurrentSession)

	// After first work session -> break
	p.state.WorkSessionsDone = 1
	p.advanceSession()
	assert.Equal(t, SessionBreak, p.GetState().CurrentType)
	assert.Equal(t, 1, p.GetState().CurrentSession)

	// After break -> work session 2
	p.advanceSession()
	assert.Equal(t, SessionWork, p.GetState().CurrentType)
	assert.Equal(t, 2, p.GetState().CurrentSession)

	// After second work session -> long break (because SessionsBeforeLong = 2)
	p.state.WorkSessionsDone = 2
	p.advanceSession()
	assert.Equal(t, SessionLongBreak, p.GetState().CurrentType)
}

func TestPomodoroCalculateWorkTime(t *testing.T) {
	p := NewPomodoro(DefaultPomodoroConfig())

	// Initially 0
	assert.Equal(t, time.Duration(0), p.CalculateWorkTime())

	// Set some work time
	p.mu.Lock()
	p.state.TotalWorkTime = 50 * time.Minute
	p.mu.Unlock()

	assert.Equal(t, 50*time.Minute, p.CalculateWorkTime())
}

func TestPomodoroWasInterrupted(t *testing.T) {
	p := NewPomodoro(DefaultPomodoroConfig())

	// Initially not interrupted
	assert.False(t, p.WasInterrupted())

	// Set interrupted
	p.mu.Lock()
	now := time.Now()
	p.state.InterruptedAt = &now
	p.mu.Unlock()

	assert.True(t, p.WasInterrupted())
}

// =============================================================================
// PomodoroState Tests
// =============================================================================

func TestPomodoroStateStruct(t *testing.T) {
	now := time.Now()
	state := PomodoroState{
		CurrentSession:   2,
		CurrentType:      SessionBreak,
		Remaining:        3 * time.Minute,
		TotalDuration:    5 * time.Minute,
		Paused:           true,
		TotalWorkTime:    50 * time.Minute,
		WorkSessionsDone: 2,
		StartTime:        now,
		PauseTime:        now,
		InterruptedAt:    &now,
	}

	assert.Equal(t, 2, state.CurrentSession)
	assert.Equal(t, SessionBreak, state.CurrentType)
	assert.Equal(t, 3*time.Minute, state.Remaining)
	assert.Equal(t, 5*time.Minute, state.TotalDuration)
	assert.True(t, state.Paused)
	assert.Equal(t, 50*time.Minute, state.TotalWorkTime)
	assert.Equal(t, 2, state.WorkSessionsDone)
	assert.NotNil(t, state.InterruptedAt)
}

// =============================================================================
// PomodoroEvent Tests
// =============================================================================

func TestPomodoroEventConstants(t *testing.T) {
	assert.Equal(t, PomodoroEvent(0), EventTick)
	assert.Equal(t, PomodoroEvent(1), EventSessionComplete)
	assert.Equal(t, PomodoroEvent(2), EventAllComplete)
	assert.Equal(t, PomodoroEvent(3), EventPaused)
	assert.Equal(t, PomodoroEvent(4), EventResumed)
	assert.Equal(t, PomodoroEvent(5), EventSkipped)
	assert.Equal(t, PomodoroEvent(6), EventQuit)
}

// =============================================================================
// SessionType Constants Tests
// =============================================================================

func TestSessionTypeConstants(t *testing.T) {
	assert.Equal(t, SessionType(0), SessionWork)
	assert.Equal(t, SessionType(1), SessionBreak)
	assert.Equal(t, SessionType(2), SessionLongBreak)
}
