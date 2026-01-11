package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Block Tests
// =============================================================================

func TestNewBlock(t *testing.T) {
	start := time.Now()
	block := NewBlock("owner1", "myproject", "task1", "Working on feature", start)

	assert.NotNil(t, block)
	assert.Equal(t, "owner1", block.OwnerKey)
	assert.Equal(t, "myproject", block.ProjectSID)
	assert.Equal(t, "task1", block.TaskSID)
	assert.Equal(t, "Working on feature", block.Note)
	assert.Equal(t, start, block.TimestampStart)
	assert.True(t, block.TimestampEnd.IsZero())
}

func TestBlockSetGetKey(t *testing.T) {
	block := &Block{}
	block.SetKey("block:abc123")
	assert.Equal(t, "block:abc123", block.GetKey())
}

func TestBlockHasTag(t *testing.T) {
	block := &Block{
		Tags: []string{"urgent", "Bug", "IMPORTANT"},
	}

	// Case-insensitive matching
	assert.True(t, block.HasTag("urgent"))
	assert.True(t, block.HasTag("URGENT"))
	assert.True(t, block.HasTag("bug"))
	assert.True(t, block.HasTag("important"))
	assert.False(t, block.HasTag("feature"))
	assert.False(t, block.HasTag(""))

	// Empty tags
	emptyBlock := &Block{}
	assert.False(t, emptyBlock.HasTag("any"))
}

func TestBlockIsActive(t *testing.T) {
	// Active block (no end time)
	active := &Block{
		TimestampStart: time.Now(),
	}
	assert.True(t, active.IsActive())

	// Completed block
	completed := &Block{
		TimestampStart: time.Now().Add(-1 * time.Hour),
		TimestampEnd:   time.Now(),
	}
	assert.False(t, completed.IsActive())
}

func TestBlockDuration(t *testing.T) {
	t.Run("completed_block", func(t *testing.T) {
		start := time.Now().Add(-2 * time.Hour)
		end := time.Now()
		block := &Block{
			TimestampStart: start,
			TimestampEnd:   end,
		}

		duration := block.Duration()
		assert.InDelta(t, 2*time.Hour, duration, float64(time.Second))
	})

	t.Run("active_block", func(t *testing.T) {
		start := time.Now().Add(-30 * time.Minute)
		block := &Block{
			TimestampStart: start,
		}

		duration := block.Duration()
		assert.InDelta(t, 30*time.Minute, duration, float64(time.Second))
	})
}

func TestBlockDurationSeconds(t *testing.T) {
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	block := &Block{
		TimestampStart: start,
		TimestampEnd:   end,
	}

	seconds := block.DurationSeconds()
	assert.InDelta(t, 3600, seconds, 1)
}

func TestGenerateBlockKey(t *testing.T) {
	key := GenerateBlockKey("abc123")
	assert.Equal(t, "block:abc123", key)
}

// =============================================================================
// Project Tests
// =============================================================================

func TestNewProject(t *testing.T) {
	project := NewProject("myproject", "My Project", "#FF5733")

	assert.NotNil(t, project)
	assert.Equal(t, "project:myproject", project.Key)
	assert.Equal(t, "myproject", project.SID)
	assert.Equal(t, "My Project", project.DisplayName)
	assert.Equal(t, "#FF5733", project.Color)
}

func TestProjectSetGetKey(t *testing.T) {
	project := &Project{}
	project.SetKey("project:test")
	assert.Equal(t, "project:test", project.GetKey())
}

func TestGenerateProjectKey(t *testing.T) {
	key := GenerateProjectKey("myproject")
	assert.Equal(t, "project:myproject", key)
}

func TestValidateColor(t *testing.T) {
	tests := []struct {
		color string
		valid bool
	}{
		{"", true},
		{"#FF0000", true},
		{"#00FF00", true},
		{"#0000FF", true},
		{"#ff5733", true},
		{"#ABCDEF", true},
		{"FF0000", false},     // Missing #
		{"#FFF", false},       // Too short
		{"#FFFFFFF", false},   // Too long
		{"#GGGGGG", false},    // Invalid hex
		{"red", false},        // Named color
		{"#12345G", false},    // Invalid char
	}

	for _, tt := range tests {
		t.Run(tt.color, func(t *testing.T) {
			result := ValidateColor(tt.color)
			assert.Equal(t, tt.valid, result)
		})
	}
}

