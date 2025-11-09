package resetpool

import "testing"

type sample struct {
	value int
	data  []int
}

func (s *sample) Reset() {
	if s == nil {
		return
	}
	s.value = 0
	s.data = s.data[:0]
}

func TestPoolGetPut(t *testing.T) {
	p := New(func() *sample {
		return &sample{}
	})

	item := p.Get()
	if item == nil {
		t.Fatal("expected non-nil item")
	}

	item.value = 10
	item.data = append(item.data, 1, 2, 3)

	p.Put(item)

	if item.value != 0 {
		t.Fatalf("value not reset: %d", item.value)
	}
	if len(item.data) != 0 {
		t.Fatalf("slice not reset, len=%d", len(item.data))
	}

	item2 := p.Get()
	if item2 == nil {
		t.Fatal("expected pooled item")
	}

	if len(item2.data) != 0 || item2.value != 0 {
		t.Fatal("pooled item not reset")
	}
}
