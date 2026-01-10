package output

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Formatter Tests
// =============================================================================

func TestNewFormatter(t *testing.T) {
	f := NewFormatter()
	assert.NotNil(t, f)
	assert.Equal(t, FormatCLI, f.Format)
	assert.Equal(t, ColorAuto, f.ColorMode)
	assert.False(t, f.NoNewline)
}

func TestFormatterIsColorEnabled(t *testing.T) {
	t.Run("color_always", func(t *testing.T) {
		f := &Formatter{ColorMode: ColorAlways}
		assert.True(t, f.IsColorEnabled())
	})

	t.Run("color_never", func(t *testing.T) {
		f := &Formatter{ColorMode: ColorNever}
		assert.False(t, f.IsColorEnabled())
	})

	t.Run("color_auto_non_terminal", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{
			Writer:    &buf,
			ColorMode: ColorAuto,
		}
		// Buffer is not a terminal
		assert.False(t, f.IsColorEnabled())
	})
}

func TestFormatterPrint(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf}

	f.Print("hello")
	assert.Equal(t, "hello", buf.String())
}

func TestFormatterPrintln(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf}

	f.Println("hello")
	assert.Equal(t, "hello\n", buf.String())
}

func TestFormatterPrintf(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf}

	f.Printf("hello %s", "world")
	assert.Equal(t, "hello world", buf.String())
}

func TestFormatterJSON(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf}

	data := map[string]string{"key": "value"}
	err := f.JSON(data)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `"key": "value"`)
}

func TestFormatterPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf}

	data := map[string]int{"count": 42}
	err := f.PrintJSON(data)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `"count": 42`)
}

// =============================================================================
// Format and ColorMode Constants Tests
// =============================================================================

func TestFormatConstants(t *testing.T) {
	assert.Equal(t, Format("cli"), FormatCLI)
	assert.Equal(t, Format("json"), FormatJSON)
	assert.Equal(t, Format("plain"), FormatPlain)
}

func TestColorModeConstants(t *testing.T) {
	assert.Equal(t, ColorMode("auto"), ColorAuto)
	assert.Equal(t, ColorMode("always"), ColorAlways)
	assert.Equal(t, ColorMode("never"), ColorNever)
}

