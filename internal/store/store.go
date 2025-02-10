package store

import "sync"

type VisitorStore struct {
	mu     sync.RWMutex
	visits map[string]int
}

func NewVisitorStore() *VisitorStore {
	return &VisitorStore{
		visits: make(map[string]int),
	}
}

func (v *VisitorStore) Add(ip string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.visits[ip]++
}

func (v *VisitorStore) GetVisitsAll() map[string]int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	visitsCopy := make(map[string]int)
	for ip, count := range v.visits {
		visitsCopy[ip] = count
	}
	return visitsCopy
}
