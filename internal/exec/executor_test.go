package exec

import (
	"testing"

	"github.com/cshaiku/goshi/internal/repair"
)

// TestExecutorDryRunEmpty tests dry-run with empty plan
func TestExecutorDryRunEmpty(t *testing.T) {
	executor := &Executor{DryRun: true}
	plan := repair.Plan{
		Actions: []repair.Action{},
	}

	err := executor.Execute(plan)
	if err != nil {
		t.Errorf("expected empty plan to succeed, got error: %v", err)
	}
}

// TestExecutorDryRunEcho tests dry-run with echo command
func TestExecutorDryRunEcho(t *testing.T) {
	executor := &Executor{DryRun: true}
	plan := repair.Plan{
		Actions: []repair.Action{
			{
				Code:        "test_echo",
				Description: "echo test",
				Command:     []string{"echo", "hello"},
			},
		},
	}

	err := executor.Execute(plan)
	if err != nil {
		t.Errorf("expected dry-run to succeed, got error: %v", err)
	}
}

// TestExecutorDryRunMultipleActions tests dry-run with multiple actions
func TestExecutorDryRunMultipleActions(t *testing.T) {
	executor := &Executor{DryRun: true}
	plan := repair.Plan{
		Actions: []repair.Action{
			{
				Code:        "action1",
				Description: "first action",
				Command:     []string{"echo", "action1"},
			},
			{
				Code:        "action2",
				Description: "second action",
				Command:     []string{"echo", "action2"},
			},
		},
	}

	err := executor.Execute(plan)
	if err != nil {
		t.Errorf("expected multiple dry-run actions to succeed, got error: %v", err)
	}
}

// TestExecutorExecuteEcho tests actual execution with echo command
func TestExecutorExecuteEcho(t *testing.T) {
	executor := &Executor{DryRun: false}
	plan := repair.Plan{
		Actions: []repair.Action{
			{
				Code:        "test_echo",
				Description: "echo test",
				Command:     []string{"echo", "test"},
			},
		},
	}

	err := executor.Execute(plan)
	if err != nil {
		t.Errorf("expected echo command to succeed, got error: %v", err)
	}
}

// TestExecutorExecuteInvalidCommand tests execution with non-existent command
func TestExecutorExecuteInvalidCommand(t *testing.T) {
	executor := &Executor{DryRun: false}
	plan := repair.Plan{
		Actions: []repair.Action{
			{
				Code:        "invalid",
				Description: "invalid command",
				Command:     []string{"nonexistent_command_xyz_123", "arg"},
			},
		},
	}

	err := executor.Execute(plan)
	if err == nil {
		t.Errorf("expected invalid command to fail, got no error")
	}
}

// TestExecutorExecuteFailingCommand tests execution with command that returns error
func TestExecutorExecuteFailingCommand(t *testing.T) {
	executor := &Executor{DryRun: false}
	plan := repair.Plan{
		Actions: []repair.Action{
			{
				Code:        "failing",
				Description: "failing command",
				Command:     []string{"sh", "-c", "exit 1"},
			},
		},
	}

	err := executor.Execute(plan)
	if err == nil {
		t.Errorf("expected failing command to return error, got no error")
	}
}

// TestExecutorDryRunNeverFailsEvenWithInvalidCommand tests that dry-run never fails
func TestExecutorDryRunNeverFailsEvenWithInvalidCommand(t *testing.T) {
	executor := &Executor{DryRun: true}
	plan := repair.Plan{
		Actions: []repair.Action{
			{
				Code:        "invalid",
				Description: "invalid command that would fail",
				Command:     []string{"nonexistent_command_xyz_123", "arg"},
			},
		},
	}

	err := executor.Execute(plan)
	if err != nil {
		t.Errorf("expected dry-run with invalid command to still succeed, got error: %v", err)
	}
}

// TestExecutorIsNotDryRun tests the inverse - actual execution mode
func TestExecutorIsNotDryRun(t *testing.T) {
	executor := &Executor{DryRun: false}

	if executor.DryRun {
		t.Errorf("expected DryRun to be false, got true")
	}
}

// TestExecutorActionStructure tests that plan action structure is preserved
func TestExecutorActionStructure(t *testing.T) {
	executor := &Executor{DryRun: true}
	plan := repair.Plan{
		Actions: []repair.Action{
			{
				Code:        "test_code",
				Description: "test description",
				Command:     []string{"echo", "test"},
			},
		},
	}

	err := executor.Execute(plan)
	if err != nil {
		t.Errorf("expected execution to succeed, got error: %v", err)
	}

	// Verify the plan wasn't modified
	if plan.Actions[0].Code != "test_code" {
		t.Errorf("expected Code to remain 'test_code'")
	}
	if plan.Actions[0].Description != "test description" {
		t.Errorf("expected Description to remain 'test description'")
	}
}
