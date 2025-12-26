package captcha

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.imageCaptcha == nil {
		t.Fatal("ImageCaptcha is nil")
	}
	if manager.sliderCaptcha == nil {
		t.Fatal("SliderCaptcha is nil")
	}
}

func TestManager_GenerateImage(t *testing.T) {
	manager := NewManager(DefaultConfig())
	result, err := manager.GenerateImage()
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}
	if result == nil {
		t.Fatal("GenerateImage returned nil")
	}
	if result.Type != TypeImage {
		t.Fatalf("Expected type %s, got %s", TypeImage, result.Type)
	}
}

func TestManager_GenerateSlider(t *testing.T) {
	manager := NewManager(DefaultConfig())
	result, err := manager.GenerateSlider()
	if err != nil {
		t.Fatalf("GenerateSlider failed: %v", err)
	}
	if result == nil {
		t.Fatal("GenerateSlider returned nil")
	}
	if result.Type != TypeSlider {
		t.Fatalf("Expected type %s, got %s", TypeSlider, result.Type)
	}
}

func TestManager_Generate(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// 测试图形验证码
	result, err := manager.Generate(TypeImage)
	if err != nil {
		t.Fatalf("Generate(TypeImage) failed: %v", err)
	}
	if result.Type != TypeImage {
		t.Fatalf("Expected type %s, got %s", TypeImage, result.Type)
	}

	// 测试滑块验证码
	result, err = manager.Generate(TypeSlider)
	if err != nil {
		t.Fatalf("Generate(TypeSlider) failed: %v", err)
	}
	if result.Type != TypeSlider {
		t.Fatalf("Expected type %s, got %s", TypeSlider, result.Type)
	}
}

func TestManager_VerifyImage(t *testing.T) {
	manager := NewManager(DefaultConfig())
	result, err := manager.GenerateImage()
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}

	code, ok := result.Data["code"].(string)
	if !ok {
		t.Fatal("Code not found in result data")
	}

	valid, err := manager.VerifyImage(result.ID, code)
	if err != nil {
		t.Fatalf("VerifyImage failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}
}

func TestManager_VerifySlider(t *testing.T) {
	manager := NewManager(DefaultConfig())
	result, err := manager.GenerateSlider()
	if err != nil {
		t.Fatalf("GenerateSlider failed: %v", err)
	}

	x, ok := result.Data["x"].(int)
	if !ok {
		t.Fatal("X not found in result data")
	}
	y, ok := result.Data["y"].(int)
	if !ok {
		t.Fatal("Y not found in result data")
	}

	valid, err := manager.VerifySlider(result.ID, x, y)
	if err != nil {
		t.Fatalf("VerifySlider failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}
}

func TestInitGlobalManager(t *testing.T) {
	config := DefaultConfig()
	InitGlobalManager(config)
	if GlobalManager == nil {
		t.Fatal("GlobalManager should be initialized")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if config.ImageWidth != 200 {
		t.Fatalf("Expected ImageWidth 200, got %d", config.ImageWidth)
	}
	if config.ImageHeight != 60 {
		t.Fatalf("Expected ImageHeight 60, got %d", config.ImageHeight)
	}
	if config.ImageLength != 4 {
		t.Fatalf("Expected ImageLength 4, got %d", config.ImageLength)
	}
	if config.Expiration != 5*time.Minute {
		t.Fatalf("Expected Expiration 5m, got %v", config.Expiration)
	}
}

func TestManager_GenerateClick(t *testing.T) {
	manager := NewManager(DefaultConfig())
	result, err := manager.GenerateClick()
	if err != nil {
		t.Fatalf("GenerateClick failed: %v", err)
	}
	if result == nil {
		t.Fatal("GenerateClick returned nil")
	}
	if result.Type != TypeClick {
		t.Fatalf("Expected type %s, got %s", TypeClick, result.Type)
	}
}

func TestManager_GeneratePuzzle(t *testing.T) {
	manager := NewManager(DefaultConfig())
	result, err := manager.GeneratePuzzle()
	if err != nil {
		t.Fatalf("GeneratePuzzle failed: %v", err)
	}
	if result == nil {
		t.Fatal("GeneratePuzzle returned nil")
	}
	if result.Type != TypePuzzle {
		t.Fatalf("Expected type %s, got %s", TypePuzzle, result.Type)
	}
}

func TestManager_VerifyClick(t *testing.T) {
	manager := NewManager(DefaultConfig())
	result, err := manager.GenerateClick()
	if err != nil {
		t.Fatalf("GenerateClick failed: %v", err)
	}

	positions, ok := result.Data["positions"].([]Point)
	if !ok {
		t.Fatal("Positions not found in result data")
	}

	valid, err := manager.VerifyClick(result.ID, positions)
	if err != nil {
		t.Fatalf("VerifyClick failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}
}

func TestManager_VerifyPuzzle(t *testing.T) {
	manager := NewManager(DefaultConfig())
	result, err := manager.GeneratePuzzle()
	if err != nil {
		t.Fatalf("GeneratePuzzle failed: %v", err)
	}

	x, ok := result.Data["x"].(int)
	if !ok {
		t.Fatal("X not found in result data")
	}
	y, ok := result.Data["y"].(int)
	if !ok {
		t.Fatal("Y not found in result data")
	}

	valid, err := manager.VerifyPuzzle(result.ID, x, y)
	if err != nil {
		t.Fatalf("VerifyPuzzle failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}
}

func TestManager_GenerateWithClickAndPuzzle(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// 测试点击验证码
	result, err := manager.Generate(TypeClick)
	if err != nil {
		t.Fatalf("Generate(TypeClick) failed: %v", err)
	}
	if result.Type != TypeClick {
		t.Fatalf("Expected type %s, got %s", TypeClick, result.Type)
	}

	// 测试拼图验证码
	result, err = manager.Generate(TypePuzzle)
	if err != nil {
		t.Fatalf("Generate(TypePuzzle) failed: %v", err)
	}
	if result.Type != TypePuzzle {
		t.Fatalf("Expected type %s, got %s", TypePuzzle, result.Type)
	}
}

func TestManager_VerifyWithClickAndPuzzle(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// 测试点击验证码验证
	clickResult, err := manager.GenerateClick()
	if err != nil {
		t.Fatalf("GenerateClick failed: %v", err)
	}
	positions, ok := clickResult.Data["positions"].([]Point)
	if !ok {
		t.Fatal("Positions not found")
	}
	valid, err := manager.Verify(TypeClick, clickResult.ID, positions)
	if err != nil {
		t.Fatalf("Verify(TypeClick) failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification for click captcha")
	}

	// 测试拼图验证码验证
	puzzleResult, err := manager.GeneratePuzzle()
	if err != nil {
		t.Fatalf("GeneratePuzzle failed: %v", err)
	}
	x, ok := puzzleResult.Data["x"].(int)
	if !ok {
		t.Fatal("X not found")
	}
	y, ok := puzzleResult.Data["y"].(int)
	if !ok {
		t.Fatal("Y not found")
	}
	valid, err = manager.Verify(TypePuzzle, puzzleResult.ID, map[string]int{"x": x, "y": y})
	if err != nil {
		t.Fatalf("Verify(TypePuzzle) failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification for puzzle captcha")
	}
}
