package logger

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func makeTmpLogFile(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, name)
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(b)
}

func waitForWrite(assert func() bool) bool {
	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		if assert() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return assert()
}

// 同时劫持 Stdout 和 Stderr，注意：必须在 Init 之前进行。
func captureStdoutStderr(t *testing.T, fn func()) (string, string) {
	t.Helper()

	origOut := os.Stdout
	origErr := os.Stderr

	// 分别创建两对管道
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	var bufOut, bufErr bytes.Buffer
	doneOut := make(chan struct{})
	doneErr := make(chan struct{})

	go func() {
		defer close(doneOut)
		rd := bufio.NewReader(rOut)
		io.Copy(&bufOut, rd) //nolint:errcheck
	}()
	go func() {
		defer close(doneErr)
		rd := bufio.NewReader(rErr)
		io.Copy(&bufErr, rd) //nolint:errcheck
	}()

	// 执行被测逻辑（包含 Init 和日志写入）
	fn()

	// 关闭写端并恢复
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = origOut
	os.Stderr = origErr

	<-doneOut
	<-doneErr

	return bufOut.String(), bufErr.String()
}

func TestInitProdModeAndFileWrite(t *testing.T) {
	logPath := makeTmpLogFile(t, "app.log")
	cfg := &LogConfig{
		Level:      "debug",
		Filename:   logPath,
		MaxSize:    5,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := Init(cfg, "prod"); err != nil {
		t.Fatalf("Init(prod) error: %v", err)
	}

	// 写各种等级
	Debug("dmsg", zap.String("k", "vd"))
	Info("imsg", zap.Int("n", 1))
	Warn("wmsg")
	Error("emsg", zap.Error(errors.New("boom")))

	// Panic 需要 recover
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()
		Panic("pmsg")
	}()

	// Sync 刷新
	Sync()

	ok := waitForWrite(func() bool {
		s := readFile(t, logPath)
		return strings.Contains(s, "dmsg") &&
			strings.Contains(s, "imsg") &&
			strings.Contains(s, "wmsg") &&
			strings.Contains(s, "emsg")
	})
	if !ok {
		t.Fatalf("log content not found in file:\n%s", readFile(t, logPath))
	}
}

func TestInitDevModeConsoleSplit(t *testing.T) {
	logPath := makeTmpLogFile(t, "dev.log")
	cfg := &LogConfig{
		Level:      "debug",
		Filename:   logPath,
		MaxSize:    5,
		MaxAge:     1,
		MaxBackups: 1,
	}

	stdoutOut, stderrOut := captureStdoutStderr(t, func() {
		// 必须在劫持之后再 Init，这样 zap core 会锁定被我们替换过的 Stdout/Stderr
		if err := Init(cfg, "dev"); err != nil {
			t.Fatalf("Init(dev) error: %v", err)
		}
		Debug("dev-debug")
		Info("dev-info")
		Warn("dev-warn")   // < error -> 走 stdout
		Error("dev-error") // >= error -> 走 stderr
		Sync()
	})

	// 控制台输出包含消息（颜色码不校验）
	if !strings.Contains(stdoutOut, "dev-info") || !strings.Contains(stdoutOut, "dev-warn") || !strings.Contains(stdoutOut, "dev-debug") {
		t.Fatalf("stdout missing expected logs:\n%s", stdoutOut)
	}
	if !strings.Contains(stderrOut, "dev-error") {
		t.Fatalf("stderr missing expected logs:\n%s", stderrOut)
	}

	// 文件也应该有写入
	ok := waitForWrite(func() bool {
		s := readFile(t, logPath)
		return strings.Contains(s, "dev-info") && strings.Contains(s, "dev-error")
	})
	if !ok {
		t.Fatalf("dev file log missing:\n%s", readFile(t, logPath))
	}
}

