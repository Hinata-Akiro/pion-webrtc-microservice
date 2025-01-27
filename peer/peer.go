package peer

import (
	"net/http"
	"sync"

	"pion-webrtc-microservice/utils"

	"github.com/pion/webrtc/v3"
)

// PeerConnectionState represents the state of the peer connection
type PeerConnectionState struct {
	PeerConnection *webrtc.PeerConnection
	Mutex          sync.Mutex
}

// PeerManager manages all active peer connections
type PeerManager struct {
	peerConnections map[string]*PeerConnectionState
	mutex           sync.Mutex
}

// NewPeerManager creates a new PeerManager
func NewPeerManager() *PeerManager {
	return &PeerManager{peerConnections: make(map[string]*PeerConnectionState)}
}

// CreatePeerConnection creates a new peer connection
func (pm *PeerManager) CreatePeerConnection(peerID string) (*PeerConnectionState, *utils.ErrorResponse) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.peerConnections[peerID]; exists {
		return nil, utils.NewErrorResponse(http.StatusConflict, "peer connection already exists")
	}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{
			URLs: []string{"stun:stun.l.google.com:19302"},
		}},
	})
	if err != nil {
		return nil, utils.NewErrorResponse(http.StatusInternalServerError, err.Error())
	}

	pm.peerConnections[peerID] = &PeerConnectionState{PeerConnection: peerConnection}
	return pm.peerConnections[peerID], nil
}

// GetPeerConnection retrieves a peer connection by ID
func (pm *PeerManager) GetPeerConnection(peerID string) (*webrtc.PeerConnection, *utils.ErrorResponse) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	peerConnection, exists := pm.peerConnections[peerID]
	if !exists {
		return nil, utils.NewErrorResponse(http.StatusNotFound, "peer connection not found")
	}

	return peerConnection.PeerConnection, nil
}

// ClosePeerConnection closes a peer connection by ID
func (pm *PeerManager) ClosePeerConnection(peerID string) *utils.ErrorResponse {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	peerConnection, exists := pm.peerConnections[peerID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "peer connection not found")
	}

	err := peerConnection.PeerConnection.Close()
	if err != nil {
		return utils.NewErrorResponse(http.StatusInternalServerError, err.Error())
	}

	delete(pm.peerConnections, peerID)
	return nil
}

// AddICECandidate adds an ICE candidate to a peer connection
func (pm *PeerManager) AddICECandidate(peerID string, candidate webrtc.ICECandidateInit) *utils.ErrorResponse {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	peerConnection, exists := pm.peerConnections[peerID]
	if !exists {
		return utils.NewErrorResponse(http.StatusNotFound, "peer connection not found")
	}

	err := peerConnection.PeerConnection.AddICECandidate(candidate)
	if err != nil {
		return utils.NewErrorResponse(http.StatusInternalServerError, err.Error())
	}

	return nil
}
