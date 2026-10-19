package telegram

import "sync"

// keyedMutexes provides one mutex per integer key. Used to serialize
// operations that must not run concurrently for the same key but are
// independent across keys (e.g. auth export/import pairs per target DC).
type keyedMutexes struct {
	mu    sync.Mutex
	locks map[int]*sync.Mutex
}

func (k *keyedMutexes) Lock(key int) {
	k.mu.Lock()
	if k.locks == nil {
		k.locks = make(map[int]*sync.Mutex)
	}
	l, ok := k.locks[key]
	if !ok {
		l = &sync.Mutex{}
		k.locks[key] = l
	}
	k.mu.Unlock()
	l.Lock()
}

func (k *keyedMutexes) Unlock(key int) {
	k.mu.Lock()
	l, ok := k.locks[key]
	k.mu.Unlock()
	if !ok {
		return // unreachable when paired with Lock
	}
	l.Unlock()
}
