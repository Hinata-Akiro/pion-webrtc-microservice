package main

import (
	"net/http"
	"time"

	"pion-webrtc-microservice/call"
	"pion-webrtc-microservice/chat"
	"pion-webrtc-microservice/peer"
	"pion-webrtc-microservice/signaling"
	"pion-webrtc-microservice/utils"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pion/webrtc/v3"
)

var (
	chatManger      = chat.NewChatManager()
	signalingManger = signaling.NewSignalingServer()
	callManager     = call.NewCallManager()
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	peerManager := peer.NewPeerManager()

	e.POST("/offer", func(c echo.Context) error {
		return handleOffer(c, peerManager)
	})
	e.POST("/ice-candidate", func(c echo.Context) error {
		return handleICECandidate(c, peerManager)
	})

	e.GET("/ws", func(c echo.Context) error {
		return handleWebSocket(c)
	})

	e.POST("/chat/session", func(c echo.Context) error {
		return createChatSession(c)
	})

	e.POST("/chat/message", func(c echo.Context) error {
		return sendChatMessage(c)
	})

	e.GET("/chat/messages/:sessionID", func(c echo.Context) error {
		return getChatMessages(c)
	})

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "Server is healthy", nil))
	})

	e.POST("/call/session", createCallSession)
	e.POST("/call/join", joinCall)
	e.POST("/call/lobby", addToLobby)
	e.POST("/call/mute", toggleMute)
	e.POST("/call/recording", toggleRecording)
	e.POST("/call/quality", updateCallQuality)
	e.GET("/call/session/:sessionID", getCallSession)
	e.POST("/call/recording/start", startRecording)
	e.POST("/call/recording/stop", stopRecording)

	e.POST("/chat/attachment", addChatAttachment)
	e.POST("/chat/reaction", addChatReaction)
	e.POST("/chat/pin", pinParticipant)
	e.POST("/chat/moderate", moderateParticipant)
	e.GET("/chat/usage/:sessionID", getChatUsage)

	e.GET("/chat/notifications", handleChatNotifications)

	e.Logger.Fatal(e.Start(":8000"))
}

func handleOffer(c echo.Context, peerManager *peer.PeerManager) error {
	var offer webrtc.SessionDescription
	if err := c.Bind(&offer); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid offer"))
	}

	peerID := c.QueryParam("peerID")
	if peerID == "" {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "peerID is required"))
	}

	peerConnectionState, errResp := peerManager.CreatePeerConnection(peerID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	if err := peerConnectionState.PeerConnection.SetRemoteDescription(offer); err != nil {
		return c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "failed to set remote description"))
	}

	answer, err := peerConnectionState.PeerConnection.CreateAnswer(nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "failed to create answer"))
	}

	if err := peerConnectionState.PeerConnection.SetLocalDescription(answer); err != nil {
		return c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "failed to set local description"))
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "answer created successfully", answer))
}

func handleICECandidate(c echo.Context, peerManager *peer.PeerManager) error {
	var candidate webrtc.ICECandidateInit
	if err := c.Bind(&candidate); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid ICE candidate"))
	}

	peerID := c.QueryParam("peerID")
	if peerID == "" {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "peerID is required"))
	}

	if errResp := peerManager.AddICECandidate(peerID, candidate); errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "ICE candidate added successfully", nil))
}

// websocket handler for signaling
func handleWebSocket(c echo.Context) error {

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to upgrade connection: "+err.Error()))
	}
	defer ws.Close()

	peerID := c.QueryParam("peerID")
	if peerID == "" {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "peerID is required"))
	}

	signalingManger.HandleWebSocket(ws, peerID)
	return nil
}

func createChatSession(c echo.Context) error {
	var request struct {
		CreatorID    string        `json:"creatorId"`
		Participants []string      `json:"participants"`
		Duration     time.Duration `json:"duration"`
		IsGroup      bool          `json:"isGroup"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}
	session, errResp := chatManger.CreateChatSession(request.CreatorID, request.Participants, request.Duration, request.IsGroup)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}
	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "chat session created successfully", session))
}

func sendChatMessage(c echo.Context) error {
	var request struct {
		SessionID  string `json:"sessionID"`
		SenderID   string `json:"senderID"`
		ReceiverID string `json:"receiverID"`
		Message    string `json:"message"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	message := chat.ChatMessage{
		ID:         utils.GenerateSessionID(),
		SenderID:   request.SenderID,
		ReceiverID: request.ReceiverID,
		Message:    request.Message,
		Timestamp:  utils.GetTimestamp(),
	}
	errResp := chatManger.AddMessage(request.SessionID, message)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}
	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "message sent successfully", nil))
}

