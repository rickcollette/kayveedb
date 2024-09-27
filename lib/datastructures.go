package lib

import (
	"errors"
	"sort"
	"sync"
)

// Data Structure: List
type ListManager struct {
	lists map[string][]string
	mu    sync.Mutex
}

func NewListManager() *ListManager {
	return &ListManager{
		lists: make(map[string][]string),
	}
}

func (lm *ListManager) LPush(key, value string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.lists[key] = append([]string{value}, lm.lists[key]...)
}

func (lm *ListManager) RPush(key, value string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.lists[key] = append(lm.lists[key], value)
}

func (lm *ListManager) LRange(key string, start, stop int) ([]string, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if list, exists := lm.lists[key]; exists {
		if start < 0 || stop > len(list) {
			return nil, errors.New("range out of bounds")
		}
		return list[start:stop], nil
	}
	return nil, errors.New("list not found")
}

// Data Structure: Set
type SetManager struct {
	sets map[string]map[string]bool
	mu   sync.Mutex
}

func NewSetManager() *SetManager {
	return &SetManager{
		sets: make(map[string]map[string]bool),
	}
}

func (sm *SetManager) SAdd(key, member string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if _, exists := sm.sets[key]; !exists {
		sm.sets[key] = make(map[string]bool)
	}
	sm.sets[key][member] = true
}

func (sm *SetManager) SMembers(key string) ([]string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if set, exists := sm.sets[key]; exists {
		members := make([]string, 0, len(set))
		for member := range set {
			members = append(members, member)
		}
		return members, nil
	}
	return nil, errors.New("set not found")
}

// Data Structure: Hash
type HashManager struct {
	hashes map[string]map[string]string
	mu     sync.Mutex
}

func NewHashManager() *HashManager {
	return &HashManager{
		hashes: make(map[string]map[string]string),
	}
}

func (hm *HashManager) HSet(key, field, value string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if _, exists := hm.hashes[key]; !exists {
		hm.hashes[key] = make(map[string]string)
	}
	hm.hashes[key][field] = value
}

func (hm *HashManager) HGet(key, field string) (string, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if hash, exists := hm.hashes[key]; exists {
		if value, exists := hash[field]; exists {
			return value, nil
		}
		return "", errors.New("field not found")
	}
	return "", errors.New("hash not found")
}

// Data Structure: Sorted Set
type ZSetManager struct {
	zsets map[string]map[string]float64
	mu    sync.Mutex
}

func NewZSetManager() *ZSetManager {
	return &ZSetManager{
		zsets: make(map[string]map[string]float64),
	}
}

func (zsm *ZSetManager) ZAdd(key, member string, score float64) {
	zsm.mu.Lock()
	defer zsm.mu.Unlock()
	if _, exists := zsm.zsets[key]; !exists {
		zsm.zsets[key] = make(map[string]float64)
	}
	zsm.zsets[key][member] = score
}

func (zsm *ZSetManager) ZRange(key string, start, stop int) ([]string, error) {
	zsm.mu.Lock()
	defer zsm.mu.Unlock()
	if zset, exists := zsm.zsets[key]; exists {
		var members []struct {
			member string
			score  float64
		}
		for member, score := range zset {
			members = append(members, struct {
				member string
				score  float64
			}{member, score})
		}
		sort.Slice(members, func(i, j int) bool {
			return members[i].score < members[j].score
		})
		var result []string
		for i := start; i < stop && i < len(members); i++ {
			result = append(result, members[i].member)
		}
		return result, nil
	}
	return nil, errors.New("sorted set not found")
}
