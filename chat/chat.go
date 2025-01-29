package chat

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pion-webrtc-microservice/utils"
)

// MessageType defines the type of message
type MessageType string

const (
	TextMessage     MessageType = "text"
	ImageMessage    MessageType = "image"
	FileMessage     MessageType = "file"
	DocumentMessage MessageType = "document"
	EmojiMessage    MessageType = "emoji"
	SystemMessage   MessageType = "system"
)

// ParticipantRole defines the role of a participant
type ParticipantRole string

const (
	RoleAdmin     ParticipantRole = "admin"
	RoleModerator ParticipantRole = "moderator"
	RoleUser      ParticipantRole = "user"
)

// ChatMessage represents a message in the chat
type ChatMessage struct {
	ID          string       `json:"id"`
	SenderID    string       `json:"senderId"`
	ReceiverID  string       `json:"receiverId"`
	Type        MessageType  `json:"type"`
	Message     string       `json:"message"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Reactions   []Reaction   `json:"reactions,omitempty"`
	Timestamp   time.Time    `json:"timestamp"`
	IsEdited    bool         `json:"isEdited"`
	IsDeleted   bool         `json:"isDeleted"`
}

// Participant represents a user in a chat session
type Participant struct {
	ID       string          `json:"id"`
	Role     ParticipantRole `json:"role"`
	IsPinned bool            `json:"isPinned"`
	IsMuted  bool            `json:"isMuted"`
	JoinTime time.Time       `json:"joinTime"`
}

// ChatSession represents a chat session
type ChatSession struct {
	ID           string                  `json:"id"`
	Participants map[string]*Participant `json:"participants"`
	StartTime    time.Time               `json:"startTime"`
	EndTime      time.Time               `json:"endTime"`
	Messages     []ChatMessage           `json:"messages"`
	IsGroup      bool                    `json:"isGroup"`
	mu           sync.Mutex
}

// ChatManager manages all chat sessions
type ChatManager struct {
	sessions map[string]*ChatSession
	Hub      *NotificationHub
	mu       sync.Mutex
}

func NewChatManager() *ChatManager {
	cm := &ChatManager{
		sessions: make(map[string]*ChatSession),
		Hub:      NewNotificationHub(),
	}
	go cm.Hub.Run()
	return cm
}

// CreateChatSession creates a new chat session with roles
func (cm *ChatManager) CreateChatSession(creatorID string, participants []string, duration time.Duration, isGroup bool) (*ChatSession, *utils.ErrorResponse) {
	participantsMap := make(map[string]*Participant)

	// Add creator as admin
	participantsMap[creatorID] = &Participant{
		ID:       creatorID,
		Role:     RoleAdmin,
		JoinTime: utils.GetTimestamp(),
	}

	// Add other participants as users
	for _, pid := range participants {
		if pid != creatorID {
			participantsMap[pid] = &Participant{
				ID:       pid,
				Role:     RoleUser,
				JoinTime: utils.GetTimestamp(),
			}
		}
	}

	session := &ChatSession{
		ID:           utils.GenerateSessionID(),
		Participants: participantsMap,
		StartTime:    utils.GetTimestamp(),
		EndTime:      utils.GetTimestamp().Add(duration),
		Messages:     []ChatMessage{},
		IsGroup:      isGroup,
	}

	cm.mu.Lock()
	cm.sessions[session.ID] = session
	cm.mu.Unlock()

	// Auto terminate the session after duration
	go func() {
		time.Sleep(duration)
		cm.TerminateSession(session.ID)
	}()

	return session, nil
}

// ModifyParticipantRole changes a participant's role
func (cm *ChatManager) ModifyParticipantRole(sessionID, adminID, participantID string, newRole ParticipantRole) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Verify admin has permission
	admin, exists := session.Participants[adminID]
	if !exists || admin.Role != RoleAdmin {
		return utils.NewErrorResponse(http.StatusForbidden, "unauthorized to modify roles")
	}

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	participant.Role = newRole
	return nil
}

// ToggleParticipantPin toggles the pinned status of a participant
func (cm *ChatManager) ToggleParticipantPin(sessionID, participantID string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	participant.IsPinned = !participant.IsPinned
	return nil
}

// AddMessage adds a message with support for different types and attachments
func (cm *ChatManager) AddMessage(sessionID string, message ChatMessage) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Verify sender is a participant
	if _, exists := session.Participants[message.SenderID]; !exists {
		return utils.NewErrorResponse(http.StatusForbidden, "sender is not a participant")
	}

	// Verify message type is valid
	switch message.Type {
	case TextMessage, ImageMessage, FileMessage, DocumentMessage, EmojiMessage, SystemMessage:
		// Valid message type
	default:
		return utils.NewErrorResponse(http.StatusBadRequest, "invalid message type")
	}

	message.ID = utils.GenerateSessionID()
	message.Timestamp = utils.GetTimestamp()
	session.Messages = append(session.Messages, message)

	// Save session after adding message
	if err := cm.SaveSession(session); err != nil {
		return utils.NewErrorResponse(http.StatusInternalServerError, "failed to persist message")
	}

	// Send notification
	cm.Hub.SendNotification(Notification{
		Type:      MessageNotification,
		SessionID: sessionID,
		Data:      message,
	})

	return nil
}

func (cm *ChatManager) GetChatMessages(sessionID string) ([]ChatMessage, *utils.ErrorResponse) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return nil, utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	return session.Messages, nil
}

func (cm *ChatManager) GetParticipants(sessionID string) ([]string, *utils.ErrorResponse) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return nil, utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	participants := make([]string, 0, len(session.Participants))
	for id := range session.Participants {
		participants = append(participants, id)
	}

	return participants, nil
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
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	delete(cm.sessions, sessionID)
	return nil
}

func (cm *ChatManager) AddAttachment(sessionID string, messageID string, attachment Attachment) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	for i, msg := range session.Messages {
		if msg.ID == messageID {
			session.Messages[i].Attachments = append(session.Messages[i].Attachments, attachment)
			return nil
		}
	}

	return utils.NewErrorResponse(http.StatusNotFound, "message not found")
}

func (cm *ChatManager) AddReaction(sessionID string, messageID string, reaction Reaction) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	for i, msg := range session.Messages {
		if msg.ID == messageID {
			session.Messages[i].Reactions = append(session.Messages[i].Reactions, reaction)
			return nil
		}
	}

	return utils.NewErrorResponse(http.StatusNotFound, "message not found")
}

func (cm *ChatManager) PinParticipant(sessionID string, participantID string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	participant.IsPinned = !participant.IsPinned
	return nil
}

func (cm *ChatManager) ModerateParticipant(sessionID string, participantID string, action string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	switch action {
	case "mute":
		participant.IsMuted = true
	case "unmute":
		participant.IsMuted = false
	case "remove":
		delete(session.Participants, participantID)
	default:
		return utils.NewErrorResponse(http.StatusBadRequest, "invalid moderation action")
	}

	// Save session after moderation
	if err := cm.SaveSession(session); err != nil {
		return utils.NewErrorResponse(http.StatusInternalServerError, "failed to persist moderation action")
	}

	// Send notification
	cm.Hub.SendNotification(Notification{
		Type:      ModerationNotification,
		SessionID: sessionID,
		Data: map[string]interface{}{
			"participantId": participantID,
			"action":        action,
		},
	})

	return nil
}

type UsageMetrics struct {
	SessionDuration time.Duration
	MessageCount    int
	AttachmentSize  int64
}

func (cm *ChatManager) GetSessionUsage(sessionID string) (*UsageMetrics, *utils.ErrorResponse) {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return nil, utils.NewErrorResponse(http.StatusNotFound, "chat session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	metrics := &UsageMetrics{
		SessionDuration: time.Since(session.StartTime),
		MessageCount:    len(session.Messages),
	}

	for _, msg := range session.Messages {
		for _, attachment := range msg.Attachments {
			metrics.AttachmentSize += attachment.Size
		}
	}

	return metrics, nil
}

// Add these methods for persistence
func (cm *ChatManager) SaveSession(session *ChatSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	path := filepath.Join("data", "sessions", session.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (cm *ChatManager) LoadSession(sessionID string) (*ChatSession, error) {
	path := filepath.Join("data", "sessions", sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session ChatSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}
