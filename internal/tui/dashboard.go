package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/storage"
)

// tickMsg is sent when the timer ticks.
type tickMsg time.Time

// refreshMsg is sent when data needs to be refreshed.
type refreshMsg struct{}

// startTrackingMsg is sent when user wants to start tracking.
type startTrackingMsg struct{}

// stopTrackingMsg is sent when user wants to stop tracking.
type stopTrackingMsg struct{}

// errMsg is sent when an error occurs.
type errMsg struct {
	err error
}

// DashboardModel is the main bubbletea model for the dashboard.
type DashboardModel struct {
	// Data
	activeBlock  *model.Block
	recentBlocks []*model.Block
	goals        []*model.Goal
	goalProgress map[string]time.Duration

	// Repositories
	blockRepo       *storage.BlockRepo
	activeBlockRepo *storage.ActiveBlockRepo
	goalRepo        *storage.GoalRepo

	// UI state
	width      int
	height     int
	err        error
	message    string
	messageExp time.Time

	// Configuration
	refreshInterval time.Duration
	maxRecentBlocks int
}

// DashboardConfig holds configuration for the dashboard.
type DashboardConfig struct {
	BlockRepo       *storage.BlockRepo
	ActiveBlockRepo *storage.ActiveBlockRepo
	GoalRepo        *storage.GoalRepo
	RefreshInterval time.Duration
	MaxRecentBlocks int
}

// NewDashboardModel creates a new dashboard model.
func NewDashboardModel(config DashboardConfig) *DashboardModel {
	if config.RefreshInterval == 0 {
		config.RefreshInterval = time.Second
	}
	if config.MaxRecentBlocks == 0 {
		config.MaxRecentBlocks = 5
	}

	return &DashboardModel{
		blockRepo:       config.BlockRepo,
		activeBlockRepo: config.ActiveBlockRepo,
		goalRepo:        config.GoalRepo,
		refreshInterval: config.RefreshInterval,
		maxRecentBlocks: config.MaxRecentBlocks,
		goalProgress:    make(map[string]time.Duration),
	}
}

// Init initializes the model.
func (m *DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
		m.refreshCmd(),
	)
}

// Update handles messages and updates the model.
func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		// Clear expired messages
		if !m.messageExp.IsZero() && time.Now().After(m.messageExp) {
			m.message = ""
			m.messageExp = time.Time{}
		}
		return m, m.tickCmd()

	case refreshMsg:
		m.loadData()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input.
func (m *DashboardModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "s":
		// Start tracking - show message that user should use CLI
		m.setMessage("Use 'humantime start on <project>' to start tracking", 3*time.Second)
		return m, nil

	case "e":
		// Stop tracking
		if m.activeBlock != nil {
			if err := m.stopTracking(); err != nil {
				m.err = err
			} else {
				m.setMessage("Tracking stopped", 2*time.Second)
				m.loadData()
			}
		} else {
			m.setMessage("No active tracking to stop", 2*time.Second)
		}
		return m, nil

	case "r":
		// Refresh data
		m.loadData()
		m.setMessage("Refreshed", time.Second)
		return m, nil
	}

	return m, nil
}