// =============================================================================
// Task Tests
// =============================================================================

func TestNewTask(t *testing.T) {
	task := NewTask("myproject", "feature", "Feature Work", "#00FF00")

	assert.NotNil(t, task)
	assert.Equal(t, "task:myproject:feature", task.Key)
	assert.Equal(t, "feature", task.SID)
	assert.Equal(t, "myproject", task.ProjectSID)
	assert.Equal(t, "Feature Work", task.DisplayName)
	assert.Equal(t, "#00FF00", task.Color)
}

func TestTaskSetGetKey(t *testing.T) {
	task := &Task{}
	task.SetKey("task:proj:t1")
	assert.Equal(t, "task:proj:t1", task.GetKey())
}

func TestGenerateTaskKey(t *testing.T) {
	key := GenerateTaskKey("myproject", "mytask")
	assert.Equal(t, "task:myproject:mytask", key)
}

// =============================================================================
// Goal Tests
// =============================================================================

func TestNewGoal(t *testing.T) {
	goal := NewGoal("myproject", GoalTypeDaily, 4*time.Hour)

	assert.NotNil(t, goal)
	assert.Equal(t, "goal:myproject", goal.Key)
	assert.Equal(t, "myproject", goal.ProjectSID)
	assert.Equal(t, GoalTypeDaily, goal.Type)
	assert.Equal(t, 4*time.Hour, goal.Target)
}

func TestGoalSetGetKey(t *testing.T) {
	goal := &Goal{}
	goal.SetKey("goal:test")
	assert.Equal(t, "goal:test", goal.GetKey())
}

func TestGenerateGoalKey(t *testing.T) {
	key := GenerateGoalKey("myproject")
	assert.Equal(t, "goal:myproject", key)
}

func TestGoalTargetSeconds(t *testing.T) {
	goal := &Goal{Target: 2 * time.Hour}
	assert.Equal(t, int64(7200), goal.TargetSeconds())
}

func TestGoalCalculateProgress(t *testing.T) {
	goal := &Goal{Target: 4 * time.Hour}

	t.Run("zero_progress", func(t *testing.T) {
		progress := goal.CalculateProgress(0)
		assert.Equal(t, time.Duration(0), progress.Current)
		assert.Equal(t, 4*time.Hour, progress.Remaining)
		assert.Equal(t, 0.0, progress.Percentage)
		assert.False(t, progress.IsComplete)
	})

	t.Run("partial_progress", func(t *testing.T) {
		progress := goal.CalculateProgress(2 * time.Hour)
		assert.Equal(t, 2*time.Hour, progress.Current)
		assert.Equal(t, 2*time.Hour, progress.Remaining)
		assert.Equal(t, 50.0, progress.Percentage)
		assert.False(t, progress.IsComplete)
	})

	t.Run("complete", func(t *testing.T) {
		progress := goal.CalculateProgress(4 * time.Hour)
		assert.Equal(t, 4*time.Hour, progress.Current)
		assert.Equal(t, time.Duration(0), progress.Remaining)
		assert.Equal(t, 100.0, progress.Percentage)
		assert.True(t, progress.IsComplete)
	})

	t.Run("over_complete", func(t *testing.T) {
		progress := goal.CalculateProgress(5 * time.Hour)
		assert.Equal(t, 5*time.Hour, progress.Current)
		assert.Equal(t, time.Duration(0), progress.Remaining)
		assert.Greater(t, progress.Percentage, 100.0)
		assert.True(t, progress.IsComplete)
	})
}

func TestGoalTypes(t *testing.T) {
	assert.Equal(t, GoalType("daily"), GoalTypeDaily)
	assert.Equal(t, GoalType("weekly"), GoalTypeWeekly)
}

// =============================================================================
// Reminder Tests
// =============================================================================

func TestNewReminder(t *testing.T) {
	deadline := time.Now().Add(24 * time.Hour)
	reminder := NewReminder("Submit report", deadline, "owner1")

	assert.NotNil(t, reminder)
	assert.Equal(t, "Submit report", reminder.Title)
	assert.Equal(t, deadline, reminder.Deadline)
	assert.Equal(t, "owner1", reminder.OwnerKey)
	assert.Equal(t, []string{"1h", "15m"}, reminder.NotifyBefore)
	assert.False(t, reminder.Completed)
}

