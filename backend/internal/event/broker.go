package event

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Event struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	OccurredAt time.Time `json:"occurred_at"`
	Data       any       `json:"data"`
}

type Broker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Event]struct{}
	bufferSize  int
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]map[chan Event]struct{}),
		bufferSize:  64,
	}
}

func ConversationTopic(conversationID uuid.UUID) string {
	return fmt.Sprintf("conversation:%s", conversationID)
}

func (b *Broker) Subscribe(topic string) (<-chan Event, func()) {
	ch := make(chan Event, b.bufferSize)

	b.mu.Lock()
	if b.subscribers[topic] == nil {
		b.subscribers[topic] = make(map[chan Event]struct{})
	}
	b.subscribers[topic][ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subscribers := b.subscribers[topic]
		if subscribers == nil {
			return
		}
		if _, ok := subscribers[ch]; !ok {
			return
		}
		delete(subscribers, ch)
		close(ch)
		if len(subscribers) == 0 {
			delete(b.subscribers, topic)
		}
	}
	return ch, unsubscribe
}

func (b *Broker) Publish(topic string, event Event) {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers[topic] {
		select {
		case ch <- event:
		default:
			log.Warn().
				Str("topic", topic).
				Str("event_type", event.Type).
				Msg("event subscriber is slow; dropping event")
		}
	}
}
