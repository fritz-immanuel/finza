package ratelimit

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type bucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type Limiter struct {
	mu      sync.Mutex
	rps     float64
	burst   float64
	ttl     time.Duration
	clock   Clock
	perUser map[int64]*bucket
}

func New(rps float64, burst int) *Limiter {
	return NewWithClock(rps, burst, realClock{})
}

func NewWithClock(rps float64, burst int, clock Clock) *Limiter {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &Limiter{
		rps:     rps,
		burst:   float64(burst),
		ttl:     10 * time.Minute,
		clock:   clock,
		perUser: make(map[int64]*bucket),
	}
}

func (l *Limiter) Allow(userID int64) bool {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.perUser[userID]
	if !ok {
		l.perUser[userID] = &bucket{tokens: l.burst - 1, lastRefill: now, lastSeen: now}
		return true
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.rps
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.lastRefill = now
	}

	b.lastSeen = now
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

func (l *Limiter) Cleanup() {
	cutoff := l.clock.Now().Add(-l.ttl)

	l.mu.Lock()
	defer l.mu.Unlock()

	for userID, b := range l.perUser {
		if b.lastSeen.Before(cutoff) {
			delete(l.perUser, userID)
		}
	}
}