func TestReminderSetGetKey(t *testing.T) {
	reminder := &Reminder{}
	reminder.SetKey("reminder:abc123")
	assert.Equal(t, "reminder:abc123", reminder.GetKey())
}

func TestReminderIsPending(t *testing.T) {
	pending := &Reminder{Completed: false}
	assert.True(t, pending.IsPending())

	completed := &Reminder{Completed: true}
	assert.False(t, completed.IsPending())
}

func TestReminderIsDue(t *testing.T) {
	past := &Reminder{Deadline: time.Now().Add(-1 * time.Hour)}
	assert.True(t, past.IsDue())

	future := &Reminder{Deadline: time.Now().Add(1 * time.Hour)}
	assert.False(t, future.IsDue())
}

func TestReminderIsDueWithin(t *testing.T) {
	reminder := &Reminder{Deadline: time.Now().Add(30 * time.Minute)}

	assert.True(t, reminder.IsDueWithin(1*time.Hour))
	assert.False(t, reminder.IsDueWithin(10*time.Minute))
}

func TestReminderIsRecurring(t *testing.T) {
	oneTime := &Reminder{RepeatRule: ""}
	assert.False(t, oneTime.IsRecurring())

	recurring := &Reminder{RepeatRule: "daily"}
	assert.True(t, recurring.IsRecurring())
}

func TestReminderNextDeadline(t *testing.T) {
	deadline := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("daily", func(t *testing.T) {
		r := &Reminder{Deadline: deadline, RepeatRule: "daily"}
		next := r.NextDeadline()
		assert.Equal(t, time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC), next)
	})

	t.Run("weekly", func(t *testing.T) {
		r := &Reminder{Deadline: deadline, RepeatRule: "weekly"}
		next := r.NextDeadline()
		assert.Equal(t, time.Date(2024, 1, 22, 10, 0, 0, 0, time.UTC), next)
	})

	t.Run("monthly", func(t *testing.T) {
		r := &Reminder{Deadline: deadline, RepeatRule: "monthly"}
		next := r.NextDeadline()
		assert.Equal(t, time.Date(2024, 2, 15, 10, 0, 0, 0, time.UTC), next)
	})

	t.Run("non_recurring", func(t *testing.T) {
		r := &Reminder{Deadline: deadline, RepeatRule: ""}
		next := r.NextDeadline()
		assert.Equal(t, deadline, next)
	})
}

func TestReminderTimeUntil(t *testing.T) {
	future := &Reminder{Deadline: time.Now().Add(1 * time.Hour)}
	until := future.TimeUntil()
	assert.InDelta(t, 1*time.Hour, until, float64(time.Second))
}

func TestReminderShortID(t *testing.T) {
	t.Run("long_key", func(t *testing.T) {
		r := &Reminder{Key: "reminder:abcdef-1234-5678"}
		assert.Equal(t, "abcdef", r.ShortID())
	})

	t.Run("short_key", func(t *testing.T) {
		r := &Reminder{Key: "rem:abc"}
		assert.Equal(t, "rem:abc", r.ShortID())
	})
}

func TestGenerateReminderKey(t *testing.T) {
	key := GenerateReminderKey("uuid-12345")
	assert.Equal(t, "reminder:uuid-12345", key)
}

func TestValidRepeatRules(t *testing.T) {
	rules := ValidRepeatRules()
	assert.Contains(t, rules, "")
	assert.Contains(t, rules, "daily")
	assert.Contains(t, rules, "weekly")
	assert.Contains(t, rules, "monthly")
}

func TestIsValidRepeatRule(t *testing.T) {
	assert.True(t, IsValidRepeatRule(""))
	assert.True(t, IsValidRepeatRule("daily"))
	assert.True(t, IsValidRepeatRule("weekly"))
	assert.True(t, IsValidRepeatRule("monthly"))
	assert.False(t, IsValidRepeatRule("yearly"))
	assert.False(t, IsValidRepeatRule("invalid"))
}

// =============================================================================
// Webhook Tests
// =============================================================================

