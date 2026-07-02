package agent

import (
	"testing"

	"github.com/Gitlawb/zero/internal/modelregistry"
	"github.com/Gitlawb/zero/internal/zeroruntime"
)

func TestCostTracker_AddUsage(t *testing.T) {
	registry, err := modelregistry.DefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("tracks input and output tokens", func(t *testing.T) {
		ct := NewCostTracker(nil, "gpt-4", 0)
		ct.AddUsage(zeroruntime.Usage{InputTokens: 100, OutputTokens: 50})
		if got := ct.InputTokens(); got != 100 {
			t.Errorf("expected 100 input tokens, got %d", got)
		}
		if got := ct.OutputTokens(); got != 50 {
			t.Errorf("expected 50 output tokens, got %d", got)
		}
	})

	t.Run("accumulates multiple calls", func(t *testing.T) {
		ct := NewCostTracker(nil, "gpt-4", 0)
		ct.AddUsage(zeroruntime.Usage{InputTokens: 100, OutputTokens: 50})
		ct.AddUsage(zeroruntime.Usage{InputTokens: 200, OutputTokens: 100})
		if got := ct.InputTokens(); got != 300 {
			t.Errorf("expected 300 input tokens, got %d", got)
		}
		if got := ct.OutputTokens(); got != 150 {
			t.Errorf("expected 150 output tokens, got %d", got)
		}
	})

	t.Run("computes cost with registry", func(t *testing.T) {
		ct := NewCostTracker(&registry, "gpt-4.1", 0)
		ct.AddUsage(zeroruntime.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000})
		cost := ct.TotalCost()
		if cost <= 0 {
			t.Errorf("expected positive cost, got %f", cost)
		}
		// gpt-4.1: $2/M input + $8/M output
		if cost < 9.9 || cost > 10.1 {
			t.Errorf("expected cost ~$10, got $%.4f", cost)
		}
	})

	t.Run("handles usage with PromptTokens fallback", func(t *testing.T) {
		ct := NewCostTracker(nil, "gpt-4", 0)
		ct.AddUsage(zeroruntime.Usage{PromptTokens: 500, CompletionTokens: 300})
		if got := ct.InputTokens(); got != 500 {
			t.Errorf("expected 500 input tokens, got %d", got)
		}
		if got := ct.OutputTokens(); got != 300 {
			t.Errorf("expected 300 output tokens, got %d", got)
		}
	})
}

func TestCostTracker_BudgetExceeded(t *testing.T) {
	t.Run("not exceeded when no max cost set", func(t *testing.T) {
		ct := NewCostTracker(nil, "gpt-4", 0)
		if ct.BudgetExceeded() {
			t.Error("expected budget not exceeded when max cost is 0")
		}
	})

	t.Run("exceeded when cost reaches max", func(t *testing.T) {
		registry, err := modelregistry.DefaultRegistry()
		if err != nil {
			t.Fatal(err)
		}
		ct := NewCostTracker(&registry, "gpt-4.1-nano", 0.001)
		ct.AddUsage(zeroruntime.Usage{InputTokens: 10_000, OutputTokens: 10_000})
		if !ct.BudgetExceeded() {
			t.Error("expected budget to be exceeded")
		}
	})

	t.Run("not exceeded when cost is under max", func(t *testing.T) {
		registry, err := modelregistry.DefaultRegistry()
		if err != nil {
			t.Fatal(err)
		}
		ct := NewCostTracker(&registry, "gpt-4.1-nano", 100)
		ct.AddUsage(zeroruntime.Usage{InputTokens: 100, OutputTokens: 100})
		if ct.BudgetExceeded() {
			t.Error("expected budget not exceeded")
		}
	})
}

func TestCostTracker_TotalCost(t *testing.T) {
	t.Run("zero cost with no usage", func(t *testing.T) {
		ct := NewCostTracker(nil, "gpt-4", 0)
		if cost := ct.TotalCost(); cost != 0 {
			t.Errorf("expected zero cost, got %f", cost)
		}
	})

	t.Run("returns max cost value", func(t *testing.T) {
		ct := NewCostTracker(nil, "gpt-4", 5.50)
		if got := ct.MaxCost(); got != 5.50 {
			t.Errorf("expected max cost 5.50, got %f", got)
		}
	})
}

func TestCostTracker_NoRegistry(t *testing.T) {
	ct := NewCostTracker(nil, "gpt-4", 10)
	ct.AddUsage(zeroruntime.Usage{InputTokens: 100, OutputTokens: 50})
	cost := ct.TotalCost()
	if cost != 0 {
		t.Errorf("expected zero cost without registry, got %f", cost)
	}
}