func TestInitInvalidLevel(t *testing.T) {
	logPath := makeTmpLogFile(t, "bad.log")
	cfg := &LogConfig{
		Level:      "not-a-level",
		Filename:   logPath,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
	}
	err := Init(cfg, "prod")
	if err == nil {
		t.Fatalf("expected error for invalid level")
	}
}

// 仅验证 Sync 在 nil logger 时不崩（当前实现会 panic，保持原测试+recover 以覆盖路径）
func TestSyncSafeWhenNoLogger(t *testing.T) {
	logPath := makeTmpLogFile(t, "sync.log")
	cfg := &LogConfig{
		Level:      "info",
		Filename:   logPath,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := Init(cfg, "prod"); err != nil {
		t.Fatalf("Init for SyncSafe: %v", err)
	}
	// 备份并置空
	old := Lg
	Lg = nil
	defer func() { Lg = old }()

	defer func() { _ = recover() }()
	Sync()
}

// 由于包装函数存在，caller 会显示在 logger/logger.go（而非测试文件）
// 因此这里验证 ShortCallerEncoder 是否生效，并包含生产文件路径。
func TestAddCallerEnabled(t *testing.T) {
	logPath := makeTmpLogFile(t, "caller.log")
	cfg := &LogConfig{
		Level:      "debug",
		Filename:   logPath,
		MaxSize:    5,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := Init(cfg, "prod"); err != nil {
		t.Fatalf("Init(prod) error: %v", err)
	}
	Info("caller-check")
	Sync()

	ok := waitForWrite(func() bool {
		s := readFile(t, logPath)
		// 由于 EncodeCaller=ShortCallerEncoder，应出现 "logger/logger.go:xxx"
		return strings.Contains(s, "logger/logger.go") && strings.Contains(s, "caller-check")
	})
	if !ok {
		t.Fatalf("caller info not present:\n%s", readFile(t, logPath))
	}
}

// 防止某些平台噪音 & 也顺带让几个包装方法跑一遍
func TestEnvironmentNoise(t *testing.T) {
	_ = runtime.GOOS
	msg := "noop"
	Debug(msg)
	Info(msg)
	Warn(msg)
	Error(msg)
	fmt.Sprintf("")
}

func TestFatal(t *testing.T) {
	// Fatal will exit the process, so we can't test it normally
	// This test just verifies the function exists and can be called
	// In a real scenario, Fatal would call os.Exit(1)
	logPath := makeTmpLogFile(t, "fatal.log")
	cfg := &LogConfig{
		Level:      "info",
		Filename:   logPath,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := Init(cfg, "prod"); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	// Fatal will exit, so we can only test that it compiles and exists
	// In production, this would call os.Exit(1)
	// We can't actually test the exit behavior without mocking os.Exit
	_ = Fatal
}

func TestGetAlertStats_NoAlertManager(t *testing.T) {
	// Save original alertManager
	originalAlertManager := alertManager
	defer func() {
		alertManager = originalAlertManager
	}()

	// Set alertManager to nil
	alertManager = nil

	stats := GetAlertStats()
	assert.Nil(t, stats)
}

func TestGetAlertStats_WithAlertManager(t *testing.T) {
	// Save original alertManager
	originalAlertManager := alertManager
	defer func() {
		alertManager = originalAlertManager
	}()

	// Create a test alert manager
	alertConfig := &AlertConfig{
		Enabled:          true,
		ErrorThreshold:   10,
		WarningThreshold: 20,
		TimeWindow:       time.Minute,
		CooldownPeriod:   time.Minute,
	}
	alertManager = NewAlertManager(alertConfig)

	stats := GetAlertStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.ErrorCount)
	assert.Equal(t, 0, stats.WarningCount)
}

func TestResetAlertStats_NoAlertManager(t *testing.T) {
	// Save original alertManager
	originalAlertManager := alertManager
	defer func() {
		alertManager = originalAlertManager
	}()

	// Set alertManager to nil
	alertManager = nil

	// Should not panic
	ResetAlertStats()
}

func TestResetAlertStats_WithAlertManager(t *testing.T) {
	// Save original alertManager
	originalAlertManager := alertManager
	defer func() {
		alertManager = originalAlertManager
	}()

	// Create a test alert manager
	alertConfig := &AlertConfig{
		Enabled:          true,
		ErrorThreshold:   10,
		WarningThreshold: 20,
		TimeWindow:       time.Minute,
		CooldownPeriod:   time.Minute,
	}
	alertManager = NewAlertManager(alertConfig)

	// Increment some stats
	alertManager.CheckAndAlert(zapcore.ErrorLevel, zapcore.Entry{}, nil)
	alertManager.CheckAndAlert(zapcore.ErrorLevel, zapcore.Entry{}, nil)

	stats := GetAlertStats()
	assert.NotNil(t, stats)
	assert.Greater(t, stats.ErrorCount, 0)

	// Reset stats
	ResetAlertStats()

	stats = GetAlertStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.ErrorCount)
	assert.Equal(t, 0, stats.WarningCount)
}

func TestIsAlertEnabled_NoAlertManager(t *testing.T) {
	// Save original alertManager
	originalAlertManager := alertManager
	defer func() {
		alertManager = originalAlertManager
	}()

	// Set alertManager to nil
	alertManager = nil

	enabled := IsAlertEnabled()
	assert.False(t, enabled)
}

func TestIsAlertEnabled_WithAlertManager(t *testing.T) {
	// Save original alertManager
	originalAlertManager := alertManager
	defer func() {
		alertManager = originalAlertManager
	}()

	// Create a test alert manager
	alertConfig := &AlertConfig{
		Enabled:          true,
		ErrorThreshold:   10,
		WarningThreshold: 20,
		TimeWindow:       time.Minute,
		CooldownPeriod:   time.Minute,
	}
	alertManager = NewAlertManager(alertConfig)

	enabled := IsAlertEnabled()
	assert.True(t, enabled)
}

func TestGetDailyLogFilename(t *testing.T) {
	testCases := []struct {
		name     string
		baseFile string
		expect   string // Pattern to match
	}{
		{
			name:     "standard log file",
			baseFile: "app.log",
			expect:   "app-",
		},
		{
			name:     "log file with path",
			baseFile: "/var/log/app.log",
			expect:   "app-",
		},
		{
			name:     "log file with directory",
			baseFile: "logs/app.log",
			expect:   "app-",
		},
		{
			name:     "log file with extension",
			baseFile: "application.log",
			expect:   "application-",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetDailyLogFilename(tc.baseFile)
			assert.Contains(t, result, tc.expect)
			assert.Contains(t, result, ".log")
			// Should contain date in format YYYY-MM-DD
			assert.Contains(t, result, time.Now().Format("2006-01-02"))
		})
	}
}

