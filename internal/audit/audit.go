// Package audit provides a tiny observer used to record metric events.
package audit

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// Event describes one audit record produced after a metrics request.
type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Listener receives audit events.
type Listener interface {
	Handle(Event)
}

// Notifier broadcasts audit events to registered listeners.
type Notifier interface {
	Publish(Event)
}

// Publisher stores listeners and publishes events to them.
type Publisher struct {
	listeners []Listener
}

// NewPublisher creates a Publisher with an empty listener list.
func NewPublisher() *Publisher {
	return &Publisher{}
}

// Register adds a new listener into the broadcast list.
func (p *Publisher) Register(listener Listener) {
	if listener == nil {
		return
	}
	p.listeners = append(p.listeners, listener)
}

// HasListeners reports whether the publisher has at least one subscriber.
func (p *Publisher) HasListeners() bool {
	return len(p.listeners) > 0
}

// Publish sends the event to every registered listener.
func (p *Publisher) Publish(event Event) {
	for _, listener := range p.listeners {
		listener.Handle(event)
	}
}

// FileListener appends events to the provided file path.
type FileListener struct {
	path string
}

// NewFileListener creates a file based listener that writes JSON lines.
func NewFileListener(path string) *FileListener {
	return &FileListener{path: path}
}

// Handle saves the event into the configured file.
func (l *FileListener) Handle(event Event) {
	if l == nil || l.path == "" {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit: marshal error: %v", err)
		return
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("audit: open file error: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Printf("audit: write file error: %v", err)
	}
}

type HTTPListener struct {
	url    string
	client *http.Client
}

// NewHTTPListener creates a listener that POSTs events to the given URL.
func NewHTTPListener(url string) *HTTPListener {
	client := &http.Client{Timeout: 5 * time.Second}
	return &HTTPListener{
		url:    url,
		client: client,
	}
}

// Handle sends the event as JSON to the remote endpoint.
func (l *HTTPListener) Handle(event Event) {
	if l == nil || l.url == "" || l.client == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit: marshal error: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, l.url, bytes.NewReader(data))
	if err != nil {
		log.Printf("audit: create request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		log.Printf("audit: http request error: %v", err)
		return
	}
	defer resp.Body.Close()
}