func TestNewWebhook(t *testing.T) {
	webhook := NewWebhook("my-hook", WebhookTypeDiscord, "https://example.com/webhook/test")

	assert.NotNil(t, webhook)
	assert.Equal(t, "webhook:my-hook", webhook.Key)
	assert.Equal(t, "my-hook", webhook.Name)
	assert.Equal(t, WebhookTypeDiscord, webhook.Type)
	assert.Equal(t, "https://example.com/webhook/test", webhook.URL)
	assert.True(t, webhook.Enabled)
}

func TestWebhookSetGetKey(t *testing.T) {
	webhook := &Webhook{}
	webhook.SetKey("webhook:test")
	assert.Equal(t, "webhook:test", webhook.GetKey())
}

func TestWebhookIsEnabled(t *testing.T) {
	enabled := &Webhook{Enabled: true}
	assert.True(t, enabled.IsEnabled())

	disabled := &Webhook{Enabled: false}
	assert.False(t, disabled.IsEnabled())
}

func TestWebhookMaskedURL(t *testing.T) {
	t.Run("long_url", func(t *testing.T) {
		w := &Webhook{URL: "https://example.com/webhook/test456789/abcdefghijklmnopqrstuvwxyz"}
		masked := w.MaskedURL()
		assert.Equal(t, "https://example.com/webhook/te***", masked)
	})

	t.Run("short_url", func(t *testing.T) {
		w := &Webhook{URL: "https://example.com/hook"}
		masked := w.MaskedURL()
		assert.Equal(t, "https://example.com/hook", masked)
	})
}

func TestGenerateWebhookKey(t *testing.T) {
	key := GenerateWebhookKey("my-hook")
	assert.Equal(t, "webhook:my-hook", key)
}

func TestValidWebhookTypes(t *testing.T) {
	types := ValidWebhookTypes()
	assert.Contains(t, types, WebhookTypeDiscord)
	assert.Contains(t, types, WebhookTypeSlack)
	assert.Contains(t, types, WebhookTypeTeams)
	assert.Contains(t, types, WebhookTypeGeneric)
}

func TestIsValidWebhookType(t *testing.T) {
	assert.True(t, IsValidWebhookType("discord"))
	assert.True(t, IsValidWebhookType("slack"))
	assert.True(t, IsValidWebhookType("teams"))
	assert.True(t, IsValidWebhookType("generic"))
	assert.False(t, IsValidWebhookType("invalid"))
	assert.False(t, IsValidWebhookType(""))
}

func TestIsValidWebhookName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"my-hook", true},
		{"MyHook", true},
		{"hook_1", true},
		{"a", true},
		{"a-b-c_123", true},

		{"", false},
		{"-hook", false},
		{"_hook", false},
		{"hook with space", false},
		{"hook@special", false},
		{string(make([]byte, 51)), false}, // Too long
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidWebhookName(tt.name)
			assert.Equal(t, tt.valid, result, "name: %s", tt.name)
		})
	}
}

