package captcha

import (
	"strings"
	"testing"
	"time"
)

func TestNewPuzzleCaptcha(t *testing.T) {
	captcha := NewPuzzleCaptcha(300, 150, 50, 5, 5*time.Minute, nil)
	if captcha == nil {
		t.Fatal("NewPuzzleCaptcha returned nil")
	}
	if captcha.width != 300 {
		t.Fatalf("Expected width 300, got %d", captcha.width)
	}
	if captcha.height != 150 {
		t.Fatalf("Expected height 150, got %d", captcha.height)
	}
	if captcha.puzzleSize != 50 {
		t.Fatalf("Expected puzzleSize 50, got %d", captcha.puzzleSize)
	}
}

func TestPuzzleCaptcha_Generate(t *testing.T) {
	captcha := NewPuzzleCaptcha(300, 150, 50, 5, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result == nil {
		t.Fatal("Generate returned nil result")
	}
	if result.ID == "" {
		t.Fatal("Result ID is empty")
	}
	if result.Type != TypePuzzle {
		t.Fatalf("Expected type %s, got %s", TypePuzzle, result.Type)
	}
	if result.Data == nil {
		t.Fatal("Result data is nil")
	}
	bgImage, ok := result.Data["background_image"].(string)
	if !ok || bgImage == "" {
		t.Fatal("Result background_image is empty")
	}
	if !strings.HasPrefix(bgImage, "data:image/png;base64,") {
		t.Fatal("Result background_image should be base64 encoded PNG")
	}
	puzzleImage, ok := result.Data["puzzle_image"].(string)
	if !ok || puzzleImage == "" {
		t.Fatal("Result puzzle_image is empty")
	}
	if !strings.HasPrefix(puzzleImage, "data:image/png;base64,") {
		t.Fatal("Result puzzle_image should be base64 encoded PNG")
	}
	x, ok := result.Data["x"].(int)
	if !ok {
		t.Fatal("Result x is not int")
	}
	if x < 20 || x > 280 {
		t.Fatalf("Expected x in range [20, 280], got %d", x)
	}
	if result.Expires.Before(time.Now()) {
		t.Fatal("Result expires time should be in the future")
	}
}

func TestPuzzleCaptcha_Verify(t *testing.T) {
	captcha := NewPuzzleCaptcha(300, 150, 50, 5, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	x, ok := result.Data["x"].(int)
	if !ok {
		t.Fatal("X not found in result data")
	}
	y, ok := result.Data["y"].(int)
	if !ok {
		t.Fatal("Y not found in result data")
	}

	// 正确位置
	valid, err := captcha.Verify(result.ID, x, y)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification")
	}

	// 再次生成验证码用于测试错误位置
	result2, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 错误位置（超出容差）
	valid, err = captcha.Verify(result2.ID, 0, 0)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Expected invalid verification")
	}
}

func TestPuzzleCaptcha_VerifyWithTolerance(t *testing.T) {
	captcha := NewPuzzleCaptcha(300, 150, 50, 5, 5*time.Minute, nil)
	result, err := captcha.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	x, ok := result.Data["x"].(int)
	if !ok {
		t.Fatal("X not found in result data")
	}
	y, ok := result.Data["y"].(int)
	if !ok {
		t.Fatal("Y not found in result data")
	}

	// 在容差范围内
	valid, err := captcha.Verify(result.ID, x+3, y+3)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Expected valid verification within tolerance")
	}
}