func getChatMessages(c echo.Context) error {
	sessionID := c.Param("sessionID")

	messages, errResp := chatManger.GetChatMessages(sessionID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "messages retrieved successfully", messages))
}

func createCallSession(c echo.Context) error {
	var request struct {
		CreatorID string           `json:"creatorId"`
		Type      call.CallType    `json:"type"`
		Quality   call.CallQuality `json:"quality"`
		Duration  time.Duration    `json:"duration"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	session, errResp := callManager.CreateCallSession(request.CreatorID, request.Type, request.Quality, request.Duration)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "call session created", session))
}

func joinCall(c echo.Context) error {
	var request struct {
		SessionID     string `json:"sessionId"`
		ParticipantID string `json:"participantId"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	// Create peer connection
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{
			URLs: []string{"stun:stun.l.google.com:19302"},
		}},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "failed to create peer connection"))
	}

	errResp := callManager.JoinCall(request.SessionID, request.ParticipantID, pc)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "joined call successfully", nil))
}

func addToLobby(c echo.Context) error {
	var request struct {
		SessionID     string `json:"sessionId"`
		ParticipantID string `json:"participantId"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := callManager.AddToLobby(request.SessionID, request.ParticipantID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "added to lobby", nil))
}

func toggleMute(c echo.Context) error {
	var request struct {
		SessionID     string `json:"sessionId"`
		ParticipantID string `json:"participantId"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := callManager.ToggleMute(request.SessionID, request.ParticipantID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "toggled mute", nil))
}

func toggleRecording(c echo.Context) error {
	sessionID := c.QueryParam("sessionId")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "sessionId is required"))
	}

	errResp := callManager.ToggleRecording(sessionID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "toggled recording", nil))
}

func updateCallQuality(c echo.Context) error {
	var request struct {
		SessionID     string `json:"sessionId"`
		ParticipantID string `json:"participantId"`
		Quality       int    `json:"quality"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := callManager.UpdateNetworkQuality(request.SessionID, request.ParticipantID, request.Quality)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "updated call quality", nil))
}

func getCallSession(c echo.Context) error {
	sessionID := c.Param("sessionID")

	session, errResp := callManager.GetCallSession(sessionID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "call session retrieved successfully", session))
}

func startRecording(c echo.Context) error {
	var request struct {
		SessionID     string `json:"sessionId"`
		ParticipantID string `json:"participantId"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := callManager.StartRecording(request.SessionID, request.ParticipantID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "recording started", nil))
}

func stopRecording(c echo.Context) error {
	var request struct {
		SessionID     string `json:"sessionId"`
		ParticipantID string `json:"participantId"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := callManager.StopRecording(request.SessionID, request.ParticipantID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "recording stopped", nil))
}

func addChatAttachment(c echo.Context) error {
	var request struct {
		SessionID  string          `json:"sessionId"`
		MessageID  string          `json:"messageId"`
		Attachment chat.Attachment `json:"attachment"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := chatManger.AddAttachment(request.SessionID, request.MessageID, request.Attachment)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "attachment added", nil))
}

func addChatReaction(c echo.Context) error {
	var request struct {
		SessionID string        `json:"sessionId"`
		MessageID string        `json:"messageId"`
		Reaction  chat.Reaction `json:"reaction"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := chatManger.AddReaction(request.SessionID, request.MessageID, request.Reaction)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "reaction added", nil))
}

func pinParticipant(c echo.Context) error {
	var request struct {
		SessionID     string `json:"sessionId"`
		ParticipantID string `json:"participantId"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := chatManger.PinParticipant(request.SessionID, request.ParticipantID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "participant pinned", nil))
}

func moderateParticipant(c echo.Context) error {
	var request struct {
		SessionID     string `json:"sessionId"`
		ParticipantID string `json:"participantId"`
		Action        string `json:"action"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
	}

	errResp := chatManger.ModerateParticipant(request.SessionID, request.ParticipantID, request.Action)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "participant moderated", nil))
}

func getChatUsage(c echo.Context) error {
	sessionID := c.Param("sessionID")

	usage, errResp := chatManger.GetSessionUsage(sessionID)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}

	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "chat usage retrieved successfully", usage))
}

func handleChatNotifications(c echo.Context) error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.NewErrorResponse(http.StatusInternalServerError, "Failed to upgrade connection"))
	}

	// Register client for notifications
	chatManger.Hub.Register <- ws

	// Handle disconnection
	go func() {
		<-c.Request().Context().Done()
		chatManger.Hub.Unregister <- ws
	}()

	return nil
}