func TestDetectWebhookType(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://discord.com/api/webhooks/000/test", WebhookTypeDiscord},
		{"https://DISCORD.COM/API/WEBHOOKS/000/test", WebhookTypeDiscord},
		{"https://hooks.slack.com/services/T00/B00/XXX", WebhookTypeSlack},
		{"https://HOOKS.SLACK.COM/services/T00/B00/XXX", WebhookTypeSlack},
		{"https://outlook.office.com/webhook/abc", WebhookTypeTeams},
		{"https://webhook.office.com/abc", WebhookTypeTeams},
		{"https://example.com/webhook", WebhookTypeGeneric},
		{"https://myserver.com/notify", WebhookTypeGeneric},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectWebhookType(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// ActiveBlock Tests
// =============================================================================

func TestNewActiveBlock(t *testing.T) {
	ab := NewActiveBlock()
	assert.NotNil(t, ab)
	assert.Equal(t, KeyActiveBlock, ab.Key)
	assert.Empty(t, ab.ActiveBlockKey)
	assert.Empty(t, ab.PreviousBlockKey)
}

func TestActiveBlockSetGetKey(t *testing.T) {
	ab := &ActiveBlock{}
	ab.SetKey("activeblock")
	assert.Equal(t, "activeblock", ab.GetKey())
}

func TestActiveBlockIsTracking(t *testing.T) {
	ab := NewActiveBlock()
	assert.False(t, ab.IsTracking())

	ab.ActiveBlockKey = "block:123"
	assert.True(t, ab.IsTracking())
}

func TestActiveBlockSetActive(t *testing.T) {
	ab := NewActiveBlock()

	// First activation
	ab.SetActive("block:1")
	assert.Equal(t, "block:1", ab.ActiveBlockKey)
	assert.Empty(t, ab.PreviousBlockKey)

	// Second activation (previous saved)
	ab.SetActive("block:2")
	assert.Equal(t, "block:2", ab.ActiveBlockKey)
	assert.Equal(t, "block:1", ab.PreviousBlockKey)
}

func TestActiveBlockClearActive(t *testing.T) {
	ab := NewActiveBlock()
	ab.ActiveBlockKey = "block:1"

	ab.ClearActive()
	assert.Empty(t, ab.ActiveBlockKey)
	assert.Equal(t, "block:1", ab.PreviousBlockKey)

	// Clearing again when already empty
	ab.ClearActive()
	assert.Empty(t, ab.ActiveBlockKey)
	assert.Equal(t, "block:1", ab.PreviousBlockKey)
}

// =============================================================================
// UndoState Tests
// =============================================================================

func TestNewUndoState(t *testing.T) {
	block := &Block{Key: "block:123", ProjectSID: "proj"}
	state := NewUndoState(UndoActionStop, "block:123", block)

	assert.NotNil(t, state)
	assert.Equal(t, KeyUndo, state.Key)
	assert.Equal(t, UndoActionStop, state.Action)
	assert.Equal(t, "block:123", state.BlockKey)
	assert.Equal(t, block, state.BlockSnapshot)
}

func TestUndoStateSetGetKey(t *testing.T) {
	state := &UndoState{}
	state.SetKey("undo")
	assert.Equal(t, "undo", state.GetKey())
}

func TestUndoActionConstants(t *testing.T) {
	assert.Equal(t, UndoAction("start"), UndoActionStart)
	assert.Equal(t, UndoAction("stop"), UndoActionStop)
	assert.Equal(t, UndoAction("delete"), UndoActionDelete)
}

// =============================================================================
// Config Tests
// =============================================================================

func TestNewConfig(t *testing.T) {
	config := NewConfig("user123")

	assert.NotNil(t, config)
	assert.Equal(t, KeyConfig, config.Key)
	assert.Equal(t, "user123", config.UserKey)
}

func TestConfigSetGetKey(t *testing.T) {
	config := &Config{}
	config.SetKey("config")
	assert.Equal(t, "config", config.GetKey())
}

// =============================================================================
// Notification Tests
// =============================================================================

func TestNewNotification(t *testing.T) {
	n := NewNotification(NotifyReminder, "Reminder", "Don't forget!")

	assert.NotNil(t, n)
	assert.Equal(t, NotifyReminder, n.Type)
	assert.Equal(t, "Reminder", n.Title)
	assert.Equal(t, "Don't forget!", n.Message)
	assert.NotNil(t, n.Fields)
	assert.False(t, n.Timestamp.IsZero())
}

func TestNotificationWithField(t *testing.T) {
	n := NewNotification(NotifyGoal, "Goal", "50% complete")
	n.WithField("project", "myproject").WithField("progress", "50%")

	assert.Equal(t, "myproject", n.Fields["project"])
	assert.Equal(t, "50%", n.Fields["progress"])
}

func TestNotificationWithFieldNilMap(t *testing.T) {
	n := &Notification{}
	n.WithField("key", "value")
	assert.Equal(t, "value", n.Fields["key"])
}

func TestNotificationWithColor(t *testing.T) {
	n := NewNotification(NotifyTest, "Test", "Testing")
	n.WithColor(0xFF0000)

	assert.Equal(t, 0xFF0000, n.Color)
}

func TestDefaultColorForType(t *testing.T) {
	tests := []struct {
		notifyType NotificationType
		expected   int
	}{
		{NotifyReminder, ColorWarning},
		{NotifyIdle, ColorInfo},
		{NotifyBreak, ColorPrimary},
		{NotifyGoal, ColorSuccess},
		{NotifyDailySummary, ColorInfo},
		{NotifyEndOfDay, ColorSuccess},
		{NotifyTest, ColorPrimary},
		{NotificationType("unknown"), ColorInfo},
	}

	for _, tt := range tests {
		t.Run(string(tt.notifyType), func(t *testing.T) {
			color := DefaultColorForType(tt.notifyType)
			assert.Equal(t, tt.expected, color)
		})
	}
}

func TestNotificationIcon(t *testing.T) {
	tests := []struct {
		notifyType NotificationType
		expected   string
	}{
		{NotifyReminder, "bell"},
		{NotifyIdle, "pause_button"},
		{NotifyBreak, "coffee"},
		{NotifyGoal, "dart"},
		{NotifyDailySummary, "sunrise"},
		{NotifyEndOfDay, "moon"},
		{NotifyTest, "test_tube"},
		{NotificationType("unknown"), "bell"},
	}

	for _, tt := range tests {
		t.Run(string(tt.notifyType), func(t *testing.T) {
			n := &Notification{Type: tt.notifyType}
			icon := n.Icon()
			assert.Equal(t, tt.expected, icon)
		})
	}
}

func TestNotificationTypeLabel(t *testing.T) {
	tests := []struct {
		notifyType NotificationType
		expected   string
	}{
		{NotifyReminder, "Reminder"},
		{NotifyIdle, "Idle Detection"},
		{NotifyBreak, "Break Reminder"},
		{NotifyGoal, "Goal Progress"},
		{NotifyDailySummary, "Daily Summary"},
		{NotifyEndOfDay, "End of Day Recap"},
		{NotifyTest, "Test Notification"},
		{NotificationType("unknown"), "Notification"},
	}

	for _, tt := range tests {
		t.Run(string(tt.notifyType), func(t *testing.T) {
			n := &Notification{Type: tt.notifyType}
			label := n.TypeLabel()
			assert.Equal(t, tt.expected, label)
		})
	}
}

func TestNotificationTypeConstants(t *testing.T) {
	assert.Equal(t, NotificationType("reminder"), NotifyReminder)
	assert.Equal(t, NotificationType("idle"), NotifyIdle)
	assert.Equal(t, NotificationType("break"), NotifyBreak)
	assert.Equal(t, NotificationType("goal"), NotifyGoal)
	assert.Equal(t, NotificationType("daily_summary"), NotifyDailySummary)
	assert.Equal(t, NotificationType("end_of_day"), NotifyEndOfDay)
	assert.Equal(t, NotificationType("test"), NotifyTest)
}

func TestColorConstants(t *testing.T) {
	assert.Equal(t, 0x57F287, ColorSuccess)
	assert.Equal(t, 0xFEE75C, ColorWarning)
	assert.Equal(t, 0x5865F2, ColorInfo)
	assert.Equal(t, 0xED4245, ColorError)
	assert.Equal(t, 0x3498DB, ColorPrimary)
}

// =============================================================================
// NotifyConfig Tests
// =============================================================================

func TestDefaultNotifyConfig(t *testing.T) {
	config := DefaultNotifyConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 30*time.Minute, config.IdleAfter)
	assert.Equal(t, 2*time.Hour, config.BreakAfter)
	assert.Equal(t, 15*time.Minute, config.BreakReset)
	assert.Equal(t, []int{50, 75, 100}, config.GoalMilestones)
	assert.Equal(t, "09:00", config.DailySummaryAt)
	assert.Equal(t, "18:00", config.EndOfDayAt)
	assert.True(t, config.Enabled["idle"])
	assert.True(t, config.Enabled["break"])
	assert.True(t, config.Enabled["goal"])
	assert.True(t, config.Enabled["reminder"])
}

func TestNotifyConfigIsTypeEnabled(t *testing.T) {
	t.Run("enabled_explicitly", func(t *testing.T) {
		config := DefaultNotifyConfig()
		assert.True(t, config.IsTypeEnabled("idle"))
	})

	t.Run("disabled_explicitly", func(t *testing.T) {
		config := DefaultNotifyConfig()
		config.Enabled["idle"] = false
		assert.False(t, config.IsTypeEnabled("idle"))
	})

	t.Run("nil_map_defaults_true", func(t *testing.T) {
		config := &NotifyConfig{}
		assert.True(t, config.IsTypeEnabled("unknown"))
	})

	t.Run("not_in_map_defaults_true", func(t *testing.T) {
		config := DefaultNotifyConfig()
		assert.True(t, config.IsTypeEnabled("nonexistent"))
	})
}

func TestNotifyConfigSetTypeEnabled(t *testing.T) {
	config := &NotifyConfig{}

	// Sets on nil map
	config.SetTypeEnabled("test", true)
	assert.True(t, config.Enabled["test"])

	config.SetTypeEnabled("test", false)
	assert.False(t, config.Enabled["test"])
}

func TestNotifyConfigClone(t *testing.T) {
	original := DefaultNotifyConfig()
	clone := original.Clone()

	// Verify deep copy
	assert.Equal(t, original.IdleAfter, clone.IdleAfter)
	assert.Equal(t, original.GoalMilestones, clone.GoalMilestones)
	assert.Equal(t, original.Enabled, clone.Enabled)

	// Modify clone - shouldn't affect original
	clone.IdleAfter = 1 * time.Hour
	clone.GoalMilestones[0] = 25
	clone.Enabled["idle"] = false

	assert.NotEqual(t, original.IdleAfter, clone.IdleAfter)
	assert.NotEqual(t, original.GoalMilestones[0], clone.GoalMilestones[0])
	assert.NotEqual(t, original.Enabled["idle"], clone.Enabled["idle"])
}

func TestNotifyConfigCloneNilFields(t *testing.T) {
	original := &NotifyConfig{
		IdleAfter: 10 * time.Minute,
	}
	clone := original.Clone()

	assert.Equal(t, original.IdleAfter, clone.IdleAfter)
	assert.Nil(t, clone.GoalMilestones)
	assert.Nil(t, clone.Enabled)
}

func TestNotifyConfigValidate(t *testing.T) {
	t.Run("valid_config", func(t *testing.T) {
		config := DefaultNotifyConfig()
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("idle_too_short", func(t *testing.T) {
		config := DefaultNotifyConfig()
		config.IdleAfter = 1 * time.Minute
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "idle_after")
	})

	t.Run("idle_too_long", func(t *testing.T) {
		config := DefaultNotifyConfig()
		config.IdleAfter = 5 * time.Hour
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "idle_after")
	})

	t.Run("break_disabled_valid", func(t *testing.T) {
		config := DefaultNotifyConfig()
		config.BreakAfter = 0
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("break_too_short", func(t *testing.T) {
		config := DefaultNotifyConfig()
		config.BreakAfter = 10 * time.Minute
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "break_after")
	})

	t.Run("break_too_long", func(t *testing.T) {
		config := DefaultNotifyConfig()
		config.BreakAfter = 10 * time.Hour
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "break_after")
	})

	t.Run("milestone_too_low", func(t *testing.T) {
		config := DefaultNotifyConfig()
		config.GoalMilestones = []int{0, 50, 100}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "milestone")
	})

	t.Run("milestone_too_high", func(t *testing.T) {
		config := DefaultNotifyConfig()
		config.GoalMilestones = []int{50, 101}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "milestone")
	})
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "test_field", Message: "test message"}
	assert.Equal(t, "test_field: test message", err.Error())
}

// =============================================================================
// Model Interface Tests
// =============================================================================

func TestModelInterface(t *testing.T) {
	// Verify all types implement Model interface
	var _ Model = &Block{}
	var _ Model = &Project{}
	var _ Model = &Task{}
	var _ Model = &Goal{}
	var _ Model = &Reminder{}
	var _ Model = &Webhook{}
	var _ Model = &ActiveBlock{}
	var _ Model = &UndoState{}
	var _ Model = &Config{}
}

// =============================================================================
// Key Prefix Constants Tests
// =============================================================================

func TestKeyPrefixConstants(t *testing.T) {
	assert.Equal(t, "block", PrefixBlock)
	assert.Equal(t, "project", PrefixProject)
	assert.Equal(t, "task", PrefixTask)
	assert.Equal(t, "goal", PrefixGoal)
	assert.Equal(t, "activeblock", KeyActiveBlock)
	assert.Equal(t, "config", KeyConfig)
	assert.Equal(t, "reminder", PrefixReminder)
	assert.Equal(t, "webhook", PrefixWebhook)
	assert.Equal(t, "config:notify", KeyNotifyConfig)
	assert.Equal(t, "undo", KeyUndo)
}
