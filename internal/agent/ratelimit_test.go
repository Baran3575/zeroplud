package agent

import (
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	t.Run("allows up to capacity", func(t *testing.T) {
		tb := NewTokenBucket(3, time.Minute)
		if !tb.Allow() {
			t.Error("expected first call to allow")
		}
		if !tb.Allow() {
			t.Error("expected second call to allow")
		}
		if !tb.Allow() {
			t.Error("expected third call to allow")
		}
		if tb.Allow() {
			t.Error("expected fourth call to be denied (empty bucket)")
		}
	})

	t.Run("refills over time", func(t *testing.T) {
		tb := NewTokenBucket(60, time.Minute)
		for range 60 {
			tb.Allow()
		}
		if tb.Allow() {
			t.Fatal("expected bucket to be empty")
		}
		tb.lastCheck = time.Now().Add(-time.Minute)
		if !tb.Allow() {
			t.Error("expected bucket to have refilled after one minute")
		}
	})

	t.Run("never exceeds capacity", func(t *testing.T) {
		tb := NewTokenBucket(5, time.Minute)
		tb.lastCheck = time.Now().Add(-2 * time.Minute)
		for range 10 {
			tb.Allow()
		}
		if tb.tokens > 5 {
			t.Errorf("tokens exceeded capacity: %d > 5", tb.tokens)
		}
	})

	t.Run("zero capacity defaults to 1", func(t *testing.T) {
		tb := NewTokenBucket(0, time.Minute)
		if tb.capacity != 1 {
			t.Errorf("expected capacity 1, got %d", tb.capacity)
		}
	})

	t.Run("negative rate defaults to 1", func(t *testing.T) {
		tb := NewTokenBucket(-1, time.Minute)
		if tb.capacity != 1 {
			t.Errorf("expected capacity 1, got %d", tb.capacity)
		}
	})
}

func TestTokenBucket_WaitTime(t *testing.T) {
	t.Run("zero wait when tokens available", func(t *testing.T) {
		tb := NewTokenBucket(10, time.Minute)
		if wt := tb.WaitTime(); wt != 0 {
			t.Errorf("expected zero wait, got %v", wt)
		}
	})

	t.Run("positive wait when empty", func(t *testing.T) {
		tb := NewTokenBucket(1, time.Minute)
		tb.Allow()
		if wt := tb.WaitTime(); wt <= 0 {
			t.Error("expected positive wait time when bucket empty")
		}
	})
}

func TestRateLimiter_Config(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetConfig("gpt-4", 10, 1000)

	t.Run("allows within limits", func(t *testing.T) {
		if !rl.Allow("gpt-4", 100) {
			t.Error("expected allowance within limits")
		}
	})

	t.Run("denies when RPM exhausted", func(t *testing.T) {
		sm := NewRateLimiter()
		sm.SetConfig("tiny", 1, 100000)
		sm.Allow("tiny", 0)
		if sm.Allow("tiny", 0) {
			t.Error("expected denial when RPM exhausted")
		}
	})
}

func TestRateLimiter_WaitTime(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetConfig("slow-model", 5, 500)

	t.Run("no wait when within limits", func(t *testing.T) {
		if wt := rl.WaitTime("slow-model", 10); wt != 0 {
			t.Errorf("expected zero wait, got %v", wt)
		}
	})

	t.Run("unconfigured model has no wait", func(t *testing.T) {
		if wt := rl.WaitTime("unknown", 100); wt != 0 {
			t.Errorf("expected zero wait for unconfigured model, got %v", wt)
		}
	})
}

func TestRateLimiter_NoConfig(t *testing.T) {
	rl := NewRateLimiter()
	if !rl.Allow("any-model", 999999) {
		t.Error("expected all calls to be allowed when no config set")
	}
}
