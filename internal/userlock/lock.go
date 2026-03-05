package userlock

import "sync"

type Locker struct {
	m sync.Map
}

func New() *Locker {
	return &Locker{}
}

func (l *Locker) Lock(userID int64) func() {
	v, _ := l.m.LoadOrStore(userID, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}
