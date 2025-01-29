package chat

import (
	"sync"

	"github.com/gorilla/websocket"
)

type NotificationType string

const (
	ReactionNotification    NotificationType = "reaction"
	ModerationNotification  NotificationType = "moderation"
	MessageNotification     NotificationType = "message"
	ParticipantNotification NotificationType = "participant"
)

type Notification struct {
	Type      NotificationType `json:"type"`
	SessionID string           `json:"sessionId"`
	Data      interface{}      `json:"data"`
}

type NotificationHub struct {
	clients    map[string]*websocket.Conn
	Broadcast  chan Notification
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
	mu         sync.Mutex
}

func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		clients:    make(map[string]*websocket.Conn),
		Broadcast:  make(chan Notification),
		Register:   make(chan *websocket.Conn),
		Unregister: make(chan *websocket.Conn),
	}
}

func (h *NotificationHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.RemoteAddr().String()] = client
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.RemoteAddr().String()]; ok {
				delete(h.clients, client.RemoteAddr().String())
				client.Close()
			}
			h.mu.Unlock()

		case notification := <-h.Broadcast:
			h.mu.Lock()
			for _, client := range h.clients {
				err := client.WriteJSON(notification)
				if err != nil {
					client.Close()
					delete(h.clients, client.RemoteAddr().String())
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *NotificationHub) SendNotification(notification Notification) {
	h.Broadcast <- notification
}
