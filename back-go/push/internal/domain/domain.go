// Package domain — модели и порты pushsvc (отправка пуш-уведомлений FCM).
package domain

import "context"

// DeviceToken — токен устройства пользователя для FCM.
type DeviceToken struct {
	Token    string
	UserID   int64
	Platform string
}

// Notification — готовое пуш-уведомление для одного получателя.
// Channel — id канала уведомлений на клиенте (messages/tasks/calls).
// Data — произвольные ключи (на клиенте читаются из data-сообщения для
// навигации и полноэкранного звонка). HighPriority — для звонков: будит
// устройство немедленно и доставляется даже в Doze.
type Notification struct {
	UserID       int64
	Title        string
	Body         string
	Channel      string
	Data         map[string]string
	HighPriority bool
}

// Каналы уведомлений — синхронизированы с Notifier на Android.
const (
	ChannelMessages = "messages"
	ChannelTasks    = "tasks"
	ChannelCalls    = "calls"
	ChannelKudos    = "kudos"
	ChannelPortal   = "portal"
	ChannelPets     = "pets"
)

// TokenStore — хранилище токенов устройств.
type TokenStore interface {
	Upsert(ctx context.Context, t DeviceToken) error
	Delete(ctx context.Context, token string) error
	// ListByUsers — токены указанных пользователей (для рассылки).
	ListByUsers(ctx context.Context, userIDs []int64) ([]DeviceToken, error)
}

// UserDirectory — имена пользователей для заголовков уведомлений.
type UserDirectory interface {
	// Names — ФИО по id (отсутствующие просто не попадают в map).
	Names(ctx context.Context, ids []int64) (map[int64]string, error)
	// MembersOf — id участников компании (company-wide события: пост портала
	// адресован всей компании, а не комнате user_{id}).
	MembersOf(ctx context.Context, companyID int64) ([]int64, error)
}

// Presence — кто сейчас онлайн (живой WS). Онлайн-получателям пуш не шлём:
// приложение на переднем плане покажет событие вживую (FCM-first).
type Presence interface {
	// Offline — подмножество ids, которых НЕТ в онлайне.
	Offline(ctx context.Context, ids []int64) ([]int64, error)
}

// Sender — отправка одного уведомления на токен. Возвращает invalid=true,
// если токен протух/недействителен (его надо удалить из хранилища).
type Sender interface {
	Enabled() bool
	Send(ctx context.Context, token string, n Notification) (invalid bool, err error)
}
