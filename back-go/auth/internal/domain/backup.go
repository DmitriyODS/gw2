package domain

import "encoding/json"

// Резервная копия (data.json в ZIP) — универсальный схемо-независимый дамп:
// таблицы обнаруживаются из БД на лету, поэтому новые таблицы попадают в бэкап
// автоматически. Каждая таблица сериализуется как JSON-массив строк (to_jsonb),
// при импорте раскрывается обратно через jsonb_populate_recordset — типы
// (timestamptz/jsonb/bytea/массивы) PostgreSQL восстанавливает сам.

const BackupVersion = 2

// BackupArchive — содержимое data.json.
type BackupArchive struct {
	Version  int                        `json:"version"`
	Sections []string                   `json:"sections"`
	Tables   map[string]json.RawMessage `json:"tables"`
}

// BackupSection — раздел выбора в модалке экспорта/восстановления.
type BackupSection struct {
	Key    string
	Tables []string
}

// SectionOther — псевдо-раздел для таблиц, не попавших ни в один из явных
// разделов (например, добавленных позже). Так новые таблицы не теряются молча.
const SectionOther = "other"

// BackupSections — статическая карта «раздел → таблицы». Состав таблиц
// синхронизирован с фронтом (front/src/utils/backupSections.js, только ключи и
// подписи). Любая таблица БД не из этого списка и не из BackupExcluded попадает
// в раздел SectionOther.
var BackupSections = []BackupSection{
	{Key: "auth", Tables: []string{"roles", "users", "user_companies", "device_tokens"}},
	{Key: "companies", Tables: []string{"companies", "company_invites"}},
	{Key: "tasks", Tables: []string{"departments", "stages", "unit_types", "tasks", "favorites", "units", "comments", "user_task_colors"}},
	{Key: "registry", Tables: []string{"registries", "registry_fields", "registry_records", "registry_shares"}},
	{Key: "calendar", Tables: []string{"calendars", "calendar_fields", "calendar_records", "calendar_shares"}},
	{Key: "diary", Tables: []string{"diaries", "diary_records", "diary_shares", "diary_user_shares"}},
	{Key: "messenger", Tables: []string{"conversations", "messages", "message_attachments"}},
	{Key: "calls", Tables: []string{"calls", "call_participants"}},
	{Key: "groove", Tables: []string{"feed_events", "feed_comments", "feed_reactions", "groove_raids", "pets", "pet_strokes", "user_locations"}},
	{Key: "integration", Tables: []string{"user_yougile_accounts"}},
}

// BackupExcluded — таблицы, которые НИКОГДА не бэкапим: транзиентные (коды
// подтверждения/сброса) и перегенерируемые (эмбеддинги), плюс служебная таблица
// миграций goose.
var BackupExcluded = map[string]bool{
	"email_verifications": true,
	"password_resets":     true,
	"task_embeddings":     true,
	"goose_db_version":    true,
}

// AvatarFile — файл аватарки в архиве (avatars/<name>). Аватарки входят в раздел
// "auth".
type AvatarFile struct {
	Name string
	Data []byte
}
