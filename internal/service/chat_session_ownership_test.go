package service

import (
	"errors"
	"testing"
)

// TestEnsureSessionRejectsForeignOwner 覆盖会话归属修复：
// 当 session_id 已属于用户 A 时，用户 B 不能通过 AddMessages 把消息写进该会话。
func TestEnsureSessionRejectsForeignOwner(t *testing.T) {
	svc := newTestChatSessionService(t)

	const sharedSession = "client-generated-shared-id"

	// 用户 1 首轮对话，创建并拥有该会话。
	if err := svc.AddMessages(sharedSession, 1, "灵山大佛多高", "通高88米", "", 0); err != nil {
		t.Fatalf("owner AddMessages failed: %v", err)
	}

	// 用户 2 用同一个 session_id 写入，必须被拒绝。
	err := svc.AddMessages(sharedSession, 2, "我是入侵者", "intruder", "", 0)
	if !errors.Is(err, ErrSessionAccessDenied) {
		t.Fatalf("foreign write err = %v, want ErrSessionAccessDenied", err)
	}

	// EnsureSession 直接调用也应拒绝非属主。
	if _, err := svc.EnsureSession(sharedSession, 2, "web"); !errors.Is(err, ErrSessionAccessDenied) {
		t.Fatalf("EnsureSession foreign err = %v, want ErrSessionAccessDenied", err)
	}

	// 属主仍可继续写入，且历史里不含入侵者内容。
	if err := svc.AddMessages(sharedSession, 1, "在哪", "太湖之滨", "", 0); err != nil {
		t.Fatalf("owner second AddMessages failed: %v", err)
	}
	msgs, err := svc.GetSessionMessages(sharedSession, 1, 50, 0)
	if err != nil {
		t.Fatalf("GetSessionMessages failed: %v", err)
	}
	for _, m := range msgs {
		if m.UserID != 1 {
			t.Fatalf("found message not owned by user 1: %+v", m)
		}
		if m.Content == "我是入侵者" || m.Content == "intruder" {
			t.Fatalf("intruder content leaked into owner session: %+v", m)
		}
	}
}
