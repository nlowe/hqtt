package testqtt

import (
	"bytes"
	"sync"
	"testing"
)

type Recorder struct {
	sync.RWMutex

	seen []Message
}

func (r *Recorder) Record(msg Message) {
	r.Lock()
	defer r.Unlock()

	r.seen = append(r.seen, msg)
}

func (r *Recorder) Reset() {
	r.Lock()
	defer r.Unlock()

	r.seen = nil
}

func (r *Recorder) All() []Message {
	r.RLock()
	defer r.RUnlock()

	seen := make([]Message, len(r.seen))
	copy(seen, r.seen)

	return seen
}

func (r *Recorder) AssertAnyMessage(t *testing.T, topic string) {
	for _, msg := range r.All() {
		if msg.Topic == topic {
			return
		}
	}

	t.Errorf("no message seen for topic %q", topic)
}

func (r *Recorder) AssertNoMessage(t *testing.T, topic string) {
	for _, msg := range r.All() {
		if msg.Topic == topic {
			t.Errorf("expected no message on topic %q but saw %q", topic, msg.Data)
		}
	}
}

func (r *Recorder) AssertMessage(t *testing.T, topic string, data []byte) {
	for _, msg := range r.All() {
		if msg.Topic != topic {
			continue
		}

		if bytes.Equal(msg.Data, data) {
			return
		}
	}

	t.Errorf("no message seen for topic %q with value %q", topic, data)
	for _, msg := range r.All() {
		t.Errorf("                      saw %q with value %q", msg.Topic, msg.Data)
	}
}

func (r *Recorder) AssertNoExactMessage(t *testing.T, topic string, data []byte) {
	for _, msg := range r.All() {
		if msg.Topic != topic {
			continue
		}

		if bytes.Equal(msg.Data, data) {
			t.Errorf("expected no message on topic %q with value %q", topic, data)
		}
	}
}
