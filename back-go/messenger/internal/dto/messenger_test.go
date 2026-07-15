package dto

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// JSON-формы — контракт с фронтом (он не меняется): сверяем дословно с тем,
// что отдавали marshmallow-схемы Flask.

func TestJSONTimePythonIsoformat(t *testing.T) {
	cases := []struct {
		in   time.Time
		want string
	}{
		{time.Date(2026, 6, 12, 10, 0, 0, 123456000, time.UTC), `"2026-06-12T10:00:00.123456+00:00"`},
		{time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC), `"2026-06-12T10:00:00+00:00"`},
	}
	for _, tc := range cases {
		raw, err := json.Marshal(JSONTime(tc.in))
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != tc.want {
			t.Fatalf("формат времени: %s, ожидалось %s", raw, tc.want)
		}
	}
}

func TestMessageJSONShape(t *testing.T) {
	sender := int64(3)
	replySender := int64(2)
	fio := "Алиса"
	text := "привет"
	replyText := "оригинал"
	created := time.Date(2026, 6, 12, 10, 0, 0, 123456000, time.UTC)
	started := time.Date(2026, 6, 12, 9, 58, 0, 0, time.UTC)
	ended := started.Add(65 * time.Second)
	callID := int64(42)

	m := NewMessage(&domain.Message{
		ID:             5,
		ConversationID: 2,
		SenderID:       &sender,
		Text:           &text,
		CreatedAt:      created,
		Kind:           domain.KindCall,
		CallID:         &callID,
		Attachments: []domain.Attachment{{
			ID: 1, FileName: "a.pdf", MimeType: "application/pdf",
			SizeBytes: 10, FilePath: "messages/2026/06/x.pdf",
		}},
		ReplyTo: &domain.ReplyPreview{
			ID: 4, SenderID: &replySender, SenderFIO: &fio, Text: &replyText,
			HasAttachments: true, Kind: domain.KindText,
		},
		Call: &domain.CallInfo{
			ID: 42, Kind: "p2p", Media: "video", Status: "ended",
			StartedAt: started, EndedAt: &ended, InitiatorID: 3,
		},
		ConvIsDevChat: false,
		ConvOwnerID:   2,
	})

	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
		"id": 5,
		"conversation_id": 2,
		"sender_id": 3,
		"is_bot": false,
		"text": "привет",
		"created_at": "2026-06-12T10:00:00.123456+00:00",
		"read_at": null,
		"attachments": [{
			"id": 1,
			"file_name": "a.pdf",
			"mime_type": "application/pdf",
			"size_bytes": 10,
			"url": "/uploads/messages/2026/06/x.pdf",
			"thumb_url": null
		}],
		"reactions": [],
		"reply_to": {
			"id": 4,
			"sender_id": 2,
			"sender_fio": "Алиса",
			"text": "оригинал",
			"has_attachments": true,
			"kind": "text"
		},
		"forwarded_from": null,
		"kind": "call",
		"call": {
			"id": 42,
			"kind": "p2p",
			"media": "video",
			"status": "ended",
			"started_at": "2026-06-12T09:58:00+00:00",
			"ended_at": "2026-06-12T09:59:05+00:00",
			"initiator_id": 3,
			"duration_sec": 65
		},
		"task": null,
		"post": null,
		"pinned_at": null,
		"pinned_by_id": null,
		"edited_at": null,
		"is_from_support": false
	}`
	assertJSONEq(t, raw, want)
}

func TestConversationListItemJSONShape(t *testing.T) {
	company := "ООО Ромашка"
	cid := int64(10)
	item := &ConversationListItem{
		ID:          7,
		UnreadCount: 1,
		IsDevChat:   true,
		CompanyID:   &cid,
		CompanyName: &company,
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
		"id": 7,
		"other_user": null,
		"last_message": null,
		"unread_count": 1,
		"last_message_at": null,
		"is_pinned": false,
		"pinned_at": null,
		"is_dev_chat": true,
		"company_id": 10,
		"company_name": "ООО Ромашка",
		"owner_user": null,
		"is_group": false,
		"title": null,
		"avatar_path": null,
		"member_count": 0,
		"my_role": "",
		"muted": false
	}`
	assertJSONEq(t, raw, want)
}

func TestConversationWithOtherFlattens(t *testing.T) {
	created := time.Date(2026, 6, 12, 8, 0, 0, 0, time.UTC)
	b := int64(3)
	cid := int64(10)
	conv := NewConversation(&domain.Conversation{
		ID: 1, UserAID: 2, UserBID: &b, CompanyID: &cid, CreatedAt: created,
	})
	raw, err := json.Marshal(&ConversationWithOther{Conversation: *conv})
	if err != nil {
		t.Fatal(err)
	}
	want := `{
		"id": 1,
		"user_a_id": 2,
		"user_b_id": 3,
		"created_at": "2026-06-12T08:00:00+00:00",
		"last_message_at": null,
		"is_dev_chat": false,
		"company_id": 10,
		"is_group": false,
		"title": null,
		"avatar_path": null,
		"created_by": null,
		"invite_code": null,
		"member_count": 0,
		"my_role": "",
		"my_muted": false,
		"other_user": null
	}`
	assertJSONEq(t, raw, want)
}

func assertJSONEq(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotAny, wantAny any
	if err := json.Unmarshal(got, &gotAny); err != nil {
		t.Fatalf("got не парсится: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantAny); err != nil {
		t.Fatalf("want не парсится: %v", err)
	}
	if !reflect.DeepEqual(gotAny, wantAny) {
		t.Fatalf("JSON-форма разошлась:\n got: %s\nwant: %s", got, want)
	}
}
