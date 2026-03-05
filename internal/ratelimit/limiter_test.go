package ratelimit

import (
	"testing"
	"time"
)

type fakeClock struct{ now time.Time }

func (f *fakeClock) Now() time.Time { return f.now }

func TestLimiter(t *testing.T) {
	fc := &fakeClock{now: time.Unix(0, 0)}
	lim := NewWithClock(2, 3, fc)

	t.Run("under limit", func(t *testing.T) {
		if !lim.Allow(1) {
			t.Fatal("first request should pass")
		}
		if !lim.Allow(1) {
			t.Fatal("second request should pass")
		}
	})

	t.Run("at burst", func(t *testing.T) {
		fc.now = fc.now.Add(2 * time.Second)
		if !lim.Allow(2) || !lim.Allow(2) || !lim.Allow(2) {
			t.Fatal("burst requests should pass")
		}
	})

	t.Run("over burst", func(t *testing.T) {
		if lim.Allow(2) {
			t.Fatal("extra request should be rejected")
		}
	})

	t.Run("after refill", func(t *testing.T) {
		fc.now = fc.now.Add(2 * time.Second)
		if !lim.Allow(2) {
			t.Fatal("request should pass after refill")
		}
	})
}
