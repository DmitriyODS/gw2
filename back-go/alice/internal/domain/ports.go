package domain

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
)

// Error — общая бизнес-ошибка платформы (pkg/apierror): коды из полей Error
// gRPC-ответов сервисов-владельцев доезжают сюда как есть.
type Error = apierror.Error

func NewError(code, message string, httpStatus int) *Error {
	return apierror.New(code, message, httpStatus)
}

// AsDomainError — достать *Error из цепочки; nil, если это не бизнес-ошибка.
func AsDomainError(err error) *Error { return apierror.As(err) }

// ── Модели ответов сервисов-владельцев (минимум для голосовых реплик) ──

type CatalogItem struct {
	ID   int64
	Name string
}

type TaskRef struct {
	ID   int64
	Name string
}

type ActiveUnit struct {
	UnitID   int64
	TaskID   int64
	UnitName string
	TaskName string
	Minutes  int
}

type StoppedUnit struct {
	UnitName string
	TaskName string
	Minutes  int
}

type Diary struct {
	ID          int64
	Name        string
	ActiveCount int
	DoneCount   int
}

type Entry struct {
	ID          int64
	DiaryID     int64
	Date        string // YYYY-MM-DD
	Title       string
	Description string
	Done        bool
}

type NoteRef struct {
	ID      int64
	Title   string
	Snippet string
}

type Note struct {
	ID    int64
	Title string
	Text  string
}

// Intent — распознанная голосовая команда.
type Intent struct {
	Kind  string // task_create, diary_add, note_read, … , unknown
	Title string // основной аргумент: название задачи/заметки, текст записи
	Text  string // вторичный текст: тело заметки/дописка
	Date  string // YYYY-MM-DD, если во фразе была дата
}

// IntentParser — ИИ-разбор фразы (aisvc.Chat ключом компании): максимальная
// точность и гибкость. Ошибка/таймаут/AI_DISABLED — фолбэк на классический
// регэксп-парсер (fail-open, как поиск задач).
type IntentParser interface {
	ParseIntent(ctx context.Context, companyID int64, utterance string, now time.Time) (*Intent, error)
}

// ── Порты gRPC-клиентов (tasksvc/diarysvc/notesvc); в тестах — фейки ──

type TasksClient interface {
	SearchTasks(ctx context.Context, companyID int64, query string, limit int) ([]TaskRef, error)
	CreateTask(ctx context.Context, companyID, userID int64, name string, departmentID int64) (*TaskRef, error)
	CloseTask(ctx context.Context, companyID, userID, taskID int64) (string, error)
	ListOpenTasks(ctx context.Context, companyID, userID int64, limit int) ([]TaskRef, int, error)
	ListDepartments(ctx context.Context, companyID int64) ([]CatalogItem, error)
	ListUnitTypes(ctx context.Context, companyID int64) ([]CatalogItem, error)
	StartUnit(ctx context.Context, companyID, userID, taskID, unitTypeID int64) (string, error)
	StopActiveUnit(ctx context.Context, userID int64) (*StoppedUnit, error)
	GetActiveUnit(ctx context.Context, userID int64) (*ActiveUnit, error)
}

type DiaryClient interface {
	ListDiaries(ctx context.Context, userID int64) ([]Diary, error)
	CreateDiary(ctx context.Context, userID int64, name string) (*Diary, error)
	ListEntries(ctx context.Context, userID, diaryID int64, from, to string) ([]Entry, error)
	CreateEntry(ctx context.Context, userID, diaryID int64, date, title string) (*Entry, error)
	SetEntryDone(ctx context.Context, userID, diaryID, entryID int64, done bool) (*Entry, error)
	MoveEntry(ctx context.Context, userID, diaryID, entryID int64, date string) (*Entry, error)
	DeleteEntry(ctx context.Context, userID, diaryID, entryID int64) error
}

type NotesClient interface {
	CreateFolder(ctx context.Context, userID int64, name string) error
	CreateNote(ctx context.Context, userID, companyID int64, title, text string) (*NoteRef, error)
	FindNotes(ctx context.Context, userID, companyID int64, query string, limit int) ([]NoteRef, error)
	GetNote(ctx context.Context, userID, noteID int64) (*Note, error)
	AppendNote(ctx context.Context, userID, companyID, noteID int64, text string) (*NoteRef, error)
	DeleteNote(ctx context.Context, userID, noteID int64) error
}
