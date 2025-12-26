package captcha

import (
	"fmt"
	"time"
)

// Manager 统一验证码管理器
type Manager struct {
	imageCaptcha  *ImageCaptcha
	sliderCaptcha *SliderCaptcha
	clickCaptcha  *ClickCaptcha
	puzzleCaptcha *PuzzleCaptcha
	store         Store
}

// Config 验证码配置
type Config struct {
	// 图形验证码配置
	ImageWidth  int
	ImageHeight int
	ImageLength int

	// 滑块验证码配置
	SliderWidth     int
	SliderHeight    int
	SliderSize      int
	SliderTolerance int

	// 点击验证码配置
	ClickWidth     int
	ClickHeight    int
	ClickCount     int
	ClickTolerance int

	// 拼图验证码配置
	PuzzleWidth     int
	PuzzleHeight    int
	PuzzleSize      int
	PuzzleTolerance int

	// 通用配置
	Expiration time.Duration
	Store      Store
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		ImageWidth:      200,
		ImageHeight:     60,
		ImageLength:     4,
		SliderWidth:     300,
		SliderHeight:    150,
		SliderSize:      50,
		SliderTolerance: 5,
		ClickWidth:      300,
		ClickHeight:     200,
		ClickCount:      3,
		ClickTolerance:  10,
		PuzzleWidth:     300,
		PuzzleHeight:    150,
		PuzzleSize:      50,
		PuzzleTolerance: 5,
		Expiration:      5 * time.Minute,
		Store:           nil, // 使用默认内存存储
	}
}

// NewManager 创建统一验证码管理器
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	store := config.Store
	if store == nil {
		store = NewMemoryStore()
	}

	return &Manager{
		imageCaptcha:  NewImageCaptcha(config.ImageWidth, config.ImageHeight, config.ImageLength, config.Expiration, store),
		sliderCaptcha: NewSliderCaptcha(config.SliderWidth, config.SliderHeight, config.SliderSize, config.SliderTolerance, config.Expiration, store),
		clickCaptcha:  NewClickCaptcha(config.ClickWidth, config.ClickHeight, config.ClickCount, config.ClickTolerance, config.Expiration, store),
		puzzleCaptcha: NewPuzzleCaptcha(config.PuzzleWidth, config.PuzzleHeight, config.PuzzleSize, config.PuzzleTolerance, config.Expiration, store),
		store:         store,
	}
}

// GenerateImage 生成图形验证码
func (m *Manager) GenerateImage() (*Result, error) {
	return m.imageCaptcha.Generate()
}

// VerifyImage 验证图形验证码
func (m *Manager) VerifyImage(id, code string) (bool, error) {
	return m.imageCaptcha.Verify(id, code)
}

// GenerateSlider 生成滑块验证码
func (m *Manager) GenerateSlider() (*Result, error) {
	return m.sliderCaptcha.Generate()
}

// VerifySlider 验证滑块验证码
func (m *Manager) VerifySlider(id string, x, y int) (bool, error) {
	return m.sliderCaptcha.Verify(id, x, y)
}

// GenerateClick 生成点击验证码
func (m *Manager) GenerateClick() (*Result, error) {
	return m.clickCaptcha.Generate()
}

// VerifyClick 验证点击验证码
func (m *Manager) VerifyClick(id string, positions []Point) (bool, error) {
	return m.clickCaptcha.Verify(id, positions)
}

// GeneratePuzzle 生成拼图验证码
func (m *Manager) GeneratePuzzle() (*Result, error) {
	return m.puzzleCaptcha.Generate()
}

// VerifyPuzzle 验证拼图验证码
func (m *Manager) VerifyPuzzle(id string, x, y int) (bool, error) {
	return m.puzzleCaptcha.Verify(id, x, y)
}

// Generate 根据类型生成验证码
func (m *Manager) Generate(captchaType Type) (*Result, error) {
	switch captchaType {
	case TypeImage:
		return m.GenerateImage()
	case TypeSlider:
		return m.GenerateSlider()
	case TypeClick:
		return m.GenerateClick()
	case TypePuzzle:
		return m.GeneratePuzzle()
	default:
		return nil, fmt.Errorf("unsupported captcha type: %s", captchaType)
	}
}

// Verify 根据类型验证验证码
func (m *Manager) Verify(captchaType Type, id string, data interface{}) (bool, error) {
	switch captchaType {
	case TypeImage:
		if code, ok := data.(string); ok {
			return m.VerifyImage(id, code)
		}
		return false, fmt.Errorf("invalid data type for image captcha, expected string")
	case TypeSlider:
		if pos, ok := data.(map[string]int); ok {
			x, xOk := pos["x"]
			y, yOk := pos["y"]
			if xOk && yOk {
				return m.VerifySlider(id, x, y)
			}
		}
		return false, fmt.Errorf("invalid data type for slider captcha, expected map[string]int with x and y")
	case TypeClick:
		if positions, ok := data.([]Point); ok {
			return m.VerifyClick(id, positions)
		}
		return false, fmt.Errorf("invalid data type for click captcha, expected []Point")
	case TypePuzzle:
		if pos, ok := data.(map[string]int); ok {
			x, xOk := pos["x"]
			y, yOk := pos["y"]
			if xOk && yOk {
				return m.VerifyPuzzle(id, x, y)
			}
		}
		return false, fmt.Errorf("invalid data type for puzzle captcha, expected map[string]int with x and y")
	default:
		return false, fmt.Errorf("unsupported captcha type: %s", captchaType)
	}
}

// GlobalManager 全局验证码管理器
var GlobalManager *Manager

// InitGlobalManager 初始化全局验证码管理器
func InitGlobalManager(config *Config) {
	GlobalManager = NewManager(config)
}
