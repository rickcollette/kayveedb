package lib

import (
	"sync"
)

type Subscriber chan string

type PubSub struct {
	channels map[string][]Subscriber
	mu       sync.Mutex
}

func NewPubSub() *PubSub {
	return &PubSub{
		channels: make(map[string][]Subscriber),
	}
}

// Publish a message to a channel
func (ps *PubSub) Publish(channel, message string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if subs, exists := ps.channels[channel]; exists {
		for _, sub := range subs {
			sub <- message
		}
	}
}

// Subscribe to a channel
func (ps *PubSub) Subscribe(channel string) Subscriber {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	sub := make(Subscriber, 100) // Buffered channel
	ps.channels[channel] = append(ps.channels[channel], sub)
	return sub
}

// Unsubscribe from a channel
func (ps *PubSub) Unsubscribe(channel string, sub Subscriber) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if subs, exists := ps.channels[channel]; exists {
		for i, s := range subs {
			if s == sub {
				ps.channels[channel] = append(subs[:i], subs[i+1:]...)
				close(sub)
				break
			}
		}
	}
}
