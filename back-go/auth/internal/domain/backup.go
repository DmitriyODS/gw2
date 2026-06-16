package domain

// Формы data.json резервной копии — байт-в-байт совместимы с прежним
// back/app/services/backup_service.py (порядок ключей = порядок полей).
// Даты — строки isoformat (или null), как _serialize_dt; при импорте их
// парсит сам PostgreSQL.

type BackupData struct {
	Roles         []BackupRole       `json:"roles"`
	Users         []BackupUser       `json:"users"`
	UserCompanies []BackupMembership `json:"user_companies"`
	Departments   []BackupDepartment `json:"departments"`
	Tasks         []BackupTask       `json:"tasks"`
	Favorites     []BackupFavorite   `json:"favorites"`
	UnitTypes     []BackupUnitType   `json:"unit_types"`
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

type BackupDepartment struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type BackupTask struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	AuthorID     int64   `json:"author_id"`
	LinkYougile  *string `json:"link_yougile"`
	ReceivedAt   *string `json:"received_at"`
	DepartmentID int64   `json:"department_id"`
	Deadline     *string `json:"deadline"`
	IsArchived   bool    `json:"is_archived"`
	ArchivedAt   *string `json:"archived_at"`
	CreatedAt    *string `json:"created_at"`
}

type BackupFavorite struct {
	TaskID int64 `json:"task_id"`
	UserID int64 `json:"user_id"`
}

type BackupUnitType struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
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
}

// AvatarFile — файл аватарки в архиве (avatars/<name>).
type AvatarFile struct {
	Name string
	Data []byte
}
