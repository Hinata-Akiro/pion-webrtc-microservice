package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"pion-webrtc-microservice/peer"
	"pion-webrtc-microservice/utils" // Import the utils package
	"github.com/pion/webrtc/v3"
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