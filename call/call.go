package call

import (
	"net/http"
	"sync"
	"time"

	"pion-webrtc-microservice/utils"

	"github.com/pion/webrtc/v3"
)

type CallType string
type CallQuality string
type ParticipantStatus string

const (
	VideoCall CallType = "video"
	AudioCall CallType = "audio"

	QualitySD CallQuality = "sd"
	QualityHD CallQuality = "hd"
	Quality4K CallQuality = "4k"

	StatusWaiting   ParticipantStatus = "waiting"
	StatusConnected ParticipantStatus = "connected"
	StatusLeft      ParticipantStatus = "left"
)

type CallParticipant struct {
	ID             string
	PeerConnection *webrtc.PeerConnection
	Status         ParticipantStatus
	IsMuted        bool
	IsVideoEnabled bool
	IsSpeaking     bool
	NetworkQuality int // 1-5 scale
	JoinTime       time.Time
	AudioDetector  *AudioLevelDetector
	MediaRecorder  *MediaRecorder
	mu             sync.Mutex
}

type CallSession struct {
	ID              string
	Type            CallType
	Quality         CallQuality
	URL             string
	Participants    map[string]*CallParticipant
	CreatorID       string
	StartTime       time.Time
	EndTime         time.Time
	IsRecording     bool
	IsLivestreaming bool
	InLobby         []string 
	mu              sync.Mutex
}

type CallManager struct {
	sessions map[string]*CallSession
	mu       sync.Mutex
}

func NewCallManager() *CallManager {
	return &CallManager{
		sessions: make(map[string]*CallSession),
	}
}

func (cm *CallManager) CreateCallSession(creatorID string, callType CallType, quality CallQuality, duration time.Duration) (*CallSession, *utils.ErrorResponse) {
	session := &CallSession{
		ID:           utils.GenerateSessionID(),
		Type:         callType,
		Quality:      quality,
		URL:          "/call/" + utils.GenerateSessionID(),
		Participants: make(map[string]*CallParticipant),
		CreatorID:    creatorID,
		StartTime:    utils.GetTimestamp(),
		EndTime:      utils.GetTimestamp().Add(duration),
	}

	// Add creator as first participant
	session.Participants[creatorID] = &CallParticipant{
		ID:       creatorID,
		Status:   StatusConnected,
		JoinTime: utils.GetTimestamp(),
	}

	cm.mu.Lock()
	cm.sessions[session.ID] = session
	cm.mu.Unlock()

	// Auto terminate
	go func() {
		time.Sleep(duration)
		cm.TerminateSession(session.ID)
	}()

	return session, nil
}

func (cm *CallManager) JoinCall(sessionID, participantID string, pc *webrtc.PeerConnection) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if participant is in lobby
	for i, id := range session.InLobby {
		if id == participantID {
			// Remove from lobby
			session.InLobby = append(session.InLobby[:i], session.InLobby[i+1:]...)
			break
		}
	}

	session.Participants[participantID] = &CallParticipant{
		ID:             participantID,
		PeerConnection: pc,
		Status:         StatusConnected,
		JoinTime:       utils.GetTimestamp(),
		NetworkQuality: 5, // Start with best quality
	}

	// Setup media tracks
	if session.Type == VideoCall {
		if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo,
			webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv}); err != nil {
			return utils.NewErrorResponse(http.StatusInternalServerError, "failed to add video transceiver")
		}
	}

	if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv}); err != nil {
		return utils.NewErrorResponse(http.StatusInternalServerError, "failed to add audio transceiver")
	}

	return nil
}

func (cm *CallManager) AddToLobby(sessionID, participantID string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.InLobby = append(session.InLobby, participantID)
	return nil
}

func (cm *CallManager) ToggleMute(sessionID, participantID string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	participant.mu.Lock()
	participant.IsMuted = !participant.IsMuted
	participant.mu.Unlock()

	return nil
}

func (cm *CallManager) UpdateNetworkQuality(sessionID, participantID string, quality int) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	participant.mu.Lock()
	participant.NetworkQuality = quality
	participant.mu.Unlock()

	return nil
}

func (cm *CallManager) ToggleRecording(sessionID string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	session.IsRecording = !session.IsRecording
	session.mu.Unlock()

	return nil
}

func (cm *CallManager) TerminateSession(sessionID string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	// Close all peer connections
	for _, participant := range session.Participants {
		if participant.PeerConnection != nil {
			participant.PeerConnection.Close()
		}
	}
	session.mu.Unlock()

	cm.mu.Lock()
	delete(cm.sessions, sessionID)
	cm.mu.Unlock()

	return nil
}

func (cm *CallManager) StartRecording(sessionID, participantID string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	participant.mu.Lock()
	defer participant.mu.Unlock()

	if participant.MediaRecorder == nil {
		participant.MediaRecorder = NewMediaRecorder()
	}

	err := participant.MediaRecorder.Start(participant.PeerConnection)
	if err != nil {
		return utils.NewErrorResponse(http.StatusInternalServerError, "failed to start recording")
	}

	return nil
}

func (cm *CallManager) ProcessAudioLevel(sessionID, participantID string, sample []byte) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	participant.mu.Lock()
	defer participant.mu.Unlock()

	if participant.AudioDetector == nil {
		participant.AudioDetector = NewAudioLevelDetector()
	}

	participant.AudioDetector.ProcessAudioLevel(sample)
	participant.IsSpeaking = participant.AudioDetector.IsSpeaking()

	return nil
}

func (cm *CallManager) StopRecording(sessionID, participantID string) *utils.ErrorResponse {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	cm.mu.Unlock()

	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	participant, exists := session.Participants[participantID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "participant not found")
	}

	participant.mu.Lock()
	defer participant.mu.Unlock()

	if participant.MediaRecorder != nil {
		participant.MediaRecorder.Stop()
	}

	return nil
}

func (cm *CallManager) GetCallSession(sessionID string) (*CallSession, *utils.ErrorResponse) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return nil, utils.NewErrorResponse(http.StatusNotFound, "call session not found")
	}

	return session, nil
}
