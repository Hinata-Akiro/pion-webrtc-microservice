package call

import (
	"encoding/binary"
	"math"
	"time"
)

type AudioLevel struct {
	Level     float64 
	Timestamp time.Time
}

type AudioLevelDetector struct {
	levels       []AudioLevel
	threshold    float64
	windowSize   time.Duration
	isSpeaking   bool
	lastUpdate   time.Time
	speakingTime time.Duration
}

func NewAudioLevelDetector() *AudioLevelDetector {
	return &AudioLevelDetector{
		threshold:  0.3, 
		windowSize: 500 * time.Millisecond,
	}
}

func (d *AudioLevelDetector) ProcessAudioLevel(sample []byte) {
	var sum float64
	for i := 0; i < len(sample); i += 2 {
		value := float64(int16(binary.LittleEndian.Uint16(sample[i:])))
		sum += math.Abs(value)
	}
	level := sum / float64(len(sample)/2) / 32768.0 

	d.levels = append(d.levels, AudioLevel{
		Level:     level,
		Timestamp: time.Now(),
	})

	// Remove old levels
	cutoff := time.Now().Add(-d.windowSize)
	for i, level := range d.levels {
		if level.Timestamp.After(cutoff) {
			d.levels = d.levels[i:]
			break
		}
	}

	// Update speaking status
	avgLevel := d.getAverageLevel()
	wasSpeaking := d.isSpeaking
	d.isSpeaking = avgLevel > d.threshold

	if d.isSpeaking && !wasSpeaking {
		d.lastUpdate = time.Now()
	} else if !d.isSpeaking && wasSpeaking {
		d.speakingTime += time.Since(d.lastUpdate)
	}
}

func (d *AudioLevelDetector) getAverageLevel() float64 {
	if len(d.levels) == 0 {
		return 0
	}

	var sum float64
	for _, level := range d.levels {
		sum += level.Level
	}
	return sum / float64(len(d.levels))
}

func (d *AudioLevelDetector) IsSpeaking() bool {
	return d.isSpeaking
}

func (d *AudioLevelDetector) GetSpeakingTime() time.Duration {
	if d.isSpeaking {
		return d.speakingTime + time.Since(d.lastUpdate)
	}
	return d.speakingTime
}
