package domain

import "encoding/json"

// Формы data.json резервной копии. Покрывают идентичность (роли, пользователи,
// компании, членства) и ядро учёта задач (отделы, этапы, типы юнитов, задачи,
// избранное, юниты). Даты — строки isoformat (или null); при импорте их парсит
// сам PostgreSQL. Порядок полей в BackupData = порядок ключей в JSON.
//
// Состав обязан соответствовать текущей мультитенантной схеме: companies и
// company_id у company-scoped таблиц — NOT NULL, без них восстановление падает
// на внешних ключах.

type BackupData struct {
	Roles         []BackupRole       `json:"roles"`
	Users         []BackupUser       `json:"users"`
	Companies     []BackupCompany    `json:"companies"`
	UserCompanies []BackupMembership `json:"user_companies"`
	Departments   []BackupDepartment `json:"departments"`
	Stages        []BackupStage      `json:"stages"`
	UnitTypes     []BackupUnitType   `json:"unit_types"`
	Tasks         []BackupTask       `json:"tasks"`
	Favorites     []BackupFavorite   `json:"favorites"`
	Units         []BackupUnit       `json:"units"`
}

// BackupMembership — связка user↔company с ролью и должностью в этой компании.
type BackupMembership struct {
	UserID    int64   `json:"user_id"`
	CompanyID int64   `json:"company_id"`
	RoleID    int64   `json:"role_id"`
	Post      *string `json:"post"`
	CreatedAt *string `json:"created_at"`
}

type BackupRole struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type BackupUser struct {
	ID            int64   `json:"id"`
	FIO           string  `json:"fio"`
	Login         string  `json:"login"`
	HashPassword  string  `json:"hash_password"`
	AvatarPath    *string `json:"avatar_path"`
	IsDefaultPass bool    `json:"is_default_pass"`
	IsActive      bool    `json:"is_active"`
	IsSuperAdmin  bool    `json:"is_super_admin"`
	CreatedAt     *string `json:"created_at"`
}

// BackupCompany — компания со всеми настройками (включая зашифрованный ключ ИИ
// и привязку YouGile). settings — сырой JSON, ai_api_key_enc — bytea (base64 в
// JSON).
type BackupCompany struct {
	ID                  int64           `json:"id"`
	Name                string          `json:"name"`
	Description         *string         `json:"description"`
	IsActive            bool            `json:"is_active"`
	Settings            json.RawMessage `json:"settings"`
	CreatedAt           *string         `json:"created_at"`
	AIEnabled           bool            `json:"ai_enabled"`
	AIAPIKeyEnc         []byte          `json:"ai_api_key_enc"`
	AIKeyHint           *string         `json:"ai_key_hint"`
	AIModelChat         string          `json:"ai_model_chat"`
	AIModelEmbedding    string          `json:"ai_model_embedding"`
	YgCompanyID         *string         `json:"yg_company_id"`
	YgCompanyName       *string         `json:"yg_company_name"`
	YgProjectID         *string         `json:"yg_project_id"`
	YgProjectTitle      *string         `json:"yg_project_title"`
	YgBoardID           *string         `json:"yg_board_id"`
	YgBoardTitle        *string         `json:"yg_board_title"`
	YgFirstColumnID     *string         `json:"yg_first_column_id"`
	YgCompletedColumnID *string         `json:"yg_completed_column_id"`
	YgWebhookID         *string         `json:"yg_webhook_id"`
	YgWebhookSecret     *string         `json:"yg_webhook_secret"`
	InviteCode          *string         `json:"invite_code"`
	CreatedBy           *int64          `json:"created_by"`
}

type BackupDepartment struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CompanyID int64  `json:"company_id"`
}

type BackupStage struct {
	ID        int64  `json:"id"`
	CompanyID int64  `json:"company_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Order     int    `json:"order"`
}

type BackupTask struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	AuthorID          int64   `json:"author_id"`
	LinkYougile       *string `json:"link_yougile"`
	ReceivedAt        *string `json:"received_at"`
	DepartmentID      int64   `json:"department_id"`
	Deadline          *string `json:"deadline"`
	IsArchived        bool    `json:"is_archived"`
	ArchivedAt        *string `json:"archived_at"`
	CreatedAt         *string `json:"created_at"`
	Color             *string `json:"color"`
	CompanyID         int64   `json:"company_id"`
	ResponsibleUserID *int64  `json:"responsible_user_id"`
	StageID           *int64  `json:"stage_id"`
	YougileTaskID     *string `json:"yougile_task_id"`
	YougileProjectID  *string `json:"yougile_project_id"`
	YougileBoardID    *string `json:"yougile_board_id"`
	YougileColumnID   *string `json:"yougile_column_id"`
	YougileSyncedAt   *string `json:"yougile_synced_at"`
	YougileSyncHash   *string `json:"yougile_sync_hash"`
	YougileIDShort    *string `json:"yougile_id_short"`
}

type BackupFavorite struct {
	TaskID int64 `json:"task_id"`
	UserID int64 `json:"user_id"`
}

type BackupUnitType struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CompanyID int64  `json:"company_id"`
}

type BackupUnit struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	UserID        int64   `json:"user_id"`
	UnitTypeID    int64   `json:"unit_type_id"`
	TaskID        int64   `json:"task_id"`
	IsEdited      bool    `json:"is_edited"`
	DatetimeStart *string `json:"datetime_start"`
	DatetimeEnd   *string `json:"datetime_end"`
	CreatedAt     *string `json:"created_at"`
	CompanyID     int64   `json:"company_id"`
}

// AvatarFile — файл аватарки в архиве (avatars/<name>).
type AvatarFile struct {
	Name string
	Data []byte
}