// View renders the dashboard.
func (m *DashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Header
	header := m.renderHeader()
	sections = append(sections, header)

	// Error message
	if m.err != nil {
		errBox := StyleError.Render(fmt.Sprintf("Error: %v", m.err))
		sections = append(sections, errBox)
	}

	// Status message
	if m.message != "" {
		msgBox := StyleWarning.Render(m.message)
		sections = append(sections, msgBox)
	}

	// Status component (current tracking)
	statusComp := NewStatusComponent(m.activeBlock, m.width)
	sections = append(sections, statusComp.View())

	// Goal progress (if any goals exist)
	if len(m.goals) > 0 && m.activeBlock != nil {
		for _, goal := range m.goals {
			if goal.ProjectSID == m.activeBlock.ProjectSID {
				current := m.goalProgress[goal.ProjectSID]
				goalComp := NewGoalComponent(goal, current, m.width)
				goalView := goalComp.View()
				if goalView != "" {
					sections = append(sections, goalView)
				}
				break
			}
		}
	}

	// Recent blocks
	blocksComp := NewBlocksComponent(m.recentBlocks, m.width, m.maxRecentBlocks)
	sections = append(sections, blocksComp.View())

	// Help bar
	sections = append(sections, HelpBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the dashboard header.
func (m *DashboardModel) renderHeader() string {
	title := StyleTitle.Render("Humantime Dashboard")
	now := time.Now().Format("Mon Jan 2, 15:04:05")
	timeStr := StyleSubtitle.Render(now)

	return lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", timeStr) + "\n"
}

// loadData loads all data from repositories.
func (m *DashboardModel) loadData() {
	// Load active block
	block, err := m.activeBlockRepo.GetActiveBlock(m.blockRepo)
	if err != nil {
		m.err = err
		return
	}
	m.activeBlock = block

	// Load recent blocks
	filter := storage.BlockFilter{
		Limit: m.maxRecentBlocks + 1, // +1 to exclude active block if present
	}
	blocks, err := m.blockRepo.ListFiltered(filter)
	if err != nil {
		m.err = err
		return
	}

	// Filter out active block from recent
	m.recentBlocks = nil
	for _, b := range blocks {
		if m.activeBlock != nil && b.Key == m.activeBlock.Key {
			continue
		}
		m.recentBlocks = append(m.recentBlocks, b)
		if len(m.recentBlocks) >= m.maxRecentBlocks {
			break
		}
	}

	// Load goals
	goals, err := m.goalRepo.List()
	if err != nil {
		// Goals are optional, don't fail on error
		m.goals = nil
	} else {
		m.goals = goals
		m.calculateGoalProgress()
	}

	m.err = nil
}

// calculateGoalProgress calculates progress for all goals.
func (m *DashboardModel) calculateGoalProgress() {
	m.goalProgress = make(map[string]time.Duration)

	for _, goal := range m.goals {
		var start, end time.Time
		now := time.Now()

		if goal.Type == model.GoalTypeDaily {
			start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			end = start.AddDate(0, 0, 1)
		} else {
			// Weekly
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7 // Sunday is 7
			}
			start = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
			end = start.AddDate(0, 0, 7)
		}

		// Get blocks in this period for this project
		filter := storage.BlockFilter{
			ProjectSID: goal.ProjectSID,
			StartAfter: start,
			EndBefore:  end,
		}
		blocks, err := m.blockRepo.ListFiltered(filter)
		if err != nil {
			continue
		}

		var total time.Duration
		for _, b := range blocks {
			total += b.Duration()
		}
		m.goalProgress[goal.ProjectSID] = total
	}
}

// stopTracking stops the current active tracking.
func (m *DashboardModel) stopTracking() error {
	if m.activeBlock == nil {
		return nil
	}

	// Set end time
	m.activeBlock.TimestampEnd = time.Now()

	// Update the block
	if err := m.blockRepo.Update(m.activeBlock); err != nil {
		return err
	}

	// Clear active tracking
	if err := m.activeBlockRepo.ClearActive(); err != nil {
		return err
	}

	return nil
}

// setMessage sets a temporary message.
func (m *DashboardModel) setMessage(msg string, duration time.Duration) {
	m.message = msg
	m.messageExp = time.Now().Add(duration)
}

// tickCmd returns a command that sends a tick message.
func (m *DashboardModel) tickCmd() tea.Cmd {
	return tea.Tick(m.refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// refreshCmd returns a command that sends a refresh message.
func (m *DashboardModel) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		return refreshMsg{}
	}
}

// Run starts the dashboard TUI.
func Run(config DashboardConfig) error {
	model := NewDashboardModel(config)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunDashboard is an alias for Run for backwards compatibility.
func RunDashboard(blockRepo *storage.BlockRepo, activeBlockRepo *storage.ActiveBlockRepo, goalRepo *storage.GoalRepo) error {
	config := DashboardConfig{
		BlockRepo:       blockRepo,
		ActiveBlockRepo: activeBlockRepo,
		GoalRepo:        goalRepo,
	}
	return Run(config)
}
