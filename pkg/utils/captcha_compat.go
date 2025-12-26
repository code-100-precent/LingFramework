package utils

import (
	"time"

	"github.com/code-100-precent/LingFramework/pkg/utils/captcha"
)

// 兼容旧接口的验证码类型
type Captcha = captcha.Result
type CaptchaManager = captcha.Manager
type CaptchaStore = captcha.Store
type MemoryCaptchaStore = captcha.MemoryStore

// NewMemoryCaptchaStore 创建内存存储（兼容旧接口）
func NewMemoryCaptchaStore() *MemoryCaptchaStore {
	return captcha.NewMemoryStore()
}

// NewCaptchaManager 创建验证码管理器（兼容旧接口）
// 注意：这个函数返回的是新的统一管理器，但只支持图形验证码
func NewCaptchaManager(width, height, length int, expiration time.Duration, store CaptchaStore) *CaptchaManager {
	config := &captcha.Config{
		ImageWidth:      width,
		ImageHeight:     height,
		ImageLength:     length,
		SliderWidth:     300,
		SliderHeight:    150,
		SliderSize:      50,
		SliderTolerance: 5,
		Expiration:      expiration,
		Store:           store,
	}
	return captcha.NewManager(config)
}

// InitGlobalCaptchaManager 初始化全局验证码管理器（兼容旧接口）
func InitGlobalCaptchaManager(store CaptchaStore) {
	config := &captcha.Config{
		ImageWidth:      200,
		ImageHeight:     60,
		ImageLength:     4,
		SliderWidth:     300,
		SliderHeight:    150,
		SliderSize:      50,
		SliderTolerance: 5,
		Expiration:      5 * time.Minute,
		Store:           store,
	}
	captcha.InitGlobalManager(config)
	GlobalCaptchaManager = captcha.GlobalManager
}

// GlobalCaptchaManager 全局验证码管理器（兼容旧接口）
var GlobalCaptchaManager *CaptchaManager
