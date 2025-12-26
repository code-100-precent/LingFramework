package captcha

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"time"
)

// SliderCaptcha 滑块验证码
type SliderCaptcha struct {
	width      int
	height     int
	sliderSize int // 滑块大小
	tolerance  int // 容差（像素）
	expiration time.Duration
	store      Store
}

// NewSliderCaptcha 创建滑块验证码管理器
func NewSliderCaptcha(width, height, sliderSize, tolerance int, expiration time.Duration, store Store) *SliderCaptcha {
	if store == nil {
		store = NewMemoryStore()
	}
	return &SliderCaptcha{
		width:      width,
		height:     height,
		sliderSize: sliderSize,
		tolerance:  tolerance,
		expiration: expiration,
		store:      store,
	}
}

// Generate 生成滑块验证码
func (sc *SliderCaptcha) Generate() (*Result, error) {
	// 生成随机位置（确保滑块不会超出边界）
	maxX := sc.width - sc.sliderSize - 20
	maxY := sc.height - sc.sliderSize - 20
	if maxX < 20 {
		maxX = 20
	}
	if maxY < 20 {
		maxY = 20
	}

	rand.Seed(time.Now().UnixNano())
	x := rand.Intn(maxX-20) + 20
	y := rand.Intn(maxY-20) + 20

	// 生成背景图
	background, err := sc.generateBackground()
	if err != nil {
		return nil, fmt.Errorf("failed to generate background: %w", err)
	}

	// 生成滑块图和挖洞后的背景图
	sliderImg, backgroundWithHole, err := sc.createSliderAndHole(background, x, y)
	if err != nil {
		return nil, fmt.Errorf("failed to create slider: %w", err)
	}

	// 转换为base64
	backgroundBase64, err := sc.imageToBase64(backgroundWithHole)
	if err != nil {
		return nil, fmt.Errorf("failed to encode background: %w", err)
	}

	sliderBase64, err := sc.imageToBase64(sliderImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode slider: %w", err)
	}

	// 生成ID
	id := sc.generateID()

	// 存储正确位置
	expires := time.Now().Add(sc.expiration)
	position := map[string]int{"x": x, "y": y}
	if err := sc.store.Set(id, position, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &Result{
		ID:      id,
		Type:    TypeSlider,
		Data:    map[string]interface{}{"background_image": backgroundBase64, "slider_image": sliderBase64, "x": x, "y": y, "width": sc.sliderSize, "height": sc.sliderSize, "tolerance": sc.tolerance},
		Expires: expires,
	}, nil
}

// Verify 验证滑块验证码
func (sc *SliderCaptcha) Verify(id string, userX, userY int) (bool, error) {
	return sc.store.VerifyWithFunc(id, map[string]int{"x": userX, "y": userY}, func(stored, input interface{}) bool {
		storedPos, ok1 := stored.(map[string]int)
		inputPos, ok2 := input.(map[string]int)
		if !ok1 || !ok2 {
			return false
		}

		storedX := storedPos["x"]
		storedY := storedPos["y"]
		inputX := inputPos["x"]
		inputY := inputPos["y"]

		// 计算距离
		dx := abs(inputX - storedX)
		dy := abs(inputY - storedY)

		// 检查是否在容差范围内
		return dx <= sc.tolerance && dy <= sc.tolerance
	})
}

// generateBackground 生成背景图
func (sc *SliderCaptcha) generateBackground() (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, sc.width, sc.height))

	rand.Seed(time.Now().UnixNano())

	// 生成随机背景（使用渐变或随机色块）
	for y := 0; y < sc.height; y++ {
		for x := 0; x < sc.width; x++ {
			// 创建渐变效果
			r := uint8(100 + (x*50)/sc.width)
			g := uint8(100 + (y*50)/sc.height)
			b := uint8(150 + rand.Intn(50))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// 添加一些干扰图案
	for i := 0; i < 20; i++ {
		x := rand.Intn(sc.width)
		y := rand.Intn(sc.height)
		radius := rand.Intn(10) + 5
		circleColor := color.RGBA{
			uint8(rand.Intn(100) + 50),
			uint8(rand.Intn(100) + 50),
			uint8(rand.Intn(100) + 50),
			255,
		}
		drawCircle(img, x, y, radius, circleColor)
	}

	return img, nil
}

// createSliderAndHole 创建滑块图和挖洞后的背景图
func (sc *SliderCaptcha) createSliderAndHole(background image.Image, x, y int) (sliderImg, backgroundWithHole image.Image, err error) {
	// 创建背景图的副本
	bgBounds := background.Bounds()
	backgroundWithHole = image.NewRGBA(bgBounds)
	draw.Draw(backgroundWithHole.(*image.RGBA), bgBounds, background, bgBounds.Min, draw.Src)

	// 创建滑块图
	sliderImg = image.NewRGBA(image.Rect(0, 0, sc.sliderSize, sc.sliderSize))

	// 从背景图中提取滑块区域
	bgRGBA := backgroundWithHole.(*image.RGBA)
	sliderRGBA := sliderImg.(*image.RGBA)

	for sy := 0; sy < sc.sliderSize; sy++ {
		for sx := 0; sx < sc.sliderSize; sx++ {
			bgX := x + sx
			bgY := y + sy
			if bgX < bgBounds.Dx() && bgY < bgBounds.Dy() {
				// 复制像素到滑块图
				sliderRGBA.Set(sx, sy, bgRGBA.At(bgX, bgY))
			}
		}
	}

	// 在背景图上挖洞（用半透明或特殊颜色填充）
	holeColor := color.RGBA{200, 200, 200, 200}
	drawSliderHole(backgroundWithHole.(*image.RGBA), x, y, sc.sliderSize, holeColor)

	// 给滑块添加边框和阴影效果
	addSliderBorder(sliderRGBA)

	return sliderImg, backgroundWithHole, nil
}

// drawSliderHole 在背景图上绘制滑块洞
func drawSliderHole(img *image.RGBA, x, y, size int, c color.Color) {
	// 绘制圆角矩形洞
	radius := size / 4
	for sy := 0; sy < size; sy++ {
		for sx := 0; sx < size; sx++ {
			// 检查是否在圆角矩形内
			if isInRoundedRect(sx, sy, size, size, radius) {
				bgX := x + sx
				bgY := y + sy
				if bgX < img.Bounds().Dx() && bgY < img.Bounds().Dy() {
					img.Set(bgX, bgY, c)
				}
			}
		}
	}
}

// addSliderBorder 给滑块添加边框
func addSliderBorder(img *image.RGBA) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 添加白色边框
	borderColor := color.RGBA{255, 255, 255, 255}
	for y := 0; y < height; y++ {
		img.Set(0, y, borderColor)
		img.Set(width-1, y, borderColor)
	}
	for x := 0; x < width; x++ {
		img.Set(x, 0, borderColor)
		img.Set(x, height-1, borderColor)
	}

	// 添加阴影效果（底部和右侧）
	shadowColor := color.RGBA{0, 0, 0, 100}
	for y := 0; y < height; y++ {
		if y < height-2 {
			img.Set(width-2, y, shadowColor)
		}
	}
	for x := 0; x < width; x++ {
		if x < width-2 {
			img.Set(x, height-2, shadowColor)
		}
	}
}

