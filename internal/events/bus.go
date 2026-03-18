package events

import "sync"

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Bus struct {
	mu   sync.RWMutex
	subs []chan Event
}

func New() *Bus { return &Bus{} }

func (b *Bus) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 32)
	b.subs = append(b.subs, ch)
	return ch
}

func (b *Bus) Publish(evt Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- evt:
		default:
		}
	}
}
