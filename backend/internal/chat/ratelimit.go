package chat

import (
	"strconv"
	"sync"
	"time"
)

const (
	messageBurstPerSecond = 1
	messageMaxPerMinute   = 20
	typingInterval        = 2 * time.Second
)

// nowFn is overridable in tests so rate limits can be exercised without
// sleeping on wall-clock time.
var nowFn = time.Now

// messageLimiter enforces a 1/second burst plus a 20/minute cap per user. It is
// process-local, which matches the single-instance SQLite deployment.
type messageLimiter struct {
	mu     sync.Mutex
	last   map[int64]time.Time
	recent map[int64][]time.Time
}

func newMessageLimiter() *messageLimiter {
	return &messageLimiter{
		last:   make(map[int64]time.Time),
		recent: make(map[int64][]time.Time),
	}
}

func (l *messageLimiter) allow(userID int64, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-time.Minute)
	recent := l.recent[userID]
	kept := recent[:0]
	for _, at := range recent {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= messageMaxPerMinute {
		l.recent[userID] = kept
		return false
	}
	if last, ok := l.last[userID]; ok && now.Sub(last) < time.Second/messageBurstPerSecond {
		l.recent[userID] = kept
		return false
	}
	kept = append(kept, now)
	l.recent[userID] = kept
	l.last[userID] = now
	return true
}

// typingLimiter allows one typing update per 2s per user and scope.
type typingLimiter struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func newTypingLimiter() *typingLimiter {
	return &typingLimiter{last: make(map[string]time.Time)}
}

func (l *typingLimiter) allow(userID, channelID, conversationID int64, now time.Time) bool {
	key := strconv.FormatInt(userID, 10) + ":" +
		strconv.FormatInt(channelID, 10) + ":" +
		strconv.FormatInt(conversationID, 10)
	l.mu.Lock()
	defer l.mu.Unlock()
	if last, ok := l.last[key]; ok && now.Sub(last) < typingInterval {
		return false
	}
	l.last[key] = now
	return true
}