// =============================================================================
// Duration Formatting Tests
// =============================================================================

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m"},
		{90 * time.Second, "1m 30s"},
		{5 * time.Minute, "5m"},
		{5*time.Minute + 30*time.Second, "5m 30s"},
		{59 * time.Minute, "59m"},
		{60 * time.Minute, "1h"},
		{90 * time.Minute, "1h 30m"},
		{2 * time.Hour, "2h"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
		{8*time.Hour + 30*time.Minute, "8h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{60 * time.Second, "1m"},
		{90 * time.Second, "1m"}, // No seconds in short form
		{5 * time.Minute, "5m"},
		{60 * time.Minute, "1h"},
		{90 * time.Minute, "1h 30m"},
		{2 * time.Hour, "2h"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDurationShort(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Time Formatting Tests
// =============================================================================

func TestFormatTime(t *testing.T) {
	tm := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	result := FormatTime(tm)
	assert.Contains(t, result, "2024-01-15")
	assert.Contains(t, result, "30")
	assert.Contains(t, result, "45")
}

func TestFormatTimeShort(t *testing.T) {
	tm := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	result := FormatTimeShort(tm)
	assert.Contains(t, result, "2024-01-15")
	assert.Contains(t, result, "30")
	assert.NotContains(t, result, ":45")
}

func TestFormatDate(t *testing.T) {
	tm := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	result := FormatDate(tm)
	assert.Contains(t, result, "2024-01-15")
	assert.NotContains(t, result, "14")
}

func TestFormatTimeOnly(t *testing.T) {
	tm := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	result := FormatTimeOnly(tm)
	assert.NotContains(t, result, "2024")
	assert.Contains(t, result, ":")
}

// =============================================================================
// CLIFormatter Tests
// =============================================================================

func TestNewCLIFormatter(t *testing.T) {
	f := NewFormatter()
	cli := NewCLIFormatter(f)
	assert.NotNil(t, cli)
	assert.Equal(t, f, cli.Formatter)
}

func TestCLIFormatterTitle(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	cli.Title("My Title")
	assert.Contains(t, buf.String(), "My Title")
}

func TestCLIFormatterSuccess(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	cli.Success("Operation completed")
	assert.Contains(t, buf.String(), "✓ Operation completed")
}

func TestCLIFormatterWarning(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	cli.Warning("Be careful")
	assert.Contains(t, buf.String(), "⚠ Be careful")
}

func TestCLIFormatterError(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	cli.Error("Something failed")
	assert.Contains(t, buf.String(), "✗ Something failed")
}

func TestCLIFormatterMuted(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	cli.Muted("Subtle text")
	assert.Contains(t, buf.String(), "Subtle text")
}

func TestCLIFormatterProjectName(t *testing.T) {
	t.Run("no_color", func(t *testing.T) {
		f := &Formatter{ColorMode: ColorNever}
		cli := NewCLIFormatter(f)
		result := cli.ProjectName("myproject")
		assert.Equal(t, "myproject", result)
	})

	t.Run("with_color", func(t *testing.T) {
		f := &Formatter{ColorMode: ColorAlways}
		cli := NewCLIFormatter(f)
		result := cli.ProjectName("myproject")
		// With color enabled, result should contain the text
		assert.Contains(t, result, "myproject")
	})
}

func TestCLIFormatterTaskName(t *testing.T) {
	f := &Formatter{ColorMode: ColorNever}
	cli := NewCLIFormatter(f)
	result := cli.TaskName("mytask")
	assert.Equal(t, "mytask", result)
}

func TestCLIFormatterDuration(t *testing.T) {
	f := &Formatter{ColorMode: ColorNever}
	cli := NewCLIFormatter(f)
	result := cli.Duration("2h 30m")
	assert.Equal(t, "2h 30m", result)
}

func TestCLIFormatterNote(t *testing.T) {
	f := &Formatter{ColorMode: ColorNever}
	cli := NewCLIFormatter(f)
	result := cli.Note("Some note")
	assert.Equal(t, "Some note", result)
}

func TestCLIFormatterFormatProjectTask(t *testing.T) {
	f := &Formatter{ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	t.Run("project_only", func(t *testing.T) {
		result := cli.FormatProjectTask("myproject", "")
		assert.Equal(t, "myproject", result)
	})

	t.Run("project_and_task", func(t *testing.T) {
		result := cli.FormatProjectTask("myproject", "mytask")
		assert.Equal(t, "myproject/mytask", result)
	})
}

func TestCLIFormatterPrintTrackingStarted(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	block := &model.Block{
		ProjectSID:     "myproject",
		TaskSID:        "mytask",
		Note:           "Working on feature",
		TimestampStart: time.Now(),
	}

	cli.PrintTrackingStarted(block)
	output := buf.String()

	assert.Contains(t, output, "Started tracking")
	assert.Contains(t, output, "myproject")
	assert.Contains(t, output, "Note:")
	assert.Contains(t, output, "Working on feature")
}

func TestCLIFormatterPrintTrackingStartedWithEnd(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	block := &model.Block{
		ProjectSID:     "myproject",
		TimestampStart: time.Now().Add(-1 * time.Hour),
		TimestampEnd:   time.Now(),
	}

	cli.PrintTrackingStarted(block)
	output := buf.String()

	assert.Contains(t, output, "Ended:")
	assert.Contains(t, output, "Duration:")
}

func TestCLIFormatterPrintTrackingStopped(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	block := &model.Block{
		ProjectSID:     "myproject",
		Note:           "Done",
		TimestampStart: time.Now().Add(-30 * time.Minute),
		TimestampEnd:   time.Now(),
	}

	cli.PrintTrackingStopped(block)
	output := buf.String()

	assert.Contains(t, output, "Stopped tracking")
	assert.Contains(t, output, "Duration:")
	assert.Contains(t, output, "Done")
}

func TestCLIFormatterPrintStatus(t *testing.T) {
	t.Run("with_active_block", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{Writer: &buf, ColorMode: ColorNever}
		cli := NewCLIFormatter(f)

		block := &model.Block{
			ProjectSID:     "myproject",
			Note:           "Working",
			TimestampStart: time.Now().Add(-15 * time.Minute),
		}

		cli.PrintStatus(block)
		output := buf.String()

		assert.Contains(t, output, "Currently tracking")
		assert.Contains(t, output, "myproject")
	})

	t.Run("no_active_block", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{Writer: &buf, ColorMode: ColorNever}
		cli := NewCLIFormatter(f)

		cli.PrintStatus(nil)
		output := buf.String()

		assert.Contains(t, output, "No active tracking")
	})
}

func TestCLIFormatterPrintNoActiveTracking(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf, ColorMode: ColorNever}
	cli := NewCLIFormatter(f)

	cli.PrintNoActiveTracking()
	output := buf.String()

	assert.Contains(t, output, "No active tracking")
	assert.Contains(t, output, "humantime start")
}

// =============================================================================
// ProgressBar Tests
// =============================================================================

func TestProgressBar(t *testing.T) {
	tests := []struct {
		percentage float64
		width      int
	}{
		{0, 10},
		{50, 10},
		{100, 10},
		{150, 10}, // Over 100%
		{-10, 10}, // Negative
		{75, 20},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			bar := ProgressBar(tt.percentage, tt.width)
			assert.Equal(t, tt.width, len([]rune(bar)))
		})
	}
}

