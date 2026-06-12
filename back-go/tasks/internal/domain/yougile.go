package domain

import "time"

// YougileAccount — привязка пользователя GW к личному API-ключу в YouGile
// (таблица user_yougile_accounts, 1:1 по user_id). Ключ хранится
// зашифрованным Fernet'ом; KeyFingerprint (last4) показываем в UI.
//
// YgCompanyID хранится здесь же, хоть и совпадает с companies.yg_company_id
// на момент подключения — нужно, чтобы при смене настроек компании было
// видно, что персональный коннект «устарел» и его надо переподключить.
type YougileAccount struct {
	ID              int64
	UserID          int64
	CompanyID       int64
	YgCompanyID     string
	YgUserID        *string
	YgLogin         string
	KeyCiphertext   []byte
	KeyFingerprint  string
	LastValidatedAt *time.Time
}

// YougileCompany — YouGile-срез компании: конфигурация интеграции
// (yg_*-колонки companies) + флаг settings.uses_yougile.
type YougileCompany struct {
	ID                  int64
	UsesYougile         bool
	YgCompanyID         *string
	YgCompanyName       *string
	YgProjectID         *string
	YgProjectTitle      *string
	YgBoardID           *string
	YgBoardTitle        *string
	YgFirstColumnID     *string
	YgCompletedColumnID *string
	YgWebhookID         *string
	YgWebhookSecret     *string
}

// YougileAPI — порт тонкого HTTP-клиента YouGile REST v2
// (internal/yougile.Client; в тестах — фейк). Ошибки — типы пакета
// internal/yougile (AuthError различается хелпером yougile.IsAuth).
type YougileAPI interface {
	ListCompanies(login, password string) ([]map[string]any, error)
	CreateKey(login, password, companyID string) (string, error)
	DeleteKey(key string) error
	Me() (map[string]any, error)
	ListProjects(limit int) ([]map[string]any, error)
	ListBoards(projectID string, limit int) ([]map[string]any, error)
	ListColumns(boardID string, limit int) ([]map[string]any, error)
	GetTask(taskID string) (map[string]any, error)
	CreateTask(body map[string]any) (map[string]any, error)
	UpdateTask(taskID string, body map[string]any) (map[string]any, error)
	FindTaskByShortID(boardID, shortID string, columnIDs []string) (map[string]any, error)
	PostChatMessage(chatID string, body map[string]any) error
	CreateWebhook(url, event string, filters []map[string]any) (map[string]any, error)
	UpdateWebhook(webhookID string, body map[string]any) error
}

// YougileCipher — Fernet-шифрование личных API-ключей (YOUGILE_ENC_KEY).
// DecryptKey: "" без ошибки — токен не расшифровался (ключ сменили,
// UI попросит переподключение); ошибка — ключ шифрования не задан/битый.
type YougileCipher interface {
	EncryptKey(plain string) ([]byte, error)
	DecryptKey(enc []byte) (string, error)
}

// YougileRepository — персистентность интеграции: личные аккаунты
// (user_yougile_accounts) + YouGile-поля компаний + поиск привязанной задачи.
type YougileRepository interface {
	// GetYougileAccount — nil, если пользователь не подключён.
	GetYougileAccount(ctx Ctx, userID int64) (*YougileAccount, error)
	// UpsertYougileAccount — создаёт или обновляет запись по user_id.
	UpsertYougileAccount(ctx Ctx, acc *YougileAccount) error
	DeleteYougileAccount(ctx Ctx, userID int64) error

	// GetYougileCompany — nil, если компании нет.
	GetYougileCompany(ctx Ctx, companyID int64) (*YougileCompany, error)
	// UpdateYougileCompanyFields — точечное обновление yg_*-колонок companies.
	UpdateYougileCompanyFields(ctx Ctx, companyID int64, fields map[string]any) error
	// SetCompanyUsesYougile — флаг uses_yougile в settings JSONB.
	SetCompanyUsesYougile(ctx Ctx, companyID int64, enabled bool) error

	// TaskByYougileID — задача компании, привязанная к карточке YG
	// (с подгруженными ссылками, как GetTask); nil — не привязана.
	TaskByYougileID(ctx Ctx, companyID int64, ygTaskID string) (*Task, error)
}
