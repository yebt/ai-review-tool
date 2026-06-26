package events

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

type Broker struct {
	mu          sync.Mutex
	subscribers map[string]map[chan Event]struct{}
}

func NewBroker() *Broker {
	return &Broker{subscribers: make(map[string]map[chan Event]struct{})}
}

func (b *Broker) Subscribe(reviewID string) (<-chan Event, func()) {
	ch := make(chan Event, 16)
	b.mu.Lock()
	if b.subscribers[reviewID] == nil {
		b.subscribers[reviewID] = make(map[chan Event]struct{})
	}
	b.subscribers[reviewID][ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subscribers[reviewID], ch)
		close(ch)
		if len(b.subscribers[reviewID]) == 0 {
			delete(b.subscribers, reviewID)
		}
	}
}

func (b *Broker) Publish(reviewID string, name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(`{"error":"event payload marshal failed"}`)
	}
	event := Event{Name: name, Data: data}
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers[reviewID] {
		select {
		case ch <- event:
		default:
		}
	}
}