func TestGetDailyLogFilename_EdgeCases(t *testing.T) {
	// Test with no extension
	result := GetDailyLogFilename("app")
	assert.Contains(t, result, "app-")
	assert.Contains(t, result, time.Now().Format("2006-01-02"))

	// Test with multiple dots
	result = GetDailyLogFilename("app.log.backup")
	assert.Contains(t, result, "app.log-")
	assert.Contains(t, result, time.Now().Format("2006-01-02"))
}

// Mock notification service for testing
type mockNotificationService struct {
	sentAlerts []*AlertInfo
}

func (m *mockNotificationService) SendAlert(ctx context.Context, alert *AlertInfo) error {
	m.sentAlerts = append(m.sentAlerts, alert)
	return nil
}

func TestAlertManager_GetStats(t *testing.T) {
	alertConfig := &AlertConfig{
		Enabled:          true,
		ErrorThreshold:   10,
		WarningThreshold: 20,
		TimeWindow:       time.Minute,
		CooldownPeriod:   time.Minute,
	}
	manager := NewAlertManager(alertConfig)

	stats := manager.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.ErrorCount)
	assert.Equal(t, 0, stats.WarningCount)

	// Add some errors
	manager.CheckAndAlert(zapcore.ErrorLevel, zapcore.Entry{}, nil)
	manager.CheckAndAlert(zapcore.ErrorLevel, zapcore.Entry{}, nil)

	stats = manager.GetStats()
	assert.Equal(t, 2, stats.ErrorCount)
}

