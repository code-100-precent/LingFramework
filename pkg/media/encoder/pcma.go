package encoder

import (
	"github.com/code-100-precent/LingFramework/pkg/media"
)

func createPCMADecode(src, pcm media.CodecConfig) media.EncoderFunc {
	sourceSampleRate := src.SampleRate
	if sourceSampleRate == 0 {
		sourceSampleRate = 8000
	}
	res := media.DefaultResampler(sourceSampleRate, pcm.SampleRate)
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		data, err := pcma2pcm(audioPacket.Payload)
		if err != nil {
			return nil, err
		}
		if _, err = res.Write(data); err != nil {
			return nil, err
		}
		data = res.Samples()
		if data == nil {
			return nil, nil
		}
		audioPacket.Payload = data
		return []media.MediaPacket{audioPacket}, nil
	}
}

func createPCMAEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	// Use configured target sample rate, default to 8000Hz (PCMA standard) if not set
	targetSampleRate := src.SampleRate
	if targetSampleRate == 0 {
		targetSampleRate = 8000 // PCMA standard sample rate
	}
	res := media.DefaultResampler(pcm.SampleRate, targetSampleRate)

	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		if _, err := res.Write(audioPacket.Payload); err != nil {
			return nil, err
		}
		data := res.Samples()
		if data == nil {
			return nil, nil
		}
		data, err := Pcm2pcma(data)
		if err != nil {
			return nil, err
		}
		return splitFrames(data, &src), nil
	}
}
