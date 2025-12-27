//go:build opus
// +build opus

package encoder

import (
	"fmt"

	"github.com/code-100-precent/LingFramework/pkg/media"
	"github.com/hraban/opus"
)

// createOPUSDecode creates an OPUS decoder
// OPUS standard sample rate is 48000Hz, but also supports 8000, 12000, 16000, 24000, 48000
func createOPUSDecode(src, pcm media.CodecConfig) media.EncoderFunc {
	// Use configured sample rate, default to 48000Hz (OPUS standard) if not set
	sourceSampleRate := src.SampleRate
	if sourceSampleRate == 0 {
		sourceSampleRate = 48000 // OPUS standard sample rate
	}

	// Determine number of channels
	channels := src.Channels
	if channels == 0 {
		channels = 1 // Default to mono
	}

	// Create OPUS decoder
	// OPUS supported sample rates: 8000, 12000, 16000, 24000, 48000
	decoder, err := opus.NewDecoder(sourceSampleRate, channels)
	if err != nil {
		panic(fmt.Errorf("failed to create opus decoder: %w", err))
	}

	// Create resampler
	res := media.DefaultResampler(sourceSampleRate, pcm.SampleRate)

	// Parse frame duration from FrameDuration (e.g., "20ms", "60ms")
	frameDurationMs := 20 // Default 20ms
	if src.FrameDuration != "" {
		// Parse format like "20ms", "40ms", "60ms"
		var ms int
		if _, err := fmt.Sscanf(src.FrameDuration, "%dms", &ms); err == nil && ms > 0 {
			frameDurationMs = ms
		}
	}

	// Calculate samples per frame
	frameSize := sourceSampleRate * frameDurationMs / 1000

	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}

		// Decode OPUS data to PCM (int16)
		// Create output buffer
		pcmBuffer := make([]int16, frameSize*channels)
		n, err := decoder.Decode(audioPacket.Payload, pcmBuffer)
		if err != nil {
			return nil, fmt.Errorf("opus decode error: %w", err)
		}

		// Convert int16 to []byte
		decodedData := make([]byte, n*channels*2)
		for i := 0; i < n*channels; i++ {
			decodedData[i*2] = byte(pcmBuffer[i])
			decodedData[i*2+1] = byte(pcmBuffer[i] >> 8)
		}

		// Resample to target sample rate
		if _, err = res.Write(decodedData); err != nil {
			return nil, err
		}

		data := res.Samples()
		if data == nil {
			return nil, nil
		}

		audioPacket.Payload = data
		return []media.MediaPacket{audioPacket}, nil
	}
}

// createOPUSEncode creates an OPUS encoder
func createOPUSEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	// Use configured target sample rate, default to 48000Hz (OPUS standard) if not set
	targetSampleRate := src.SampleRate
	if targetSampleRate == 0 {
		targetSampleRate = 48000 // OPUS standard sample rate
	}

	// Validate sample rate is a supported OPUS value
	validRates := []int{8000, 12000, 16000, 24000, 48000}
	isValid := false
	for _, rate := range validRates {
		if targetSampleRate == rate {
			isValid = true
			break
		}
	}
	if !isValid {
		// If not a valid sample rate, use the closest valid value
		if targetSampleRate < 10000 {
			targetSampleRate = 8000
		} else if targetSampleRate < 14000 {
			targetSampleRate = 12000
		} else if targetSampleRate < 20000 {
			targetSampleRate = 16000
		} else if targetSampleRate < 36000 {
			targetSampleRate = 24000
		} else {
			targetSampleRate = 48000
		}
	}

	// Determine number of channels
	channels := src.Channels
	if channels == 0 {
		channels = 1 // Default to mono
	}

	// Create OPUS encoder
	// Use AppAudio mode for better audio quality (instead of AppVoIP)
	encoder, err := opus.NewEncoder(targetSampleRate, channels, opus.AppAudio)
	if err != nil {
		panic(fmt.Errorf("failed to create opus encoder: %w", err))
	}

	// Disable DTX (Discontinuous Transmission) - must be done before setting bitrate
	// DTX generates very small packets (8 bytes) when silence is detected, but hardware may not support it
	// Disabling DTX ensures all frames are fully encoded
	if err := encoder.SetDTX(false); err != nil {
		panic(fmt.Errorf("failed to disable opus DTX: %w", err))
	}

	// Use maximum bitrate to ensure even silent segments are encoded as normal-sized packets
	// This avoids OPUS generating DTX packets
	if err := encoder.SetBitrateToMax(); err != nil {
		panic(fmt.Errorf("failed to set opus bitrate to max: %w", err))
	}

	// Set complexity to 10 (highest quality, 0-10)
	// Higher complexity improves audio quality but increases CPU usage
	if err := encoder.SetComplexity(10); err != nil {
		panic(fmt.Errorf("failed to set opus complexity: %w", err))
	}

	// Create resampler
	res := media.DefaultResampler(pcm.SampleRate, targetSampleRate)

	// Parse frame duration from FrameDuration (e.g., "20ms", "60ms")
	frameDurationMs := 20 // Default 20ms
	if src.FrameDuration != "" {
		// Parse format like "20ms", "40ms", "60ms"
		var ms int
		if _, err := fmt.Sscanf(src.FrameDuration, "%dms", &ms); err == nil && ms > 0 {
			frameDurationMs = ms
		}
	}

	// Calculate samples per frame
	frameSize := targetSampleRate * frameDurationMs / 1000

	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}

		// Resample to OPUS target sample rate
		if _, err := res.Write(audioPacket.Payload); err != nil {
			return nil, err
		}

		data := res.Samples()
		if data == nil {
			return nil, nil
		}

		// Convert []byte to []int16
		pcmSamples := make([]int16, len(data)/2)
		for i := 0; i < len(pcmSamples); i++ {
			pcmSamples[i] = int16(data[i*2]) | int16(data[i*2+1])<<8
		}

		// Calculate samples needed per frame
		samplesPerFrame := frameSize * channels
		totalSamples := len(pcmSamples)

		// If data is less than one frame, pad with silence
		if totalSamples < samplesPerFrame {
			paddedSamples := make([]int16, samplesPerFrame)
			copy(paddedSamples, pcmSamples)
			pcmSamples = paddedSamples
			totalSamples = samplesPerFrame
		}

		// Only encode the first frame (caller is responsible for frame splitting)
		opusBuffer := make([]byte, 4000) // Large enough buffer
		n, err := encoder.Encode(pcmSamples[:samplesPerFrame], opusBuffer)
		if err != nil {
			return nil, fmt.Errorf("opus encode error: %w", err)
		}

		audioPacket.Payload = opusBuffer[:n]
		return []media.MediaPacket{audioPacket}, nil
	}
}
