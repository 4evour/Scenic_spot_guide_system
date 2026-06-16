package service

import (
	"database/sql"
	"strings"
	"sync"
	"testing"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerChatSessionTestDriver sync.Once

func newTestChatSessionService(t *testing.T) *ChatSessionService {
	t.Helper()

	const driverName = "modernc-chat-session-test"
	registerChatSessionTestDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	return NewChatSessionService(
		repository.NewChatSessionRepository(db),
		repository.NewChatMessageRepository(db),
	)
}

func TestGetSessionMessagesReturnsEmptyForUnknownSession(t *testing.T) {
	service := newTestChatSessionService(t)

	messages, err := service.GetSessionMessages("new-local-session", 1, 50, 0)
	if err != nil {
		t.Fatalf("GetSessionMessages returned error: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("messages length = %d, want 0", len(messages))
	}
}
