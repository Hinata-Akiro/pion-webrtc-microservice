package chat

import (
	"sync"
	"time"
	"net/http"


	"pion-webrtc-microservice/utils"
)

type ChatMessage struct {
	ID         string    `json:"id"`
	SenderID   string    `json:"senderId"`
	ReceiverID string    `json:"receiverId"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
}

type ChatSession struct {
	ID           string
	Participants []string
	StartTime    time.Time
	EndTime      time.Time
	Messages     []ChatMessage
	mu           sync.Mutex
}

type ChatManager struct {
	sessions map[string]*ChatSession
	mu       sync.Mutex
}

func NewChatManager() *ChatManager {
	return &ChatManager{sessions: make(map[string]*ChatSession)}
}

func (cm *ChatManager) CreateChatSession(participants []string, duration time.Duration) (*ChatSession, *utils.ErrorResponse) {
	session := &ChatSession{
		ID:           utils.GenerateSessionID(),
		Participants: participants,
		StartTime:    utils.GetTimestamp(),
		EndTime:      utils.GetTimestamp().Add(duration),
		Messages:     []ChatMessage{},
	}

	cm.mu.Lock()
	cm.sessions[session.ID] = session
	cm.mu.Unlock()

	//Auto terminate the session after the duration
	go func() {
		time.Sleep(duration)
		cm.mu.Lock()
		delete(cm.sessions, session.ID)
		cm.mu.Unlock()
	}()

	return session, nil

}

func (cm *ChatManager) AddMessage(sessionID string, message ChatMessage) *utils.ErrorResponse {
	cm.mu.Lock()
    defer cm.mu.Unlock()

    session, exists := cm.sessions[sessionID]
    if!exists {
        return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
    }

	session.mu.Lock()
    session.Messages = append(session.Messages, message)
	session.mu.Unlock()

    return nil
}


func (cm *ChatManager) GetChatMessages(sessionID string) ([]ChatMessage, *utils.ErrorResponse) {
	cm.mu.Lock()
    defer cm.mu.Unlock()

    session, exists := cm.sessions[sessionID]
    if!exists {
        return nil, utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
    }

    return session.Messages, nil
}

func (cm *ChatManager) GetParticipants(sessionID string) ([]string, *utils.ErrorResponse) {
	cm.mu.Lock()
    defer cm.mu.Unlock()

    session, exists := cm.sessions[sessionID]
    if!exists {
        return nil, utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
    }

    return session.Participants, nil
}

func (cm *ChatManager) GetActiveSessions() ([]string, *utils.ErrorResponse) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	var activeSessions []string
	for sessionID, session := range cm.sessions {
        if time.Now().Before(session.EndTime) {
            activeSessions = append(activeSessions, sessionID)
        }
    }

	return activeSessions, nil
}

func (cm *ChatManager) TerminateSession(sessionID string) *utils.ErrorResponse {
	cm.mu.Lock()
    defer cm.mu.Unlock()

    _, exists := cm.sessions[sessionID]
    if!exists {
        return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
    }

    delete(cm.sessions, sessionID)
    return nil
}