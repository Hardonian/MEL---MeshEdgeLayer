package events

import (
	"sync"
	"sync/atomic"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Bus struct {
	mu          sync.RWMutex
	subs        []chan Event
	published   atomic.Uint64
	dropped     atomic.Uint64
	droppedByTy sync.Map
}

func New() *Bus { return &Bus{} }

func (b *Bus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 256)
	b.subs = append(b.subs, ch)
	return ch
}

func (b *Bus) Publish(evt Event) (delivered bool, dropped int) {
	b.published.Add(1)
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- evt:
			delivered = true
		default:
			dropped++
		}
	}
	if dropped > 0 {
		b.dropped.Add(uint64(dropped))
		current, _ := b.droppedByTy.LoadOrStore(evt.Type, new(atomic.Uint64))
		current.(*atomic.Uint64).Add(uint64(dropped))
	}
	return delivered, dropped
}

type Stats struct {
	Published     uint64            `json:"published"`
	Dropped       uint64            `json:"dropped"`
	DroppedByType map[string]uint64 `json:"dropped_by_type"`
}

func (b *Bus) Stats() Stats {
	stats := Stats{Published: b.published.Load(), Dropped: b.dropped.Load(), DroppedByType: map[string]uint64{}}
	b.droppedByTy.Range(func(key, value any) bool {
		stats.DroppedByType[key.(string)] = value.(*atomic.Uint64).Load()
		return true
	})
	return stats
}
