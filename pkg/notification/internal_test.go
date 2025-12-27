package notification

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupInternalTestDB(t *testing.T) *gorm.DB {
	silentLogger := logger.New(
		log.New(io.Discard, "", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: silentLogger,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&InternalNotification{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db
}

func TestNewInternalNotificationAdapter(t *testing.T) {
	db := setupInternalTestDB(t)
	service := NewInternalNotificationService(db)
	adapter := NewInternalNotificationAdapter(service)

	assert.NotNil(t, adapter)
	assert.Equal(t, service, adapter.service)
}

func TestInternalNotificationAdapter_Type(t *testing.T) {
	db := setupInternalTestDB(t)
	service := NewInternalNotificationService(db)
	adapter := NewInternalNotificationAdapter(service)

	assert.Equal(t, TypeInternal, adapter.Type())
}

func TestInternalNotificationAdapter_Send(t *testing.T) {
	db := setupInternalTestDB(t)
	service := NewInternalNotificationService(db)
	adapter := NewInternalNotificationAdapter(service)

	// Test with empty recipients
	req := NotificationRequest{
		Type:    TypeInternal,
		Title:   "Test",
		Content: "Test Content",
		To:      []string{},
	}

	err := adapter.Send(context.Background(), req)
	assert.Equal(t, ErrInvalidRecipient, err)

	// Test with valid user ID
	req = NotificationRequest{
		Type:    TypeInternal,
		Title:   "Test Title",
		Content: "Test Content",
		To:      []string{"1"},
	}

	err = adapter.Send(context.Background(), req)
	assert.NoError(t, err)

	// Verify notification was created
	var notification InternalNotification
	err = db.Where("user_id = ?", 1).First(&notification).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test Title", notification.Title)
	assert.Equal(t, "Test Content", notification.Content)

	// Test with invalid user ID
	req = NotificationRequest{
		Type:    TypeInternal,
		Title:   "Test",
		Content: "Test Content",
		To:      []string{"invalid"},
	}

	err = adapter.Send(context.Background(), req)
	assert.Error(t, err)

	// Test with multiple user IDs
	req = NotificationRequest{
		Type:    TypeInternal,
		Title:   "Test",
		Content: "Test Content",
		To:      []string{"2", "3"},
	}

	err = adapter.Send(context.Background(), req)
	assert.NoError(t, err)

	// Verify notifications were created
	var count int64
	db.Model(&InternalNotification{}).Where("user_id IN ?", []uint{2, 3}).Count(&count)
	assert.Equal(t, int64(2), count)
}
