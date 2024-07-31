package cache

import (
	"crypto/ed25519"
	"github.com/google/uuid"
	"sync"
)

type KeyCache interface {
	Set(realmID uuid.UUID, key ed25519.PrivateKey)
	Get(realmID uuid.UUID) (ed25519.PrivateKey, bool)
}

type InMemoryKeyCache struct {
	mu    sync.RWMutex
	cache map[uuid.UUID]ed25519.PrivateKey
}

func NewInMemoryKeyCache() *InMemoryKeyCache {
	return &InMemoryKeyCache{
		cache: make(map[uuid.UUID]ed25519.PrivateKey),
	}
}

func (kc *InMemoryKeyCache) Set(realmID uuid.UUID, key ed25519.PrivateKey) {
	kc.mu.Lock()
	defer kc.mu.Unlock()
	kc.cache[realmID] = key
}

func (kc *InMemoryKeyCache) Get(realmID uuid.UUID) (ed25519.PrivateKey, bool) {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	key, found := kc.cache[realmID]
	return key, found
}
