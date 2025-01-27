# WebRTC Microservice

A lightweight WebRTC signaling server built with Go and Pion WebRTC. This microservice handles SDP offer/answer exchange and ICE candidate management for establishing peer-to-peer connections.

---

## Table of Contents
1. [Features](#features)
2. [Prerequisites](#prerequisites)
3. [Installation](#installation)
4. [API Documentation](#api-documentation)
   - [Health Check](#health-check)
   - [Create SDP Offer](#create-sdp-offer)
   - [Add ICE Candidate](#add-ice-candidate)


---

## Features
- **SDP Offer/Answer Exchange**: Handles WebRTC SDP offer and answer exchange.
- **ICE Candidate Management**: Manages ICE candidates for establishing peer connections.
- **Health Check**: Provides a health check endpoint to verify server status.
- **Scalable**: Uses a `PeerManager` to manage multiple peer connections concurrently.

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
#### Endpoint: `GET /health`

**Description:**  
Checks the health of the server.

**Response:**
```json
{
  "status_code": 200,
  "message": "Server is healthy"
}
```

### Create SDP Offer
#### Endpoint: POST /offer?peerID=<peerID>

**Query Parameters:**
- `peerID`: A unique identifier for the peer connection (required).

**Description:** 
Accepts an SDP offer from a client and returns an SDP answer.

**Request Body**:
```json
{
  "type": "offer",
  "sdp": "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n..."
}
```

**Success Response**:
```json
{
  "status_code": 200,
  "message": "answer created successfully",
  "data": {
    "type": "answer",
    "sdp": "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n..."
  }
}
```

### Add ICE Candidate
#### Endpoint: `POST /ice-candidate?peerID=<peerID>`

**Description:**  
Adds an ICE candidate to a peer connection.

**Query Parameters:**
- `peerID`: A unique identifier for the peer connection (required).

**Request Body:**
```json
{
  "candidate": "candidate:1234567890 1 udp 2122194687 192.168.1.1 12345 typ host",
  "sdpMid": "0",
  "sdpMLineIndex": 0
}
```