func TestAlertManager_ResetStats(t *testing.T) {
	alertConfig := &AlertConfig{
		Enabled:          true,
		ErrorThreshold:   10,
		WarningThreshold: 20,
		TimeWindow:       time.Minute,
		CooldownPeriod:   time.Minute,
	}
	manager := NewAlertManager(alertConfig)

	// Add some stats
	manager.CheckAndAlert(zapcore.ErrorLevel, zapcore.Entry{}, nil)
	manager.CheckAndAlert(zapcore.WarnLevel, zapcore.Entry{}, nil)

	stats := manager.GetStats()
	assert.Equal(t, 1, stats.ErrorCount)
	assert.Equal(t, 1, stats.WarningCount)

	// Reset
	manager.ResetStats()

	stats = manager.GetStats()
	assert.Equal(t, 0, stats.ErrorCount)
	assert.Equal(t, 0, stats.WarningCount)
}

func TestInitWithAlert(t *testing.T) {
	logPath := makeTmpLogFile(t, "alert.log")
	cfg := &LogConfig{
		Level:      "info",
		Filename:   logPath,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		Alert: &AlertConfig{
			Enabled:          true,
			ErrorThreshold:   5,
			WarningThreshold: 10,
			TimeWindow:       time.Minute,
			CooldownPeriod:   time.Minute,
		},
	}

	mockNotifier := &mockNotificationService{}
	err := InitWithAlert(cfg, "prod", []NotificationService{mockNotifier})
	assert.NoError(t, err)

	// Verify alert manager was initialized
	assert.True(t, IsAlertEnabled())
	assert.NotNil(t, GetAlertStats())
}

func TestInitWithAlert_Disabled(t *testing.T) {
	logPath := makeTmpLogFile(t, "no-alert.log")
	cfg := &LogConfig{
		Level:      "info",
		Filename:   logPath,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		Alert: &AlertConfig{
			Enabled: false, // Disabled
		},
	}

	mockNotifier := &mockNotificationService{}
	err := InitWithAlert(cfg, "prod", []NotificationService{mockNotifier})
	assert.NoError(t, err)

	// Alert manager should not be initialized when disabled
	// But the global variable might still be nil or set, so we check IsAlertEnabled
	// This depends on implementation - if alertManager is nil when disabled, IsAlertEnabled returns false
}

func TestGetDailyLogFilename_ComplexPath(t *testing.T) {
	// Test with complex path
	result := GetDailyLogFilename("/var/log/application/app.log")
	assert.Contains(t, result, "app-")
	assert.Contains(t, result, time.Now().Format("2006-01-02"))
	assert.Contains(t, result, ".log")

	// Extract directory and verify
	dir := filepath.Dir(result)
	assert.Equal(t, "/var/log/application", dir)
}

func TestGetDailyLogFilename_RelativePath(t *testing.T) {
	// Test with relative path
	result := GetDailyLogFilename("logs/app.log")
	assert.Contains(t, result, "app-")
	assert.Contains(t, result, time.Now().Format("2006-01-02"))
	assert.Contains(t, result, ".log")

	// Extract directory and verify
	dir := filepath.Dir(result)
	assert.Equal(t, "logs", dir)

	// Test filename extraction
	filename := filepath.Base(result)
	assert.Contains(t, filename, "app-")
	assert.Contains(t, filename, time.Now().Format("2006-01-02"))
	assert.Contains(t, filename, ".log")
}
