package auth

import (
	"fmt"
	"sync"
	"time"
)

const (
	maxLoginAttempts  = 5
	loginLockDuration = 15 * time.Minute
	loginFailWindow   = 10 * time.Minute

	maxLoginPerIP         = 30
	loginIPWindow         = 10 * time.Minute
	maxPasswordChanges    = 5
	passwordChangeWindow  = 10 * time.Minute
	maxSensitiveAPIPerMin = 60
	sensitiveAPIWindow    = time.Minute
	maxStrictAPIPerMin    = 20
	strictAPIWindow       = time.Minute
)

type slidingWindowLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

var (
	loginIPLimit        = &slidingWindowLimiter{hits: make(map[string][]time.Time)}
	passwordChangeLimit = &slidingWindowLimiter{hits: make(map[string][]time.Time)}
	sensitiveAPILimit   = &slidingWindowLimiter{hits: make(map[string][]time.Time)}
	strictAPILimit      = &slidingWindowLimiter{hits: make(map[string][]time.Time)}
)

type loginLimiter struct {
	mu          sync.Mutex
	failures    map[string][]time.Time
	lockedUntil map[string]time.Time
}

var loginLimit = &loginLimiter{
	failures:    make(map[string][]time.Time),
	lockedUntil: make(map[string]time.Time),
}

func loginKey(ip, username string) string {
	return ip + "|" + username
}

func CheckLoginAllowed(ip, username string) error {
	key := loginKey(ip, username)
	loginLimit.mu.Lock()
	defer loginLimit.mu.Unlock()
	if until, ok := loginLimit.lockedUntil[key]; ok {
		if time.Now().Before(until) {
			return fmt.Errorf("登录尝试过多，请 %d 分钟后再试", int(time.Until(until).Minutes())+1)
		}
		delete(loginLimit.lockedUntil, key)
		delete(loginLimit.failures, key)
	}
	return nil
}

func RecordLoginFailure(ip, username string) {
	key := loginKey(ip, username)
	now := time.Now()
	loginLimit.mu.Lock()
	defer loginLimit.mu.Unlock()
	cutoff := now.Add(-loginFailWindow)
	var recent []time.Time
	for _, t := range loginLimit.failures[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	loginLimit.failures[key] = recent
	if len(recent) >= maxLoginAttempts {
		loginLimit.lockedUntil[key] = now.Add(loginLockDuration)
	}
}

func RecordLoginSuccess(ip, username string) {
	key := loginKey(ip, username)
	loginLimit.mu.Lock()
	defer loginLimit.mu.Unlock()
	delete(loginLimit.failures, key)
	delete(loginLimit.lockedUntil, key)
}

func ClearLoginLimits() {
	loginLimit.mu.Lock()
	defer loginLimit.mu.Unlock()
	loginLimit.failures = make(map[string][]time.Time)
	loginLimit.lockedUntil = make(map[string]time.Time)
}

func checkSlidingWindow(l *slidingWindowLimiter, key string, max int, window time.Duration) error {
	if max <= 0 || key == "" {
		return nil
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-window)
	var recent []time.Time
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= max {
		return fmt.Errorf("请求过于频繁，请稍后再试")
	}
	return nil
}

func recordSlidingWindow(l *slidingWindowLimiter, key string, window time.Duration) {
	if key == "" {
		return
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-window)
	var recent []time.Time
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	l.hits[key] = append(recent, now)
}

func CheckLoginIPAllowed(ip string) error {
	return checkSlidingWindow(loginIPLimit, ip, maxLoginPerIP, loginIPWindow)
}

func RecordLoginIPAttempt(ip string) {
	recordSlidingWindow(loginIPLimit, ip, loginIPWindow)
}

func CheckPasswordChangeAllowed(userID uint, ip string) error {
	key := fmt.Sprintf("%d|%s", userID, ip)
	return checkSlidingWindow(passwordChangeLimit, key, maxPasswordChanges, passwordChangeWindow)
}

func RecordPasswordChange(userID uint, ip string) {
	key := fmt.Sprintf("%d|%s", userID, ip)
	recordSlidingWindow(passwordChangeLimit, key, passwordChangeWindow)
}

func CheckSensitiveAPI(key string) error {
	return checkSlidingWindow(sensitiveAPILimit, key, maxSensitiveAPIPerMin, sensitiveAPIWindow)
}

func RecordSensitiveAPI(key string) {
	recordSlidingWindow(sensitiveAPILimit, key, sensitiveAPIWindow)
}

func CheckStrictAPI(key string) error {
	return checkSlidingWindow(strictAPILimit, key, maxStrictAPIPerMin, strictAPIWindow)
}

func RecordStrictAPI(key string) {
	recordSlidingWindow(strictAPILimit, key, strictAPIWindow)
}

func SensitiveAPIKey(userID uint, ip, scope string) string {
	return fmt.Sprintf("%s|%d|%s", scope, userID, ip)
}
