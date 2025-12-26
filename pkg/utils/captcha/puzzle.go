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

// PuzzleCaptcha 拼图验证码
type PuzzleCaptcha struct {
	width      int
	height     int
	puzzleSize int // 拼图块大小
	tolerance  int // 容差（像素）
	expiration time.Duration
	store      Store
}

// NewPuzzleCaptcha 创建拼图验证码管理器
func NewPuzzleCaptcha(width, height, puzzleSize, tolerance int, expiration time.Duration, store Store) *PuzzleCaptcha {
	if store == nil {
		store = NewMemoryStore()
	}
	return &PuzzleCaptcha{
		width:      width,
		height:     height,
		puzzleSize: puzzleSize,
		tolerance:  tolerance,
		expiration: expiration,
		store:      store,
	}
}

// Generate 生成拼图验证码
func (pc *PuzzleCaptcha) Generate() (*Result, error) {
	// 生成随机位置（确保拼图块不会超出边界）
	maxX := pc.width - pc.puzzleSize - 20
	maxY := pc.height - pc.puzzleSize - 20
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
	background, err := pc.generateBackground()
	if err != nil {
		return nil, fmt.Errorf("failed to generate background: %w", err)
	}

	// 生成拼图块和挖洞后的背景图
	puzzleImg, backgroundWithHole, err := pc.createPuzzleAndHole(background, x, y)
	if err != nil {
		return nil, fmt.Errorf("failed to create puzzle: %w", err)
	}

	// 转换为base64
	backgroundBase64, err := pc.imageToBase64(backgroundWithHole)
	if err != nil {
		return nil, fmt.Errorf("failed to encode background: %w", err)
	}

	puzzleBase64, err := pc.imageToBase64(puzzleImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode puzzle: %w", err)
	}

	// 生成ID
	id := pc.generateID()

	// 存储正确位置
	expires := time.Now().Add(pc.expiration)
	position := map[string]int{"x": x, "y": y}
	if err := pc.store.Set(id, position, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &Result{
		ID:      id,
		Type:    TypePuzzle,
		Data:    map[string]interface{}{"background_image": backgroundBase64, "puzzle_image": puzzleBase64, "x": x, "y": y, "tolerance": pc.tolerance},
		Expires: expires,
	}, nil
}

// Verify 验证拼图验证码
func (pc *PuzzleCaptcha) Verify(id string, userX, userY int) (bool, error) {
	return pc.store.VerifyWithFunc(id, map[string]int{"x": userX, "y": userY}, func(stored, input interface{}) bool {
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
		return dx <= pc.tolerance && dy <= pc.tolerance
	})
}

// generateBackground 生成背景图
func (pc *PuzzleCaptcha) generateBackground() (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, pc.width, pc.height))

	rand.Seed(time.Now().UnixNano())

	// 生成随机背景（使用渐变或随机色块）
	for y := 0; y < pc.height; y++ {
		for x := 0; x < pc.width; x++ {
			// 创建渐变效果
			r := uint8(100 + (x*50)/pc.width)
			g := uint8(100 + (y*50)/pc.height)
			b := uint8(150 + rand.Intn(50))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// 添加一些干扰图案
	for i := 0; i < 15; i++ {
		x := rand.Intn(pc.width)
		y := rand.Intn(pc.height)
		radius := rand.Intn(8) + 3
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

// createPuzzleAndHole 创建拼图块和挖洞后的背景图
func (pc *PuzzleCaptcha) createPuzzleAndHole(background image.Image, x, y int) (puzzleImg, backgroundWithHole image.Image, err error) {
	// 创建背景图的副本
	bgBounds := background.Bounds()
	backgroundWithHole = image.NewRGBA(bgBounds)
	draw.Draw(backgroundWithHole.(*image.RGBA), bgBounds, background, bgBounds.Min, draw.Src)

	// 创建拼图块（不规则形状，类似拼图块）
	puzzleImg = image.NewRGBA(image.Rect(0, 0, pc.puzzleSize, pc.puzzleSize))

	// 从背景图中提取拼图块区域
	bgRGBA := backgroundWithHole.(*image.RGBA)
	puzzleRGBA := puzzleImg.(*image.RGBA)

	// 生成拼图形状（带凸起和凹陷）
	shape := pc.generatePuzzleShape()

	for sy := 0; sy < pc.puzzleSize; sy++ {
		for sx := 0; sx < pc.puzzleSize; sx++ {
			bgX := x + sx
			bgY := y + sy
			if bgX < bgBounds.Dx() && bgY < bgBounds.Dy() {
				// 检查是否在拼图形状内
				if shape[sy][sx] {
					// 复制像素到拼图块
					puzzleRGBA.Set(sx, sy, bgRGBA.At(bgX, bgY))
				}
			}
		}
	}

	// 在背景图上挖洞
	holeColor := color.RGBA{200, 200, 200, 200}
	drawPuzzleHole(backgroundWithHole.(*image.RGBA), x, y, pc.puzzleSize, shape, holeColor)

	// 给拼图块添加边框和阴影效果
	addPuzzleBorder(puzzleRGBA, shape)

	return puzzleImg, backgroundWithHole, nil
}

// generatePuzzleShape 生成拼图形状（带凸起和凹陷）
func (pc *PuzzleCaptcha) generatePuzzleShape() [][]bool {
	shape := make([][]bool, pc.puzzleSize)
	for i := range shape {
		shape[i] = make([]bool, pc.puzzleSize)
	}

	// 基本矩形区域
	margin := pc.puzzleSize / 4
	for y := margin; y < pc.puzzleSize-margin; y++ {
		for x := margin; x < pc.puzzleSize-margin; x++ {
			shape[y][x] = true
		}
	}

	// 添加左侧凸起
	protrusionY := pc.puzzleSize / 2
	protrusionHeight := pc.puzzleSize / 3
	protrusionWidth := pc.puzzleSize / 4
	for y := protrusionY - protrusionHeight/2; y < protrusionY+protrusionHeight/2; y++ {
		if y >= 0 && y < pc.puzzleSize {
			for x := 0; x < protrusionWidth; x++ {
				if x < pc.puzzleSize {
					shape[y][x] = true
				}
			}
		}
	}

	// 添加右侧凹陷
	indentationY := pc.puzzleSize / 3
	indentationHeight := pc.puzzleSize / 4
	indentationWidth := pc.puzzleSize / 5
	for y := indentationY - indentationHeight/2; y < indentationY+indentationHeight/2; y++ {
		if y >= 0 && y < pc.puzzleSize {
			for x := pc.puzzleSize - indentationWidth; x < pc.puzzleSize; x++ {
				if x >= 0 && x < pc.puzzleSize {
					shape[y][x] = false
				}
			}
		}
	}

	return shape
}

// drawPuzzleHole 在背景图上绘制拼图洞
func drawPuzzleHole(img *image.RGBA, x, y, size int, shape [][]bool, c color.Color) {
	for sy := 0; sy < size; sy++ {
		for sx := 0; sx < size; sx++ {
			if sy < len(shape) && sx < len(shape[sy]) && shape[sy][sx] {
				bgX := x + sx
				bgY := y + sy
				if bgX < img.Bounds().Dx() && bgY < img.Bounds().Dy() {
					img.Set(bgX, bgY, c)
				}
			}
		}
	}
}

// addPuzzleBorder 给拼图块添加边框
func addPuzzleBorder(img *image.RGBA, shape [][]bool) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 添加白色边框
	borderColor := color.RGBA{255, 255, 255, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < len(shape) && x < len(shape[y]) && shape[y][x] {
				// 检查是否是边缘
				isEdge := false
				if y == 0 || y == height-1 || x == 0 || x == width-1 {
					isEdge = true
				} else {
					// 检查周围是否有非拼图区域
					if (y > 0 && (!shape[y-1][x])) ||
						(y < height-1 && (!shape[y+1][x])) ||
						(x > 0 && (!shape[y][x-1])) ||
						(x < width-1 && (!shape[y][x+1])) {
						isEdge = true
					}
				}
				if isEdge {
					img.Set(x, y, borderColor)
				}
			}
		}
	}

	// 添加阴影效果（底部和右侧）
	shadowColor := color.RGBA{0, 0, 0, 100}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < len(shape) && x < len(shape[y]) && shape[y][x] {
				// 右侧阴影
				if x < width-2 && (x+1 >= len(shape[y]) || !shape[y][x+1]) {
					img.Set(x+1, y, shadowColor)
				}
				// 底部阴影
				if y < height-2 && (y+1 >= len(shape) || x >= len(shape[y+1]) || !shape[y+1][x]) {
					img.Set(x, y+1, shadowColor)
				}
			}
		}
	}
}

// imageToBase64 将图片转换为base64
func (pc *PuzzleCaptcha) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// generateID 生成验证码ID
func (pc *PuzzleCaptcha) generateID() string {
	rand.Seed(time.Now().UnixNano())
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var id bytes.Buffer
	for i := 0; i < 32; i++ {
		id.WriteByte(chars[rand.Intn(len(chars))])
	}
	return id.String()
}
