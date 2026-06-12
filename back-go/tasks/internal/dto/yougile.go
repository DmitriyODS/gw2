package dto

// JSON-формы /api/yougile — байт-в-байт со схемами Flask
// (schemas/yougile.py); порядок полей алфавитный, как jsonify.

// YougileStatus — GET /api/yougile/status (YougileAccountStatusSchema).
type YougileStatus struct {
	CompanyEnabled  bool      `json:"company_enabled"`
	Connected       bool      `json:"connected"`
	KeyFingerprint  *string   `json:"key_fingerprint"`
	LastValidatedAt *JSONTime `json:"last_validated_at"`
	YgCompanyID     *string   `json:"yg_company_id"`
	YgLogin         *string   `json:"yg_login"`
}

// YougileConnectResult — POST /api/yougile/account.
type YougileConnectResult struct {
	Connected      bool   `json:"connected"`
	KeyFingerprint string `json:"key_fingerprint"`
	YgCompanyID    string `json:"yg_company_id"`
	YgLogin        string `json:"yg_login"`
}

// YougileRotateResult — POST /api/yougile/account/rotate.
type YougileRotateResult struct {
	Connected      bool   `json:"connected"`
	KeyFingerprint string `json:"key_fingerprint"`
}

// YougileCompanyItem — элемент POST /api/yougile/companies/lookup.
type YougileCompanyItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// YougileProject / YougileBoard / YougileColumn — каталоги админ-визарда.
type YougileProject struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type YougileBoard struct {
	ID        string  `json:"id"`
	ProjectID *string `json:"projectId"`
	Title     string  `json:"title"`
}

type YougileColumn struct {
	BoardID *string `json:"boardId"`
	ID      string  `json:"id"`
	Title   string  `json:"title"`
}

// YougileSettings — GET/PUT /api/yougile/company-settings и POST /reset
// (YougileCompanySettingsSchema).
type YougileSettings struct {
	Enabled             bool    `json:"enabled"`
	WebhookRegistered   bool    `json:"webhook_registered"`
	YgBoardID           *string `json:"yg_board_id"`
	YgBoardTitle        *string `json:"yg_board_title"`
	YgCompanyID         *string `json:"yg_company_id"`
	YgCompanyName       *string `json:"yg_company_name"`
	YgCompletedColumnID *string `json:"yg_completed_column_id"`
	YgFirstColumnID     *string `json:"yg_first_column_id"`
	YgProjectID         *string `json:"yg_project_id"`
	YgProjectTitle      *string `json:"yg_project_title"`
}

// ── Запросы (после schema-валидации в транспорте) ────────────────

// YougileConnect — POST /api/yougile/account (YougileConnectFinishSchema).
type YougileConnect struct {
	Login       string
	Password    string
	YgCompanyID *string
}

// YougileSettingsUpdate — PUT /company-settings: все поля опциональны,
// *Set = поле передано (частичное обновление, как "key in payload").
type YougileSettingsUpdate struct {
	Enabled                bool
	EnabledSet             bool
	YgCompanyID            *string
	YgCompanyIDSet         bool
	YgCompanyName          *string
	YgCompanyNameSet       bool
	YgProjectID            *string
	YgProjectIDSet         bool
	YgProjectTitle         *string
	YgProjectTitleSet      bool
	YgBoardID              *string
	YgBoardIDSet           bool
	YgBoardTitle           *string
	YgBoardTitleSet        bool
	YgCompletedColumnID    *string
	YgCompletedColumnIDSet bool
}

// YougileImport — POST /api/yougile/import-task (YougileImportTaskSchema).
type YougileImport struct {
	URL               string
	DepartmentID      int64
	ResponsibleUserID *int64
	StageID           *int64
	PullDeadline      bool
}
