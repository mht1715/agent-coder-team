package event

import (
	"testing"
	"time"
)

func TestBrokerPublishSubscribe(t *testing.T) {
	broker := NewBroker()
	events, unsubscribe := broker.Subscribe("test")
	defer unsubscribe()

	broker.Publish("test", Event{ID: "evt-1", Type: "message.created", Data: map[string]string{"id": "msg-1"}})

	select {
	case event := <-events:
		if event.ID != "evt-1" {
			t.Fatalf("expected evt-1, got %s", event.ID)
		}
		if event.Type != "message.created" {
			t.Fatalf("expected message.created, got %s", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBrokerUnsubscribe(t *testing.T) {
	broker := NewBroker()
	events, unsubscribe := broker.Subscribe("test")
	unsubscribe()

	if _, ok := <-events; ok {
		t.Fatal("expected channel to be closed")
	}
}
