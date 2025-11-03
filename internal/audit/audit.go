package audit

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Listener interface {
	Handle(Event)
}

type Notifier interface {
	Publish(Event)
}

type Publisher struct {
	listeners []Listener
}

func NewPublisher() *Publisher {
	return &Publisher{}
}

func (p *Publisher) Register(listener Listener) {
	if listener == nil {
		return
	}
	p.listeners = append(p.listeners, listener)
}

func (p *Publisher) HasListeners() bool {
	return len(p.listeners) > 0
}

func (p *Publisher) Publish(event Event) {
	for _, listener := range p.listeners {
		listener.Handle(event)
	}
}

type FileListener struct {
	path string
}

func NewFileListener(path string) *FileListener {
	return &FileListener{path: path}
}

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

func NewHTTPListener(url string) *HTTPListener {
	client := &http.Client{Timeout: 5 * time.Second}
	return &HTTPListener{
		url:    url,
		client: client,
	}
}

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
