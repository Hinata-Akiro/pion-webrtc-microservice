# WebRTC Microservice

A lightweight WebRTC signaling server built with Go and Pion WebRTC. This microservice handles WebRTC signaling, chat functionality, and video/audio calls with features like recording and quality management.

---

## Table of Contents
1. [Features](#features)
2. [Prerequisites](#prerequisites)
3. [Installation](#installation)
4. [API Documentation](#api-documentation)
   - [Health Check](#health-check)
   - [WebRTC Endpoints](#webrtc-endpoints)
   - [Chat Endpoints](#chat-endpoints)
   - [Call Endpoints](#call-endpoints)

---

## Features
- **WebRTC Signaling**: Handles SDP offer/answer exchange and ICE candidates
- **Chat System**: Real-time chat with support for text, images, files, and reactions
- **Video/Audio Calls**: Full-featured call system with recording capabilities
- **User Management**: Participant roles, moderation, and session management
- **Real-time Notifications**: WebSocket-based notification system
- **Scalable Architecture**: Concurrent session handling with proper synchronization

---

## Prerequisites
- Go 1.20 or higher
- Git
- Postman or any HTTP client for testing

---

## Installation

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/your-username/pion-webrtc-microservice.git
   cd pion-webrtc-microservice
   ```

2. **Install Dependencies**:
   ```bash
   go mod download
   ```

3. **Run the Server**:
   ```bash
   go run main.go
   ```

## API Documentation

### Health Check
#### `GET /health`
Checks the health of the server.
```json
{
  "status": 200,
  "message": "Server is healthy",
  "data": null
}
```

### WebRTC Endpoints

#### `POST /offer?peerID=<peerID>`
Creates an SDP answer for an offer.
```json
// Request
{
  "type": "offer",
  "sdp": "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n..."
}

// Response
{
  "status": 200,
  "message": "answer created successfully",
  "data": {
    "type": "answer",
    "sdp": "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n..."
  }
}
```

#### `POST /ice-candidate?peerID=<peerID>`
Adds an ICE candidate.
```json
// Request
{
  "candidate": "candidate:1234567890 1 udp 2122194687 192.168.1.1 12345 typ host",
  "sdpMid": "0",
  "sdpMLineIndex": 0
}
```

### Chat Endpoints

#### `POST /chat/session`
Creates a new chat session.
```json
// Request
{
    "creatorId": "user123",
    "participants": ["user456", "user789"],
    "duration": 3600000000000,
    "isGroup": true
}
```

#### `POST /chat/message`
Sends a chat message.
```json
// Request
{
    "sessionID": "sess_abc123",
    "senderID": "user123",
    "receiverID": "user456",
    "message": "Hello!",
    "type": "text"
}
```

#### `POST /chat/attachment`
Adds an attachment to a message.
```json
// Request
{
    "sessionId": "sess_abc123",
    "messageId": "msg_xyz789",
    "attachment": {
        "type": "image",
        "url": "https://example.com/image.jpg",
        "name": "vacation.jpg",
        "size": 1024000,
        "mimeType": "image/jpeg"
    }
}
```

#### `POST /chat/reaction`
Adds a reaction to a message.
```json
// Request
{
    "sessionId": "sess_abc123",
    "messageId": "msg_xyz789",
    "reaction": {
        "type": "👍",
        "userId": "user123",
        "timestamp": "2024-01-29T10:05:00Z"
    }
}
```

#### `GET /chat/messages/:sessionID`
Retrieves messages from a chat session.

#### `GET /chat/usage/:sessionID`
Gets usage metrics for a chat session.

### Call Endpoints

#### `POST /call/session`
Creates a new call session.
```json
// Request
{
    "creatorId": "user123",
    "type": "video",
    "quality": "high",
    "duration": 3600000000000
}
```

#### `POST /call/join`
Joins an existing call.
```json
// Request
{
    "sessionId": "call_abc123",
    "participantId": "user456"
}
```

#### `POST /call/recording/start`
Starts call recording.
```json
// Request
{
    "sessionId": "call_abc123",
    "participantId": "user123"
}
```

#### `POST /call/recording/stop`
Stops call recording.
```json
// Request
{
    "sessionId": "call_abc123",
    "participantId": "user123"
}
```

#### `POST /call/quality`
Updates call quality settings.
```json
// Request
{
    "sessionId": "call_abc123",
    "participantId": "user456",
    "quality": 85
}
```

#### `GET /call/session/:sessionID`
Gets call session details.

### WebSocket Endpoints

#### `GET /ws?peerID=<peerID>`
WebSocket connection for signaling.

#### `GET /chat/notifications`
WebSocket connection for chat notifications.

---

## Error Handling

All endpoints return error responses in the following format:
```json
{
    "status": 400,
    "message": "error description",
    "data": null
}
```

## Rate Limiting

The server implements rate limiting to prevent abuse. Excessive requests will receive a 429 status code.

---

## License
MIT License


