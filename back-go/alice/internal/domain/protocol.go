// Package domain — протокол вебхука Яндекс.Диалогов (навык Алисы) и
// состояние многоходового диалога.
package domain

import "encoding/json"

// WebhookRequest — POST-запрос Диалогов (значимые для нас поля).
type WebhookRequest struct {
	Meta struct {
		Timezone string `json:"timezone"`
	} `json:"meta"`
	Session struct {
		New  bool `json:"new"`
		User struct {
			UserID      string `json:"user_id"`
			AccessToken string `json:"access_token"`
		} `json:"user"`
	} `json:"session"`
	Request struct {
		Type              string `json:"type"` // SimpleUtterance | ButtonPressed | …
		Command           string `json:"command"`
		OriginalUtterance string `json:"original_utterance"`
	} `json:"request"`
	State struct {
		Session json.RawMessage `json:"session"`
	} `json:"state"`
	Version string `json:"version"`
}

// WebhookResponse — ответ навыка. Либо Response (+SessionState), либо
// StartAccountLinking (директива связки аккаунтов — без Response).
type WebhookResponse struct {
	Response            *Response    `json:"response,omitempty"`
	SessionState        *DialogState `json:"session_state,omitempty"`
	StartAccountLinking *struct{}    `json:"start_account_linking,omitempty"`
	Version             string       `json:"version"`
}

type Response struct {
	Text       string `json:"text"`
	TTS        string `json:"tts,omitempty"`
	EndSession bool   `json:"end_session"`
}

// Option — вариант выбора в уточняющем вопросе (отдел, тип юнита, задача…).
type Option struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// EntryOption — вариант записи ежедневника (нужен и id ежедневника).
type EntryOption struct {
	EntryID int64  `json:"entry_id"`
	DiaryID int64  `json:"diary_id"`
	Name    string `json:"name"`
}

// DialogState — состояние многоходового диалога; хранится у Яндекса
// (session_state ответа приходит обратно в state.session) — сервер stateless.
type DialogState struct {
	// Pending — чего ждём от пользователя: choose_department, choose_unit_type,
	// choose_task, choose_diary, choose_note, choose_entry, confirm_close_task,
	// confirm_delete_note, confirm_delete_entry, append_text, move_date.
	Pending string `json:"pending,omitempty"`
	// Kind — исходный интент, ради которого задан уточняющий вопрос.
	Kind string `json:"kind,omitempty"`

	Title   string `json:"title,omitempty"`
	Text    string `json:"text,omitempty"`
	Date    string `json:"date,omitempty"` // YYYY-MM-DD
	TaskID  int64  `json:"task_id,omitempty"`
	NoteID  int64  `json:"note_id,omitempty"`
	DiaryID int64  `json:"diary_id,omitempty"`
	EntryID int64  `json:"entry_id,omitempty"`

	Options      []Option      `json:"options,omitempty"`
	EntryOptions []EntryOption `json:"entry_options,omitempty"`
}
