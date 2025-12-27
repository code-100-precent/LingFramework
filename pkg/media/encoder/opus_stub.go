//go:build !opus
// +build !opus

package encoder

import (
	"fmt"

	"github.com/code-100-precent/LingFramework/pkg/media"
)

// createOPUSDecode creates a stub OPUS decoder that returns an error
// This file is compiled when the opus build tag is not set
func createOPUSDecode(src, pcm media.CodecConfig) media.EncoderFunc {
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		return nil, fmt.Errorf("OPUS codec support is not available. Build with -tags opus to enable OPUS support")
	}
}

// createOPUSEncode creates a stub OPUS encoder that returns an error
// This file is compiled when the opus build tag is not set
func createOPUSEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		return nil, fmt.Errorf("OPUS codec support is not available. Build with -tags opus to enable OPUS support")
	}
}
