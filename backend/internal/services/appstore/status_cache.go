package appstore

import "time"

const liveStatusCacheTTL = 30 * time.Second
const reconcileMinInterval = 5 * time.Minute

type statusEntry struct {
	status string
	at     time.Time
}

func (s *Service) LiveStatus(key string) string {
	s.statusMu.RLock()
	if e, ok := s.statusCache[key]; ok && time.Since(e.at) < liveStatusCacheTTL {
		st := e.status
		s.statusMu.RUnlock()
		return st
	}
	s.statusMu.RUnlock()

	st := s.detectAppStatus(key)
	s.statusMu.Lock()
	if s.statusCache == nil {
		s.statusCache = make(map[string]statusEntry)
	}
	s.statusCache[key] = statusEntry{status: st, at: time.Now()}
	s.statusMu.Unlock()
	return st
}

func (s *Service) LiveStatusMap(keys []string) map[string]string {
	out := make(map[string]string, len(keys))
	missing := make([]string, 0)
	now := time.Now()

	s.statusMu.RLock()
	for _, key := range keys {
		if e, ok := s.statusCache[key]; ok && now.Sub(e.at) < liveStatusCacheTTL {
			out[key] = e.status
		} else {
			missing = append(missing, key)
		}
	}
	s.statusMu.RUnlock()

	for _, key := range missing {
		out[key] = s.LiveStatus(key)
	}
	return out
}

func (s *Service) InvalidateLiveStatus(keys ...string) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if s.statusCache == nil {
		return
	}
	if len(keys) == 0 {
		s.statusCache = make(map[string]statusEntry)
		return
	}
	for _, key := range keys {
		delete(s.statusCache, key)
	}
}
