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

func TestAddMessageCreatesOwnedSessionAndPersistsSingleMessage(t *testing.T) {
	service := newTestChatSessionService(t)

	if err := service.AddMessage("manual-session", 7, "user", "我要看灵山大佛", "", 0); err != nil {
		t.Fatalf("AddMessage returned error: %v", err)
	}

	messages, err := service.GetSessionMessages("manual-session", 7, 50, 0)
	if err != nil {
		t.Fatalf("GetSessionMessages returned error: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "我要看灵山大佛" || messages[0].UserID != 7 {
		t.Fatalf("unexpected persisted message: %+v", messages[0])
	}

	sessions, total, err := service.ListSessions(7, 1, 20)
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if total != 1 || len(sessions) != 1 {
		t.Fatalf("sessions total=%d len=%d, want 1", total, len(sessions))
	}
	if sessions[0].Title != "我要看灵山大佛" || sessions[0].MessageCount != 1 {
		t.Fatalf("unexpected session metadata: %+v", sessions[0])
	}
}

func TestAddMessageRejectsInvalidRoleAndEmptyContent(t *testing.T) {
	service := newTestChatSessionService(t)

	if err := service.AddMessage("manual-session", 7, "debug", "内容", "", 0); err == nil {
		t.Fatal("AddMessage accepted invalid role")
	}
	if err := service.AddMessage("manual-session", 7, "user", "   ", "", 0); err == nil {
		t.Fatal("AddMessage accepted empty content")
	}
}
