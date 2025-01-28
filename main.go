package main

import (
	"net/http"
	"time"

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
	chatManger  = chat.NewChatManager()
	signalingManger = signaling.NewSignalingServer()
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

//websocket handler for signaling 
func  handleWebSocket(c echo.Context) error {

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

func createChatSession( c echo.Context) error {
	var request  struct {
		Participants []string `json:"participants"`
		Duration     time.Duration      `json:"duration"`
	}
	if err := c.Bind(&request); err!= nil {
        return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
    }
	session, errResp := chatManger.CreateChatSession(request.Participants, request.Duration)
	if errResp != nil {
		return c.JSON(errResp.StatusCode, errResp)
	}
	return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "chat session created successfully", session))
}

func sendChatMessage(c echo.Context) error {
	var request  struct {
        SessionID string `json:"sessionID"`
		SenderID  string `json:"senderID"`
		ReceiverID string `json:"receiverID"`
        Message   string `json:"message"`
    }
    if err := c.Bind(&request); err!= nil {
        return c.JSON(http.StatusBadRequest, utils.NewErrorResponse(http.StatusBadRequest, "invalid request"))
    }

	message := chat.ChatMessage{
		ID : utils.GenerateSessionID(),
		SenderID: request.SenderID,
        ReceiverID: request.ReceiverID,
        Message: request.Message,
        Timestamp: utils.GetTimestamp(),
	}
    errResp := chatManger.AddMessage(request.SessionID, message)
    if errResp!= nil {
        return c.JSON(errResp.StatusCode, errResp)
    }
    return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "message sent successfully", nil))
}

func getChatMessages(c echo.Context) error {
	sessionID := c.Param("sessionID")

    messages, errResp := chatManger.GetChatMessages(sessionID)
    if errResp!= nil {
        return c.JSON(errResp.StatusCode, errResp)
    }

    return c.JSON(http.StatusOK, utils.NewSuccessResponse(http.StatusOK, "messages retrieved successfully", messages))
}