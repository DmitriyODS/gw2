// Package hub — реестр WS-клиентов и комнат realtime-шлюза.
//
// Комнаты повторяют прежний Flask-SocketIO: каждый авторизованный клиент
// состоит в "all" и "user_{id}". События доставляются кадрами
// {"event": ..., "data": ...} — формат тонкой WS-обёртки фронта.
package hub

import (
	"encoding/json"
	"strconv"
	"sync"
	"sync/atomic"
)

// sendBuffer — ёмкость исходящей очереди клиента; переполнение (мёртвый или
// безнадёжно медленный потребитель) закрывает соединение — клиент
// переподключится и дотянет состояние REST-запросами.
const sendBuffer = 256

var connSeq atomic.Int64

// Client — одно WS-соединение авторизованного пользователя.
type Client struct {
	UserID int64
	ConnID string

	send   chan []byte
	closed sync.Once
	done   chan struct{}
}

func NewClient(userID int64) *Client {
	return &Client{
		UserID: userID,
		ConnID: strconv.FormatInt(connSeq.Add(1), 10),
		send:   make(chan []byte, sendBuffer),
		done:   make(chan struct{}),
	}
}

// Frames — канал исходящих кадров (читает writer-горутина транспорта).
func (c *Client) Frames() <-chan []byte { return c.send }

// Done — закрыт, когда клиента нужно отключить.
func (c *Client) Done() <-chan struct{} { return c.done }

// Close — идемпотентное закрытие (slow consumer, ошибка записи, disconnect).
func (c *Client) Close() {
	c.closed.Do(func() { close(c.done) })
}

func (c *Client) push(frame []byte) {
	select {
	case c.send <- frame:
	case <-c.done:
	default:
		// Очередь забита — потребитель мёртв.
		c.Close()
	}
}

// Frame — кадр протокола {"event": ..., "data": ...}.
type Frame struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// MarshalFrame — собрать кадр из уже сериализованного payload'а.
func MarshalFrame(event string, data json.RawMessage) []byte {
	if data == nil {
		data = json.RawMessage("null")
	}
	raw, _ := json.Marshal(Frame{Event: event, Data: data})
	return raw
}

// Hub — комнаты → клиенты.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
	// membership — обратный индекс для быстрого Remove.
	membership map[*Client][]string
}

func New() *Hub {
	return &Hub{
		rooms:      map[string]map[*Client]struct{}{},
		membership: map[*Client][]string{},
	}
}

// Add — зарегистрировать клиента в комнатах (идемпотентно).
func (h *Hub) Add(c *Client, rooms ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, room := range rooms {
		if h.rooms[room] == nil {
			h.rooms[room] = map[*Client]struct{}{}
		}
		h.rooms[room][c] = struct{}{}
	}
	h.membership[c] = append(h.membership[c], rooms...)
}

// Remove — убрать клиента из всех комнат.
func (h *Hub) Remove(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, room := range h.membership[c] {
		delete(h.rooms[room], c)
		if len(h.rooms[room]) == 0 {
			delete(h.rooms, room)
		}
	}
	delete(h.membership, c)
}

// Broadcast — доставить кадр всем клиентам комнаты.
func (h *Hub) Broadcast(room string, frame []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[room] {
		c.push(frame)
	}
}
