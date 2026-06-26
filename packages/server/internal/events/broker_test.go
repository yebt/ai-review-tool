package events

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestBrokerSubscribePublishAndUnsubscribe(t *testing.T) {
	broker := NewBroker()
	ch, unsubscribe := broker.Subscribe("review_1")

	broker.Publish("review_1", "review.started", map[string]any{"review_id": "review_1"})
	event := receiveEvent(t, ch)
	if event.Name != "review.started" {
		t.Fatalf("event name = %q, want review.started", event.Name)
	}
	var payload map[string]string
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("event data is not JSON: %v", err)
	}
	if payload["review_id"] != "review_1" {
		t.Fatalf("payload review_id = %q, want review_1", payload["review_id"])
	}

	unsubscribe()
	if _, ok := <-ch; ok {
		t.Fatal("channel remained open after unsubscribe")
	}
}

func TestBrokerPublishesOnlyToMatchingReviewSubscribers(t *testing.T) {
	broker := NewBroker()
	matching, unsubscribeMatching := broker.Subscribe("review_1")
	defer unsubscribeMatching()
	other, unsubscribeOther := broker.Subscribe("review_2")
	defer unsubscribeOther()

	broker.Publish("review_1", "agent.started", map[string]any{"review_id": "review_1"})
	if event := receiveEvent(t, matching); event.Name != "agent.started" {
		t.Fatalf("event name = %q, want agent.started", event.Name)
	}
	assertNoEvent(t, other)
}

func TestBrokerConcurrentPublishDelivery(t *testing.T) {
	broker := NewBroker()
	ch, unsubscribe := broker.Subscribe("review_1")
	defer unsubscribe()

	const publishers = 8
	var wg sync.WaitGroup
	for i := 0; i < publishers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			broker.Publish("review_1", "agent.completed", map[string]any{"review_id": "review_1"})
		}()
	}
	wg.Wait()

	for i := 0; i < publishers; i++ {
		if event := receiveEvent(t, ch); event.Name != "agent.completed" {
			t.Fatalf("event %d name = %q, want agent.completed", i, event.Name)
		}
	}
}

func TestBrokerSlowConsumerDoesNotBlockPublish(t *testing.T) {
	broker := NewBroker()
	ch, unsubscribe := broker.Subscribe("review_1")
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 128; i++ {
			broker.Publish("review_1", "agent.started", map[string]any{"review_id": "review_1"})
		}
	}()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Publish blocked behind a slow subscriber")
	}

	if len(ch) == 0 {
		t.Fatal("slow subscriber did not receive any queued events")
	}
}

func receiveEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case event, ok := <-ch:
		if !ok {
			t.Fatal("event channel closed unexpectedly")
		}
		return event
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}
	return Event{}
}

func assertNoEvent(t *testing.T, ch <-chan Event) {
	t.Helper()
	select {
	case event := <-ch:
		t.Fatalf("unexpected event: %+v", event)
	case <-time.After(20 * time.Millisecond):
	}
}