// isInRoundedRect 检查点是否在圆角矩形内
func isInRoundedRect(x, y, width, height, radius int) bool {
	// 检查是否在矩形内
	if x < 0 || x >= width || y < 0 || y >= height {
		return false
	}

	// 检查四个角的圆角
	// 左上角
	if x < radius && y < radius {
		dx := x - radius
		dy := y - radius
		return dx*dx+dy*dy <= radius*radius
	}
	// 右上角
	if x >= width-radius && y < radius {
		dx := x - (width - radius)
		dy := y - radius
		return dx*dx+dy*dy <= radius*radius
	}
	// 左下角
	if x < radius && y >= height-radius {
		dx := x - radius
		dy := y - (height - radius)
		return dx*dx+dy*dy <= radius*radius
	}
	// 右下角
	if x >= width-radius && y >= height-radius {
		dx := x - (width - radius)
		dy := y - (height - radius)
		return dx*dx+dy*dy <= radius*radius
	}

	return true
}

// drawCircle 绘制圆形
func drawCircle(img *image.RGBA, cx, cy, radius int, c color.Color) {
	bounds := img.Bounds()
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y {
				dx := x - cx
				dy := y - cy
				if dx*dx+dy*dy <= radius*radius {
					img.Set(x, y, c)
				}
			}
		}
	}
}

// imageToBase64 将图片转换为base64
func (sc *SliderCaptcha) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// generateID 生成验证码ID
func (sc *SliderCaptcha) generateID() string {
	rand.Seed(time.Now().UnixNano())
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var id bytes.Buffer
	for i := 0; i < 32; i++ {
		id.WriteByte(chars[rand.Intn(len(chars))])
	}
	return id.String()
}