func TestProgressBarContent(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		bar := ProgressBar(0, 10)
		assert.Equal(t, "░░░░░░░░░░", bar)
	})

	t.Run("half", func(t *testing.T) {
		bar := ProgressBar(50, 10)
		assert.Equal(t, "█████░░░░░", bar)
	})

	t.Run("full", func(t *testing.T) {
		bar := ProgressBar(100, 10)
		assert.Equal(t, "██████████", bar)
	})
}

// =============================================================================
// Table Tests
// =============================================================================

func TestCLIFormatterPrintTable(t *testing.T) {
	t.Run("with_rows", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{Writer: &buf, ColorMode: ColorNever}
		cli := NewCLIFormatter(f)

		headers := []string{"Name", "Duration"}
		rows := []TableRow{
			{Columns: []string{"project1", "2h"}},
			{Columns: []string{"project2", "30m"}},
		}

		cli.PrintTable(headers, rows)
		output := buf.String()

		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "Duration")
		assert.Contains(t, output, "project1")
		assert.Contains(t, output, "project2")
		assert.Contains(t, output, "─")
	})

	t.Run("empty_rows", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{Writer: &buf, ColorMode: ColorNever}
		cli := NewCLIFormatter(f)

		cli.PrintTable([]string{"Name"}, []TableRow{})
		assert.Empty(t, buf.String())
	})
}

// =============================================================================
// JSONFormatter Tests
// =============================================================================

func TestNewJSONFormatter(t *testing.T) {
	f := NewFormatter()
	jf := NewJSONFormatter(f)
	assert.NotNil(t, jf)
	assert.Equal(t, f, jf.Formatter)
}

func TestNewBlockOutput(t *testing.T) {
	block := &model.Block{
		Key:            "block:123",
		ProjectSID:     "myproject",
		TaskSID:        "mytask",
		Note:           "Working",
		Tags:           []string{"urgent"},
		TimestampStart: time.Now().Add(-1 * time.Hour),
		TimestampEnd:   time.Now(),
	}

	out := NewBlockOutput(block)

	assert.Equal(t, "block:123", out.Key)
	assert.Equal(t, "myproject", out.ProjectSID)
	assert.Equal(t, "mytask", out.TaskSID)
	assert.Equal(t, "Working", out.Note)
	assert.Equal(t, []string{"urgent"}, out.Tags)
	assert.NotEmpty(t, out.TimestampStart)
	assert.NotEmpty(t, out.TimestampEnd)
	assert.False(t, out.IsActive)
	assert.InDelta(t, 3600, out.DurationSeconds, 1)
}

