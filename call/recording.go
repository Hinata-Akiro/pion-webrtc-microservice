package call

import (
	"bytes"
	"io"

	"github.com/pion/webrtc/v3"
)

type MediaRecorder struct {
	audioWriter   io.Writer
	videoWriter   io.Writer
	audioTrack    *webrtc.TrackLocalStaticSample
	videoTrack    *webrtc.TrackLocalStaticSample
	isRecording   bool
	stopRecording chan struct{}
}

func NewMediaRecorder() *MediaRecorder {
	return &MediaRecorder{
		audioWriter:   &bytes.Buffer{},
		videoWriter:   &bytes.Buffer{},
		stopRecording: make(chan struct{}),
	}
}

func (mr *MediaRecorder) Start(pc *webrtc.PeerConnection) error {
	// Create audio track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"},
		"audio", "recording",
	)
	if err != nil {
		return err
	}
	mr.audioTrack = audioTrack

	// Create video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "video/vp8"},
		"video", "recording",
	)
	if err != nil {
		return err
	}
	mr.videoTrack = videoTrack

	// Add tracks to peer connection
	if _, err = pc.AddTrack(audioTrack); err != nil {
		return err
	}
	if _, err = pc.AddTrack(videoTrack); err != nil {
		return err
	}

	mr.isRecording = true
	return nil
}

func (mr *MediaRecorder) Stop() {
	if mr.isRecording {
		close(mr.stopRecording)
		mr.isRecording = false
	}
}
