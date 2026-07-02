package agent

import (
	"sync"

	"github.com/Gitlawb/zero/internal/modelregistry"
	"github.com/Gitlawb/zero/internal/zeroruntime"
)

type CostTracker struct {
	mu          sync.Mutex
	registry    *modelregistry.Registry
	modelID     string
	totalCost   float64
	maxCost     float64
	inputTokens int
	outputTokens int
}

func NewCostTracker(registry *modelregistry.Registry, modelID string, maxCost float64) *CostTracker {
	return &CostTracker{
		registry: registry,
		modelID:  modelID,
		maxCost:  maxCost,
	}
}

func (ct *CostTracker) AddUsage(usage zeroruntime.Usage) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.inputTokens += usage.EffectiveInputTokens()
	ct.outputTokens += usage.EffectiveOutputTokens()

	if ct.registry != nil {
		breakdown, err := ct.registry.EstimateCost(ct.modelID, usage)
		if err == nil {
			ct.totalCost += breakdown.TotalCost
		}
	}
}

func (ct *CostTracker) TotalCost() float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.totalCost
}

func (ct *CostTracker) InputTokens() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.inputTokens
}

func (ct *CostTracker) OutputTokens() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.outputTokens
}

func (ct *CostTracker) BudgetExceeded() bool {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.maxCost > 0 && ct.totalCost >= ct.maxCost
}

func (ct *CostTracker) MaxCost() float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.maxCost
}