func TestNewBlockOutputActive(t *testing.T) {
	block := &model.Block{
		Key:            "block:123",
		ProjectSID:     "myproject",
		TimestampStart: time.Now().Add(-30 * time.Minute),
	}

	out := NewBlockOutput(block)

	assert.True(t, out.IsActive)
	assert.Empty(t, out.TimestampEnd)
}

func TestNewBlocksResponse(t *testing.T) {
	blocks := []*model.Block{
		{
			Key:            "block:1",
			ProjectSID:     "proj1",
			TimestampStart: time.Now().Add(-2 * time.Hour),
			TimestampEnd:   time.Now().Add(-1 * time.Hour),
		},
		{
			Key:            "block:2",
			ProjectSID:     "proj2",
			TimestampStart: time.Now().Add(-1 * time.Hour),
			TimestampEnd:   time.Now(),
		},
	}

	resp := NewBlocksResponse(blocks, 10)

	assert.Equal(t, 2, len(resp.Blocks))
	assert.Equal(t, 10, resp.TotalCount)
	assert.Equal(t, 2, resp.ShownCount)
	assert.InDelta(t, 7200, resp.TotalDurationSeconds, 2)
}

func TestNewProjectOutput(t *testing.T) {
	project := &model.Project{
		SID:         "myproject",
		DisplayName: "My Project",
		Color:       "#FF5733",
	}

	out := NewProjectOutput(project, 2*time.Hour)

	assert.Equal(t, "myproject", out.SID)
	assert.Equal(t, "My Project", out.DisplayName)
	assert.Equal(t, "#FF5733", out.Color)
	assert.Equal(t, int64(7200), out.TotalDurationSeconds)
}

func TestNewTaskOutput(t *testing.T) {
	task := &model.Task{
		SID:         "mytask",
		ProjectSID:  "myproject",
		DisplayName: "My Task",
		Color:       "#00FF00",
	}

	out := NewTaskOutput(task, 30*time.Minute)

	assert.Equal(t, "mytask", out.SID)
	assert.Equal(t, "myproject", out.ProjectSID)
	assert.Equal(t, "My Task", out.DisplayName)
	assert.Equal(t, "#00FF00", out.Color)
	assert.Equal(t, int64(1800), out.TotalDurationSeconds)
}

func TestJSONFormatterPrintStatus(t *testing.T) {
	t.Run("idle", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{Writer: &buf}
		jf := NewJSONFormatter(f)

		err := jf.PrintStatus(nil)
		require.NoError(t, err)

		var resp StatusResponse
		err = json.Unmarshal(buf.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "idle", resp.Status)
		assert.Nil(t, resp.ActiveBlock)
	})

	t.Run("tracking", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{Writer: &buf}
		jf := NewJSONFormatter(f)

		block := &model.Block{
			ProjectSID:     "myproject",
			TimestampStart: time.Now(),
		}

		err := jf.PrintStatus(block)
		require.NoError(t, err)

		var resp StatusResponse
		err = json.Unmarshal(buf.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "tracking", resp.Status)
		assert.NotNil(t, resp.ActiveBlock)
	})
}

func TestJSONFormatterPrintStart(t *testing.T) {
	t.Run("without_previous", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{Writer: &buf}
		jf := NewJSONFormatter(f)

		block := &model.Block{
			ProjectSID:     "myproject",
			TimestampStart: time.Now(),
		}

		err := jf.PrintStart(block, nil)
		require.NoError(t, err)

		var resp StartResponse
		err = json.Unmarshal(buf.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "started", resp.Status)
		assert.NotNil(t, resp.Block)
		assert.Nil(t, resp.PreviousBlock)
	})

	t.Run("with_previous", func(t *testing.T) {
		var buf bytes.Buffer
		f := &Formatter{Writer: &buf}
		jf := NewJSONFormatter(f)

		block := &model.Block{
			ProjectSID:     "newproject",
			TimestampStart: time.Now(),
		}
		previous := &model.Block{
			ProjectSID:     "oldproject",
			TimestampStart: time.Now().Add(-1 * time.Hour),
			TimestampEnd:   time.Now(),
		}

		err := jf.PrintStart(block, previous)
		require.NoError(t, err)

		var resp StartResponse
		err = json.Unmarshal(buf.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "started", resp.Status)
		assert.NotNil(t, resp.Block)
		assert.NotNil(t, resp.PreviousBlock)
	})
}

