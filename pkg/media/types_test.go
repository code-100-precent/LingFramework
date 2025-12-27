package media

import (
	"testing"
	"time"
)

func TestTextPacket_Body(t *testing.T) {
	p := &TextPacket{
		Text: "hello world",
	}
	body := p.Body()
	if string(body) != "hello world" {
		t.Errorf("expected body 'hello world', got '%s'", string(body))
	}
}

func TestTextPacket_String(t *testing.T) {
	tests := []struct {
		name     string
		packet   *TextPacket
		contains []string
	}{
		{
			name: "user text",
			packet: &TextPacket{
				Text: "hello",
			},
			contains: []string{"hello", "user"},
		},
		{
			name: "transcribed text",
			packet: &TextPacket{
				Text:          "hello",
				IsTranscribed: true,
			},
			contains: []string{"hello", "Transcribed"},
		},
		{
			name: "llm generated text",
			packet: &TextPacket{
				Text:           "hello",
				IsLLMGenerated: true,
			},
			contains: []string{"hello", "LLMGenerated"},
		},
		{
			name: "partial text",
			packet: &TextPacket{
				Text:      "hello",
				IsPartial: true,
			},
			contains: []string{"IsPartial: true"},
		},
		{
			name: "end text",
			packet: &TextPacket{
				Text:  "hello",
				IsEnd: true,
			},
			contains: []string{"IsEnd: true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.packet.String()
			for _, contain := range tt.contains {
				if !contains(str, contain) {
					t.Errorf("expected string to contain '%s', got '%s'", contain, str)
				}
			}
		})
	}
}

func TestAudioPacket_Body(t *testing.T) {
	payload := []byte{1, 2, 3, 4}
	p := &AudioPacket{
		Payload: payload,
	}
	body := p.Body()
	if len(body) != len(payload) {
		t.Errorf("expected body length %d, got %d", len(payload), len(body))
	}
	for i := range payload {
		if body[i] != payload[i] {
			t.Errorf("expected body[%d] = %d, got %d", i, payload[i], body[i])
		}
	}
}

func TestAudioPacket_String(t *testing.T) {
	tests := []struct {
		name     string
		packet   *AudioPacket
		contains []string
	}{
		{
			name: "basic packet",
			packet: &AudioPacket{
				Payload: []byte{1, 2, 3},
			},
			contains: []string{"3 bytes"},
		},
		{
			name: "first packet",
			packet: &AudioPacket{
				Payload:       []byte{1, 2},
				IsFirstPacket: true,
			},
			contains: []string{"IsFirstPacket: true"},
		},
		{
			name: "synthesized packet",
			packet: &AudioPacket{
				Payload:       []byte{1, 2},
				IsSynthesized: true,
			},
			contains: []string{"IsSynthesized: true"},
		},
		{
			name: "silence packet",
			packet: &AudioPacket{
				Payload:   []byte{1, 2},
				IsSilence: true,
			},
			contains: []string{"IsSilence: true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.packet.String()
			for _, contain := range tt.contains {
				if !contains(str, contain) {
					t.Errorf("expected string to contain '%s', got '%s'", contain, str)
				}
			}
		})
	}
}

func TestClosePacket_Body(t *testing.T) {
	p := &ClosePacket{
		Reason: "test",
	}
	body := p.Body()
	if body != nil {
		t.Errorf("expected nil body, got %v", body)
	}
}

func TestClosePacket_String(t *testing.T) {
	p := &ClosePacket{
		Reason: "test reason",
	}
	str := p.String()
	if !contains(str, "test reason") {
		t.Errorf("expected string to contain 'test reason', got '%s'", str)
	}
}

func TestStateChange_SafeGetStr(t *testing.T) {
	tests := []struct {
		name     string
		state    *StateChange
		idx      int
		expected string
	}{
		{
			name: "valid index",
			state: &StateChange{
				Params: []any{"hello", "world"},
			},
			idx:      0,
			expected: "hello",
		},
		{
			name: "out of range",
			state: &StateChange{
				Params: []any{"hello"},
			},
			idx:      5,
			expected: "",
		},
		{
			name: "negative index",
			state: &StateChange{
				Params: []any{"hello"},
			},
			idx:      -1,
			expected: "",
		},
		{
			name: "non-string type",
			state: &StateChange{
				Params: []any{123},
			},
			idx:      0,
			expected: "",
		},
		{
			name: "empty params",
			state: &StateChange{
				Params: []any{},
			},
			idx:      0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.SafeGetStr(tt.idx)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestMediaData_String(t *testing.T) {
	tests := []struct {
		name     string
		data     *MediaData
		contains []string
	}{
		{
			name: "state type",
			data: &MediaData{
				Type:   MediaDataTypeState,
				State:  StateChange{State: "begin"},
				Sender: "test",
			},
			contains: []string{"begin"},
		},
		{
			name: "packet type",
			data: &MediaData{
				Type:   MediaDataTypePacket,
				Packet: &AudioPacket{Payload: []byte{1, 2}},
				Sender: "test",
			},
			contains: []string{"Packet:"},
		},
		{
			name: "other type",
			data: &MediaData{
				Type:   "other",
				Sender: "test",
			},
			contains: []string{"Type:other"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.data.String()
			for _, contain := range tt.contains {
				if !contains(str, contain) {
					t.Errorf("expected string to contain '%s', got '%s'", contain, str)
				}
			}
		})
	}
}

func TestCompletedData_MarshalJSON(t *testing.T) {
	d := &CompletedData{
		SenderName: "test",
		Duration:   time.Second,
		Result:     "result",
		DialogID:   "dialog1",
	}
	data, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}

func TestCompletedData_String(t *testing.T) {
	d := CompletedData{
		SenderName: "test",
		Duration:   time.Second,
		Result:     "result",
		DialogID:   "dialog1",
	}
	str := d.String()
	if !contains(str, "test") || !contains(str, "result") {
		t.Errorf("expected string to contain sender and result, got '%s'", str)
	}
}

func TestTranscribingData_MarshalJSON(t *testing.T) {
	d := &TranscribingData{
		SenderName: "test",
		Duration:   time.Second,
		Result:     "result",
		DialogID:   "dialog1",
	}
	data, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON data")
	}
}

func TestTranscribingData_String(t *testing.T) {
	d := TranscribingData{
		SenderName: "test",
		Duration:   time.Second,
		Result:     "result",
		DialogID:   "dialog1",
	}
	str := d.String()
	if !contains(str, "test") || !contains(str, "result") {
		t.Errorf("expected string to contain sender and result, got '%s'", str)
	}
}

func TestCodecConfig_String(t *testing.T) {
	config := CodecConfig{
		Codec:         "pcm",
		SampleRate:    16000,
		Channels:      1,
		BitDepth:      16,
		FrameDuration: "20ms",
	}
	str := config.String()
	if !contains(str, "pcm") || !contains(str, "16000") {
		t.Errorf("expected string to contain codec and sample rate, got '%s'", str)
	}
}

func TestDefaultCodecConfig(t *testing.T) {
	config := DefaultCodecConfig()
	if config.Codec != "pcm" {
		t.Errorf("expected codec 'pcm', got '%s'", config.Codec)
	}
	if config.SampleRate != 16000 {
		t.Errorf("expected sample rate 16000, got %d", config.SampleRate)
	}
	if config.Channels != 1 {
		t.Errorf("expected channels 1, got %d", config.Channels)
	}
	if config.BitDepth != 16 {
		t.Errorf("expected bit depth 16, got %d", config.BitDepth)
	}
}
