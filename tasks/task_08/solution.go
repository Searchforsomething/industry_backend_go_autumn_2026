package main

import (
	"fmt"
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	if clock == nil || burst <= 0 {
		var zeroTime time.Time
		return &Limiter{
			clock:  clock,
			rate:   0,
			burst:  0,
			tokens: 0,
			last:   zeroTime,
		}
	}
	if ratePerSec < 0 {
		ratePerSec = 0
	}
	return &Limiter{
		clock:  clock,
		rate:   ratePerSec,
		burst:  burst,
		tokens: float64(burst),
		last:   clock.Now(),
	}
}
func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n <= 0 || n > l.burst {
		return false
	}
	nowTime := l.clock.Now()
	diffTime := nowTime.Sub(l.last).Seconds()
	if diffTime <= 0 {
		diffTime = 0
	} else {
		l.last = nowTime
	}
	newTokens := min(float64(l.burst), l.tokens+diffTime*l.rate)
	fmt.Println(newTokens)

	if float64(n) <= newTokens {
		l.tokens = newTokens - float64(n)
		return true
	}
	l.tokens = newTokens
	return false
}
