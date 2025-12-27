package encoder

import (
	"testing"

	"github.com/code-100-precent/LingFramework/pkg/media"
)

func TestRegisterCodec(t *testing.T) {
	// Test registering a new codec
	RegisterCodec("test_codec", PcmToPcm, PcmToPcm)

	if !HasCodec("test_codec") {
		t.Error("expected codec to be registered")
	}

	// Test case-insensitive
	if !HasCodec("TEST_CODEC") {
		t.Error("expected codec lookup to be case-insensitive")
	}
}

func TestCreateEncode(t *testing.T) {
	src := media.CodecConfig{
		Codec:      CodecPCM,
		SampleRate: 16000,
	}
	pcm := media.CodecConfig{
		Codec:      CodecPCM,
		SampleRate: 16000,
	}

	encode, err := CreateEncode(src, pcm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if encode == nil {
		t.Fatal("expected non-nil encoder")
	}
}

func TestCreateEncode_UnsupportedCodec(t *testing.T) {
	src := media.CodecConfig{
		Codec:      "unsupported",
		SampleRate: 16000,
	}
	pcm := media.CodecConfig{
		Codec:      CodecPCM,
		SampleRate: 16000,
	}

	_, err := CreateEncode(src, pcm)
	if err != media.ErrCodecNotSupported {
		t.Errorf("expected ErrCodecNotSupported, got %v", err)
	}
}

func TestCreateDecode(t *testing.T) {
	src := media.CodecConfig{
		Codec:      CodecPCM,
		SampleRate: 16000,
	}
	pcm := media.CodecConfig{
		Codec:      CodecPCM,
		SampleRate: 16000,
	}

	decode, err := CreateDecode(src, pcm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decode == nil {
		t.Fatal("expected non-nil decoder")
	}
}

func TestCreateDecode_UnsupportedCodec(t *testing.T) {
	src := media.CodecConfig{
		Codec:      "unsupported",
		SampleRate: 16000,
	}
	pcm := media.CodecConfig{
		Codec:      CodecPCM,
		SampleRate: 16000,
	}

	_, err := CreateDecode(src, pcm)
	if err != media.ErrCodecNotSupported {
		t.Errorf("expected ErrCodecNotSupported, got %v", err)
	}
}

func TestHasCodec(t *testing.T) {
	// Test existing codecs
	if !HasCodec(CodecPCM) {
		t.Error("expected PCM codec to be available")
	}
	if !HasCodec(CodecPCMU) {
		t.Error("expected PCMU codec to be available")
	}
	if !HasCodec(CodecPCMA) {
		t.Error("expected PCMA codec to be available")
	}
	if !HasCodec(CodecG722) {
		t.Error("expected G722 codec to be available")
	}

	// Test non-existent codec
	if HasCodec("nonexistent") {
		t.Error("expected nonexistent codec to return false")
	}

	// Test case-insensitive
	if !HasCodec("pcm") {
		t.Error("expected case-insensitive lookup")
	}
	if !HasCodec("PCM") {
		t.Error("expected case-insensitive lookup")
	}
}

func TestStripWavHeader(t *testing.T) {
	// Test with WAV header
	wavData := make([]byte, 100)
	copy(wavData, "RIFF")
	copy(wavData[8:], "WAVE")

	result := StripWavHeader(wavData)
	if len(result) != 56 { // 100 - 44
		t.Errorf("expected result length 56, got %d", len(result))
	}

	// Test without WAV header
	normalData := []byte{1, 2, 3, 4, 5}
	result = StripWavHeader(normalData)
	if len(result) != len(normalData) {
		t.Errorf("expected same length, got %d vs %d", len(result), len(normalData))
	}

	// Test with data smaller than header
	smallData := []byte{1, 2, 3}
	result = StripWavHeader(smallData)
	if len(result) != len(smallData) {
		t.Errorf("expected same length for small data, got %d vs %d", len(result), len(smallData))
	}
}

func TestSplitFrames(t *testing.T) {
	data := make([]byte, 3200) // 100ms at 16kHz, 16-bit mono

	// Test without frame duration
	src := &media.CodecConfig{
		SampleRate: 16000,
	}
	packets := splitFrames(data, src)
	if len(packets) != 1 {
		t.Errorf("expected 1 packet without frame duration, got %d", len(packets))
	}

	// Test with frame duration
	src.FrameDuration = "20ms"
	packets = splitFrames(data, src)
	if len(packets) == 0 {
		t.Error("expected packets with frame duration")
	}

	// Test with invalid frame duration (too small)
	src.FrameDuration = "1ms"
	packets = splitFrames(data, src)
	if len(packets) == 0 {
		t.Error("expected packets even with invalid duration")
	}

	// Test with invalid frame duration (too large)
	src.FrameDuration = "500ms"
	packets = splitFrames(data, src)
	if len(packets) == 0 {
		t.Error("expected packets even with invalid duration")
	}
}

func TestCodecConstants(t *testing.T) {
	// Verify codec constants are defined
	if CodecPCM == "" {
		t.Error("expected CodecPCM to be defined")
	}
	if CodecPCMU == "" {
		t.Error("expected CodecPCMU to be defined")
	}
	if CodecPCMA == "" {
		t.Error("expected CodecPCMA to be defined")
	}
	if CodecG722 == "" {
		t.Error("expected CodecG722 to be defined")
	}
	if CodecOPUS == "" {
		t.Error("expected CodecOPUS to be defined")
	}
}

func TestCreateEncode_CaseInsensitive(t *testing.T) {
	src := media.CodecConfig{
		Codec:      "PCM", // Uppercase
		SampleRate: 16000,
	}
	pcm := media.CodecConfig{
		Codec:      CodecPCM,
		SampleRate: 16000,
	}

	_, err := CreateEncode(src, pcm)
	if err != nil {
		t.Errorf("expected no error for case-insensitive codec, got %v", err)
	}
}

func TestCreateDecode_CaseInsensitive(t *testing.T) {
	src := media.CodecConfig{
		Codec:      "pcm", // Lowercase
		SampleRate: 16000,
	}
	pcm := media.CodecConfig{
		Codec:      CodecPCM,
		SampleRate: 16000,
	}

	_, err := CreateDecode(src, pcm)
	if err != nil {
		t.Errorf("expected no error for case-insensitive codec, got %v", err)
	}
}
