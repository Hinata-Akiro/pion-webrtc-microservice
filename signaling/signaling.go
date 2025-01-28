package signaling


import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)


type SignalingServer struct {
	clients      map[string]*websocket.Conn
	mutex        sync.Mutex
}


func NewSignalingServer() *SignalingServer {
    return &SignalingServer{clients: make(map[string]*websocket.Conn)}
}

func (s *SignalingServer) HandleWebSocket(conn *websocket.Conn, peerID string) {
	s.mutex.Lock()
	s.clients[peerID] = conn
	s.mutex.Unlock()

	defer func ()  {
		s.mutex.Lock()
		delete(s.clients, peerID)
		s.mutex.Unlock()
		conn.Close()
	}()


	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
           log.Println("Error reading message:", err)
		   break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Panicln("Error decoding message:", err)
			continue
		}

		s.handleSignalMessage(peerID,msg)
	}
}

func (s *SignalingServer) handleSignalMessage(peerID string, msg map[string]interface{}) {
	targetPeerId, ok := msg["targetPeerId"].(string)
	if !ok {
		log.Println("targetPeerId is not a string")
		return
	}

	s.mutex.Lock()
	targetConn, exists := s.clients[targetPeerId]
	s.mutex.Unlock()

	if !exists {
		log.Printf("No client found for peerId: %s\n", targetPeerId)
        return
	}

	if err := targetConn.WriteJSON(msg); err != nil {
		log.Printf("Error writing to client for peerId: %s, error: %v\n", targetPeerId, err)
        return
	}
}