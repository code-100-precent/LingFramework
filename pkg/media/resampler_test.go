package media

import (
	"testing"
)

func TestDefaultResampler(t *testing.T) {
	converter := DefaultResampler(16000, 48000)
	if converter == nil {
		t.Fatal("expected non-nil converter")
	}
	defer converter.Close()
}

func TestSetDefaultResampler(t *testing.T) {
	originalFactory := defaultConverterFactory

	customFactory := func(inputRate, outputRate int) SampleRateConverter {
		return NewInterpolatingConverter(inputRate, outputRate)
	}

	SetDefaultResampler(customFactory)

	converter := DefaultResampler(16000, 48000)
	if converter == nil {
		t.Fatal("expected non-nil converter")
	}
	converter.Close()

	// Restore original
	SetDefaultResampler(originalFactory)
}

func TestResamplePCM_SameRate(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6}
	result, err := ResamplePCM(data, 16000, 16000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(data) {
		t.Errorf("expected same length, got %d vs %d", len(result), len(data))
	}
}

func TestResamplePCM_DifferentRate(t *testing.T) {
	// Create sample PCM data (16-bit samples)
	data := make([]byte, 320) // 160 samples at 16kHz = 10ms
	for i := 0; i < len(data); i += 2 {
		data[i] = byte(i % 256)
		data[i+1] = byte((i / 256) % 256)
	}

	result, err := ResamplePCM(data, 16000, 48000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	// Should be approximately 3x the size (48000/16000 = 3)
	if len(result) < len(data) {
		t.Errorf("expected result to be larger, got %d vs %d", len(result), len(data))
	}
}

func TestNewInterpolatingConverter(t *testing.T) {
	converter := NewInterpolatingConverter(16000, 48000)
	if converter == nil {
		t.Fatal("expected non-nil converter")
	}

	ic, ok := converter.(*InterpolatingConverter)
	if !ok {
		t.Fatal("expected InterpolatingConverter type")
	}
	if ic.sourceRate != 16000 {
		t.Errorf("expected source rate 16000, got %d", ic.sourceRate)
	}
	if ic.targetRate != 48000 {
		t.Errorf("expected target rate 48000, got %d", ic.targetRate)
	}
	if ic.useCubic {
		t.Error("expected linear interpolation by default")
	}

	converter.Close()
}

func TestNewCubicInterpolatingConverter(t *testing.T) {
	converter := NewCubicInterpolatingConverter(16000, 48000)
	if converter == nil {
		t.Fatal("expected non-nil converter")
	}

	ic, ok := converter.(*InterpolatingConverter)
	if !ok {
		t.Fatal("expected InterpolatingConverter type")
	}
	if !ic.useCubic {
		t.Error("expected cubic interpolation")
	}

	converter.Close()
}

func TestInterpolatingConverter_ConvertSamples_SameRate(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 16000,
		targetRate: 16000,
	}

	samples := []byte{1, 2, 3, 4}
	result := ic.ConvertSamples(samples)
	if len(result) != len(samples) {
		t.Errorf("expected same length, got %d vs %d", len(result), len(samples))
	}
}

func TestInterpolatingConverter_ConvertSamples_OddLength(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 16000,
		targetRate: 48000,
	}

	samples := []byte{1, 2, 3} // Odd length
	result := ic.ConvertSamples(samples)
	if result != nil {
		t.Error("expected nil result for odd length samples")
	}
}

func TestInterpolatingConverter_ConvertSamples_Upsample(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 16000,
		targetRate: 48000,
		useCubic:   false,
	}

	// Create sample data (even number of bytes)
	samples := make([]byte, 320) // 160 samples
	for i := 0; i < len(samples); i++ {
		samples[i] = byte(i % 256)
	}

	result := ic.ConvertSamples(samples)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	// Should be approximately 3x larger
	if len(result) < len(samples) {
		t.Errorf("expected larger result, got %d vs %d", len(result), len(samples))
	}
}

func TestInterpolatingConverter_ConvertSamples_Downsample(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 48000,
		targetRate: 16000,
		useCubic:   false,
	}

	// Create sample data
	samples := make([]byte, 960) // 480 samples at 48kHz
	for i := 0; i < len(samples); i++ {
		samples[i] = byte(i % 256)
	}

	result := ic.ConvertSamples(samples)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	// Should be approximately 1/3 the size
	if len(result) > len(samples) {
		t.Errorf("expected smaller result, got %d vs %d", len(result), len(samples))
	}
}

func TestInterpolatingConverter_ConvertSamples_Cubic(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 16000,
		targetRate: 48000,
		useCubic:   true,
	}

	// Create sample data with enough samples for cubic interpolation
	samples := make([]byte, 320) // 160 samples
	for i := 0; i < len(samples); i++ {
		samples[i] = byte(i % 256)
	}

	result := ic.ConvertSamples(samples)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestInterpolatingConverter_Write(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 16000,
		targetRate: 48000,
	}

	data := []byte{1, 2, 3, 4}
	n, err := ic.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected written bytes %d, got %d", len(data), n)
	}
	if len(ic.buffer) == 0 {
		t.Error("expected buffer to contain converted samples")
	}
}

func TestInterpolatingConverter_Samples(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 16000,
		targetRate: 48000,
		buffer:     []byte{1, 2, 3, 4},
	}

	result := ic.Samples()
	if len(result) != 4 {
		t.Errorf("expected 4 bytes, got %d", len(result))
	}

	// Buffer should be cleared
	result2 := ic.Samples()
	if len(result2) != 0 {
		t.Errorf("expected empty buffer after Samples(), got %d bytes", len(result2))
	}
}

func TestInterpolatingConverter_Close(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 16000,
		targetRate: 48000,
	}

	err := ic.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInterpolatingConverter_EdgeCases(t *testing.T) {
	ic := &InterpolatingConverter{
		sourceRate: 16000,
		targetRate: 48000,
		useCubic:   true,
	}

	// Test with very small input
	samples := []byte{1, 2}
	result := ic.ConvertSamples(samples)
	if result == nil {
		t.Error("expected non-nil result even for small input")
	}

	// Test with empty input
	result = ic.ConvertSamples([]byte{})
	if result == nil {
		t.Error("expected non-nil result for empty input")
	}
}
