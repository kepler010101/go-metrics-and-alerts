package repository

import (
	"sync"
)

func (m *MemStorage) Reset() {
	if m == nil {
		return
	}
	clear(m.gauges)
	clear(m.counters)
	if m.mu != nil {
		if resetter, ok := interface{}(m.mu).(interface{ Reset() }); ok {
			resetter.Reset()
		} else {
			*m.mu = sync.Mutex{}
		}
	}
}