func TestJSONFormatterPrintStop(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf}
	jf := NewJSONFormatter(f)

	block := &model.Block{
		ProjectSID:     "myproject",
		TimestampStart: time.Now().Add(-1 * time.Hour),
		TimestampEnd:   time.Now(),
	}

	err := jf.PrintStop(block)
	require.NoError(t, err)

	var resp StopResponse
	err = json.Unmarshal(buf.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "stopped", resp.Status)
	assert.NotNil(t, resp.Block)
}

func TestJSONFormatterPrintError(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf}
	jf := NewJSONFormatter(f)

	err := jf.PrintError("error", "something failed", "Please try again")
	require.NoError(t, err)

	var resp ErrorResponse
	err = json.Unmarshal(buf.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "something failed", resp.Error)
	assert.Equal(t, "Please try again", resp.Message)
}

func TestJSONFormatterPrintBlocks(t *testing.T) {
	var buf bytes.Buffer
	f := &Formatter{Writer: &buf}
	jf := NewJSONFormatter(f)

	blocks := []*model.Block{
		{
			ProjectSID:     "proj1",
			TimestampStart: time.Now().Add(-1 * time.Hour),
			TimestampEnd:   time.Now(),
		},
	}

	err := jf.PrintBlocks(blocks, 5)
	require.NoError(t, err)

	var resp BlocksResponse
	err = json.Unmarshal(buf.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, 1, len(resp.Blocks))
	assert.Equal(t, 5, resp.TotalCount)
	assert.Equal(t, 1, resp.ShownCount)
}

// =============================================================================
// JSON Output Struct Tests
// =============================================================================

func TestStatusResponseStruct(t *testing.T) {
	resp := StatusResponse{
		Status:      "tracking",
		ActiveBlock: &BlockOutput{ProjectSID: "myproject"},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded StatusResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "tracking", decoded.Status)
	assert.NotNil(t, decoded.ActiveBlock)
}

func TestBlocksResponseStruct(t *testing.T) {
	resp := BlocksResponse{
		Blocks:               []*BlockOutput{{ProjectSID: "proj1"}},
		TotalCount:           10,
		ShownCount:           1,
		TotalDurationSeconds: 3600,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded BlocksResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 10, decoded.TotalCount)
	assert.Equal(t, int64(3600), decoded.TotalDurationSeconds)
}

func TestProjectsResponseStruct(t *testing.T) {
	resp := ProjectsResponse{
		Projects: []*ProjectOutput{
			{SID: "proj1", DisplayName: "Project 1"},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ProjectsResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 1, len(decoded.Projects))
	assert.Equal(t, "proj1", decoded.Projects[0].SID)
}

func TestStatsResponseStruct(t *testing.T) {
	resp := StatsResponse{
		Period: &PeriodOutput{
			Start:    "2024-01-01",
			End:      "2024-01-31",
			Grouping: "day",
		},
		Summary: &SummaryOutput{
			TotalDurationSeconds: 36000,
		},
		Groups: []*GroupOutput{
			{Label: "Mon", DurationSeconds: 7200},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded StatsResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.NotNil(t, decoded.Period)
	assert.Equal(t, "day", decoded.Period.Grouping)
	assert.NotNil(t, decoded.Summary)
	assert.Equal(t, 1, len(decoded.Groups))
}

func TestProjectSummaryOutputStruct(t *testing.T) {
	summary := ProjectSummaryOutput{
		ProjectSID:      "myproject",
		DurationSeconds: 7200,
		Percentage:      45.5,
		ByTask: []*TaskSummaryOutput{
			{TaskSID: "task1", DurationSeconds: 3600},
		},
	}

	data, err := json.Marshal(summary)
	require.NoError(t, err)

	var decoded ProjectSummaryOutput
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "myproject", decoded.ProjectSID)
	assert.Equal(t, 45.5, decoded.Percentage)
	assert.Equal(t, 1, len(decoded.ByTask))
}
