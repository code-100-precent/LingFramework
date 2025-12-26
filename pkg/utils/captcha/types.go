package captcha

import "time"

// Type 验证码类型
type Type string

const (
	TypeImage  Type = "image"  // 图形验证码
	TypeSlider Type = "slider" // 滑块验证码
	TypeClick  Type = "click"  // 点击验证码
	TypePuzzle Type = "puzzle" // 拼图验证码
)

// Result 验证码生成结果
type Result struct {
	ID      string                 `json:"id"`      // 验证码ID
	Type    Type                   `json:"type"`    // 验证码类型
	Data    map[string]interface{} `json:"data"`    // 验证码数据（根据类型不同而不同）
	Expires time.Time              `json:"expires"` // 过期时间
}

// ImageCaptchaData 图形验证码数据
type ImageCaptchaData struct {
	Image string `json:"image"` // Base64编码的图片
	Code  string `json:"code"`  // 验证码内容（仅用于测试，生产环境不应返回）
}

// SliderCaptchaData 滑块验证码数据
type SliderCaptchaData struct {
	BackgroundImage string `json:"background_image"` // 背景图Base64
	SliderImage     string `json:"slider_image"`     // 滑块图Base64
	X               int    `json:"x"`                // 滑块应该移动到的X坐标
	Y               int    `json:"y"`                // 滑块应该移动到的Y坐标
	Width           int    `json:"width"`            // 滑块宽度
	Height          int    `json:"height"`           // 滑块高度
	Tolerance       int    `json:"tolerance"`        // 容差（像素）
}

// ClickCaptchaData 点击验证码数据
type ClickCaptchaData struct {
	Image     string  `json:"image"`     // 图片Base64
	Positions []Point `json:"positions"` // 需要点击的位置列表
	Count     int     `json:"count"`     // 需要点击的数量
	Tolerance int     `json:"tolerance"` // 容差（像素）
}

// Point 坐标点
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// PuzzleCaptchaData 拼图验证码数据
type PuzzleCaptchaData struct {
	BackgroundImage string `json:"background_image"` // 背景图Base64
	PuzzleImage     string `json:"puzzle_image"`     // 拼图块Base64
	X               int    `json:"x"`                // 拼图应该移动到的X坐标
	Y               int    `json:"y"`                // 拼图应该移动到的Y坐标
	Tolerance       int    `json:"tolerance"`        // 容差（像素）
}
