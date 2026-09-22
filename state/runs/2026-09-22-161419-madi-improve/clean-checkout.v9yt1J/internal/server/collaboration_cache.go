package server

import (
	"sync"

	"github.com/reearth/ygo/crdt"
)

const collaborationCacheEntries = 8
const collaborationCacheBudget = 64 << 20

type collaborationCachedDocument struct {
	mu                sync.Mutex
	doc               *crdt.Doc
	epoch, hash, body string
	sequence          int64
	weight            int64
	raw               []byte
}

// The database row lock is acquired BEFORE this cache lock. Entries are never
// authoritative: an epoch/sequence/hash mismatch (including backup rollback and
// later sequence reuse) discards them. An uncommitted mutation is never retained.
func (s *Server) collaborationCached(id string) (*collaborationRuntime, *collaborationCachedDocument) {
	s.collaborationMu.Lock()
	rt := s.collaboration
	s.collaborationMu.Unlock()
	if rt == nil || rt.ctx.Err() != nil {
		return nil, nil
	}
	rt.mu.Lock()
	active := len(rt.rooms[id]) > 0
	rt.mu.Unlock()
	if !active {
		return nil, nil
	}
	rt.cacheMu.Lock()
	if rt.cache == nil {
		rt.cache = map[string]*collaborationCachedDocument{}
	}
	entry := rt.cache[id]
	if entry == nil && len(rt.cache) < collaborationCacheEntries {
		entry = &collaborationCachedDocument{}
		rt.cache[id] = entry
	}
	rt.cacheMu.Unlock()
	if entry == nil {
		return nil, nil
	}
	entry.mu.Lock()
	return rt, entry
}

func (rt *collaborationRuntime) discardCached(entry *collaborationCachedDocument) {
	if entry.doc != nil {
		entry.doc.Destroy()
		entry.doc = nil
	}
	rt.cacheBytes.Add(-entry.weight)
	entry.weight = 0
	entry.epoch, entry.hash, entry.body = "", "", ""
	entry.raw = nil
}

func (rt *collaborationRuntime) clearCollaborationCache(id string) {
	rt.cacheMu.Lock()
	entries := []*collaborationCachedDocument{}
	for key, entry := range rt.cache {
		if id == "*" || id == key {
			entries = append(entries, entry)
			delete(rt.cache, key)
		}
	}
	rt.cacheMu.Unlock()
	for _, entry := range entries {
		entry.mu.Lock()
		rt.discardCached(entry)
		entry.mu.Unlock()
	}
}
