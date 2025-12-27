package media

import (
	"testing"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter(StrategyBroadcast)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
	if router.defaultStrategy != StrategyBroadcast {
		t.Errorf("expected default strategy %d, got %d", StrategyBroadcast, router.defaultStrategy)
	}
}

func TestRouter_AddRule(t *testing.T) {
	router := NewRouter(StrategyBroadcast)

	rule := RouteRule{
		Condition: func(packet MediaPacket) bool {
			return true
		},
		Targets:  []string{"target1"},
		Strategy: StrategyRoundRobin,
	}

	router.AddRule(rule)

	if len(router.rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(router.rules))
	}
}

func TestRouter_Route_Broadcast(t *testing.T) {
	router := NewRouter(StrategyBroadcast)

	conn1 := NewTransportConnector("conn1", nil, DirectionOutput)
	conn2 := NewTransportConnector("conn2", nil, DirectionOutput)
	conn3 := NewTransportConnector("conn3", nil, DirectionOutput)

	available := []*TransportConnector{conn1, conn2, conn3}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	result := router.Route(packet, available)

	if len(result) != 3 {
		t.Errorf("expected 3 connectors, got %d", len(result))
	}
}

func TestRouter_Route_RoundRobin(t *testing.T) {
	router := NewRouter(StrategyRoundRobin)

	conn1 := NewTransportConnector("conn1", nil, DirectionOutput)
	conn2 := NewTransportConnector("conn2", nil, DirectionOutput)
	conn3 := NewTransportConnector("conn3", nil, DirectionOutput)

	available := []*TransportConnector{conn1, conn2, conn3}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}

	// First call
	result1 := router.Route(packet, available)
	if len(result1) != 1 {
		t.Errorf("expected 1 connector, got %d", len(result1))
	}

	// Second call - should get next connector
	result2 := router.Route(packet, available)
	if len(result2) != 1 {
		t.Errorf("expected 1 connector, got %d", len(result2))
	}

	// Results should be different (round robin)
	if result1[0].ID == result2[0].ID {
		t.Error("expected different connectors in round robin")
	}
}

func TestRouter_Route_FirstAvailable(t *testing.T) {
	router := NewRouter(StrategyFirstAvailable)

	conn1 := NewTransportConnector("conn1", nil, DirectionOutput)
	conn2 := NewTransportConnector("conn2", nil, DirectionOutput)

	available := []*TransportConnector{conn1, conn2}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	result := router.Route(packet, available)

	if len(result) != 1 {
		t.Errorf("expected 1 connector, got %d", len(result))
	}
	if result[0].ID != "conn1" {
		t.Errorf("expected first connector, got %s", result[0].ID)
	}
}

func TestRouter_Route_WithRule(t *testing.T) {
	router := NewRouter(StrategyBroadcast)

	// Add rule for audio packets
	router.AddRule(RouteRule{
		Condition: func(packet MediaPacket) bool {
			_, ok := packet.(*AudioPacket)
			return ok
		},
		Strategy: StrategyFirstAvailable,
	})

	conn1 := NewTransportConnector("conn1", nil, DirectionOutput)
	conn2 := NewTransportConnector("conn2", nil, DirectionOutput)
	available := []*TransportConnector{conn1, conn2}

	// Audio packet should match rule
	audioPacket := &AudioPacket{Payload: []byte{1, 2, 3}}
	result := router.Route(audioPacket, available)
	if len(result) != 1 {
		t.Errorf("expected 1 connector with rule, got %d", len(result))
	}

	// Text packet should use default strategy
	textPacket := &TextPacket{Text: "hello"}
	result = router.Route(textPacket, available)
	if len(result) != 2 {
		t.Errorf("expected 2 connectors with default strategy, got %d", len(result))
	}
}

func TestRouter_Route_EmptyAvailable(t *testing.T) {
	router := NewRouter(StrategyBroadcast)

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}
	result := router.Route(packet, nil)

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}

	result = router.Route(packet, []*TransportConnector{})
	if result != nil {
		t.Errorf("expected nil result for empty slice, got %v", result)
	}
}

func TestRouter_RoundRobin_WrapsAround(t *testing.T) {
	router := NewRouter(StrategyRoundRobin)

	conn1 := NewTransportConnector("conn1", nil, DirectionOutput)
	conn2 := NewTransportConnector("conn2", nil, DirectionOutput)
	available := []*TransportConnector{conn1, conn2}

	packet := &AudioPacket{Payload: []byte{1, 2, 3}}

	// Call multiple times to test wrap-around
	results := make([]string, 4)
	for i := 0; i < 4; i++ {
		result := router.Route(packet, available)
		results[i] = result[0].ID
	}

	// Should alternate: conn1, conn2, conn1, conn2
	if results[0] != results[2] {
		t.Error("expected round robin to wrap around")
	}
	if results[1] != results[3] {
		t.Error("expected round robin to wrap around")
	}
}

func TestNewTransportConnector(t *testing.T) {
	conn := NewTransportConnector("conn1", nil, DirectionOutput)

	if conn.ID != "conn1" {
		t.Errorf("expected ID 'conn1', got '%s'", conn.ID)
	}
	if conn.Direction != DirectionOutput {
		t.Errorf("expected direction '%s', got '%s'", DirectionOutput, conn.Direction)
	}
	if !conn.Active {
		t.Error("expected connector to be active by default")
	}
}

func TestTransportConnector_String(t *testing.T) {
	conn := NewTransportConnector("conn1", nil, DirectionInput)
	str := conn.String()

	if !contains(str, "conn1") {
		t.Errorf("expected string to contain 'conn1', got '%s'", str)
	}
	if !contains(str, DirectionInput) {
		t.Errorf("expected string to contain direction, got '%s'", str)
	}
}

func TestTransportConnector_SetActive(t *testing.T) {
	conn := NewTransportConnector("conn1", nil, DirectionOutput)

	if !conn.IsActive() {
		t.Error("expected connector to be active initially")
	}

	conn.SetActive(false)
	if conn.IsActive() {
		t.Error("expected connector to be inactive after SetActive(false)")
	}

	conn.SetActive(true)
	if !conn.IsActive() {
		t.Error("expected connector to be active after SetActive(true)")
	}
}

func TestTransportConnector_IsActive(t *testing.T) {
	conn := NewTransportConnector("conn1", nil, DirectionOutput)

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = conn.IsActive()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
