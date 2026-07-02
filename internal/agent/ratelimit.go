package agent

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu        sync.Mutex
	capacity  int
	tokens    int
	interval  time.Duration
	lastCheck time.Time
}

func NewTokenBucket(rate int, interval time.Duration) *TokenBucket {
	capacity := rate
	if capacity <= 0 {
		capacity = 1
	}
	return &TokenBucket{
		capacity:  capacity,
		tokens:    capacity,
		interval:  interval,
		lastCheck: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastCheck)
	tb.lastCheck = now

	refill := int(elapsed.Nanoseconds()) * tb.capacity / int(tb.interval.Nanoseconds())
	if refill > 0 {
		tb.tokens += refill
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
	}

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

func (tb *TokenBucket) WaitTime() time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if tb.tokens > 0 {
		return 0
	}
	elapsed := time.Since(tb.lastCheck)
	wait := tb.interval - elapsed
	if wait < 0 {
		return 0
	}
	return wait
}

type RateLimiter struct {
	mu   sync.Mutex
	rpm  map[string]*TokenBucket
	tpm  map[string]*TokenBucket
	conf map[string]RPMTPMConfig
}

type RPMTPMConfig struct {
	RPM int
	TPM int
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		rpm:  make(map[string]*TokenBucket),
		tpm:  make(map[string]*TokenBucket),
		conf: make(map[string]RPMTPMConfig),
	}
}

func (rl *RateLimiter) SetConfig(modelID string, rpm, tpm int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.conf[modelID] = RPMTPMConfig{RPM: rpm, TPM: tpm}
	if rpm > 0 {
		rl.rpm[modelID] = NewTokenBucket(rpm, time.Minute)
	} else {
		delete(rl.rpm, modelID)
	}
	if tpm > 0 {
		rl.tpm[modelID] = NewTokenBucket(tpm, time.Minute)
	} else {
		delete(rl.tpm, modelID)
	}
}

func (rl *RateLimiter) Allow(modelID string, tokens int) bool {
	rpmBucket, tpmBucket := rl.bucketsFor(modelID)
	if rpmBucket == nil && tpmBucket == nil {
		return true
	}
	if rpmBucket != nil && !rpmBucket.Allow() {
		return false
	}
	if tpmBucket != nil {
		needed := tokens
		if needed <= 0 {
			needed = 1
		}
		for range needed {
			if !tpmBucket.Allow() {
				return false
			}
		}
	}
	return true
}

func (rl *RateLimiter) WaitTime(modelID string, tokens int) time.Duration {
	rpmBucket, tpmBucket := rl.bucketsFor(modelID)
	if rpmBucket == nil && tpmBucket == nil {
		return 0
	}
	wait := time.Duration(0)
	if rpmBucket != nil {
		if wt := rpmBucket.WaitTime(); wt > wait {
			wait = wt
		}
	}
	if tpmBucket != nil {
		wt := tpmBucket.WaitTime()
		if tokens > 0 {
			est := time.Duration(tokens) * wt / time.Duration(1)
			if est > wt {
				wt = est
			}
		}
		if wt > wait {
			wait = wt
		}
	}
	return wait
}

func (rl *RateLimiter) bucketsFor(modelID string) (*TokenBucket, *TokenBucket) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.rpm[modelID], rl.tpm[modelID]
}
