package blockfrost

import (
	"context"
	"github.com/blockblu-io/leaderlog-api/pkg/chain"
	"sync"
)

// fetchPoolMetadataFunc is a function to fetch pool metadata
type fetchPoolMetadataFunc func(context.Context, string) (*chain.StakePool, error)

// poolCache allows fetching pool metadata by making use of a
// cache. The cache stores a specified number of pool metadata.
type poolCache struct {
	size        int
	latestPools []string
	cache       map[string]chain.StakePool
	lock        sync.Mutex
}

// newPoolCache creates a new pool cache of a given size.
func newPoolCache(cacheSize int) poolCache {
	return poolCache{
		size:        cacheSize,
		latestPools: []string{},
		cache:       map[string]chain.StakePool{},
		lock:        sync.Mutex{},
	}
}

// registerPool registers the given pool ID into the cache. Old
// entries of the cache might be deleted depending on the size
// of the cache.
func (p *poolCache) registerPool(poolID string) {
	cleanList := make([]string, 0)
	for _, pID := range p.latestPools {
		if pID != poolID {
			cleanList = append(cleanList, pID)
		}
	}
	var newList []string = nil
	if len(cleanList) >= (p.size - 1) {
		delete(p.cache, cleanList[0])
		newList = cleanList[1:]
	} else {
		newList = cleanList[:]
	}
	newList = append(newList, poolID)
}

// fetchPoolMetadata fetches the metadata of the pool with the given ID. If
// the metadata is cached, then the cached metadata will be returned. If it
// isn't in the cache, then the given function 'f' will be called to fetch
// the metadata.
//
// An error will be returned, if the metadata couldn't be fetched.
func (p *poolCache) fetchPoolMetadata(ctx context.Context, poolID string,
	f fetchPoolMetadataFunc) (*chain.StakePool, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.registerPool(poolID)
	pool, found := p.cache[poolID]
	if found {
		return &pool, nil
	} else {
		pool, err := f(ctx, poolID)
		if err != nil {
			return nil, err
		}
		p.cache[poolID] = *pool
		return pool, nil
	}
}
