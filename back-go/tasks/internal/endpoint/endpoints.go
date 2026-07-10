// Package endpoint — go-kit обёртки use-case'ов: единая сигнатура
// (ctx, request) → (response, error). Та же схема, что в остальных сервисах.
package endpoint

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/service"
)

type Endpoints struct {
	ListTasks      endpoint.Endpoint
	CreateTask     endpoint.Endpoint
	GetTask        endpoint.Endpoint
	UpdateTask     endpoint.Endpoint
	DeleteTask     endpoint.Endpoint
	ArchiveTask    endpoint.Endpoint
	RestoreTask    endpoint.Endpoint
	SetTaskColor   endpoint.Endpoint
	ToggleFavorite endpoint.Endpoint
	SetResponsible endpoint.Endpoint
	SetStage       endpoint.Endpoint
	Contributors   endpoint.Endpoint

	TaskUnits  endpoint.Endpoint
	CreateUnit endpoint.Endpoint
	ActiveUnit endpoint.Endpoint
	UpdateUnit endpoint.Endpoint
	StopUnit   endpoint.Endpoint
	DeleteUnit endpoint.Endpoint

	ListComments  endpoint.Endpoint
	CreateComment endpoint.Endpoint
	UpdateComment endpoint.Endpoint
	DeleteComment endpoint.Endpoint

	ListUnitTypes  endpoint.Endpoint
	CreateUnitType endpoint.Endpoint
	UpdateUnitType endpoint.Endpoint
	DeleteUnitType endpoint.Endpoint

	ListDepartments  endpoint.Endpoint
	CreateDepartment endpoint.Endpoint
	UpdateDepartment endpoint.Endpoint
	DeleteDepartment endpoint.Endpoint

	ListTags    endpoint.Endpoint
	CreateTag   endpoint.Endpoint
	UpdateTag   endpoint.Endpoint
	DeleteTag   endpoint.Endpoint
	SetTaskTags endpoint.Endpoint

	ListStages    endpoint.Endpoint
	CreateStage   endpoint.Endpoint
	UpdateStage   endpoint.Endpoint
	DeleteStage   endpoint.Endpoint
	ReorderStages endpoint.Endpoint

	StatsCommon        endpoint.Endpoint
	StatsExtended      endpoint.Endpoint
	ExportCommonXLSX   endpoint.Endpoint
	ExportExtendedXLSX endpoint.Endpoint
	StatsUserTasks     endpoint.Endpoint
	StatsProfile       endpoint.Endpoint
	StatsEmployees     endpoint.Endpoint
	StatsResponsibles  endpoint.Endpoint

	// Инструменты ИИ-ассистента (gRPC TasksService, зовёт aisvc).
	AssistantStatsSummary endpoint.Endpoint
	AssistantDepartments  endpoint.Endpoint
	AssistantTopEmployees endpoint.Endpoint
	AssistantByUnitTypes  endpoint.Endpoint
	AssistantCalendar     endpoint.Endpoint
	AssistantSearchTasks  endpoint.Endpoint
	AssistantTaskLink     endpoint.Endpoint

	YougileStatus          endpoint.Endpoint
	YougileConnect         endpoint.Endpoint
	YougileDisconnect      endpoint.Endpoint
	YougileRotate          endpoint.Endpoint
	YougileLookupCompanies endpoint.Endpoint
	YougileProjects        endpoint.Endpoint
	YougileBoards          endpoint.Endpoint
	YougileColumns         endpoint.Endpoint
	YougileGetSettings     endpoint.Endpoint
	YougileUpdateSettings  endpoint.Endpoint
	YougileReset           endpoint.Endpoint
	YougileImport          endpoint.Endpoint
	YougileExport          endpoint.Endpoint
	YougileUnlink          endpoint.Endpoint
	YougileWebhook         endpoint.Endpoint
	YougileRegisterWebhook endpoint.Endpoint
}

// ── Транспорт-независимые запросы ────────────────────────────────

type CreateTaskRequest struct {
	ActorID   int64
	CompanyID int64
	Body      dto.TaskCreate
}

// CompanyID во всех by-id запросах — активная компания актора из токена:
// сервис отвечает 404 на задачи/юниты чужих компаний (multi-tenancy).

type UpdateTaskRequest struct {
	TaskID    int64
	ActorID   int64
	CompanyID *int64
	Body      dto.TaskUpdate
}

type TaskActorRequest struct {
	TaskID    int64
	ActorID   int64
	CompanyID *int64
}

type TaskColorRequest struct {
	TaskID    int64
	UserID    int64
	CompanyID *int64
	Color     *string
}

type SetResponsibleRequest struct {
	TaskID            int64
	ActorID           int64
	CompanyID         *int64
	ResponsibleUserID *int64
}

type SetStageRequest struct {
	TaskID    int64
	ActorID   int64
	CompanyID *int64
	StageID   *int64
}

// TagCreateRequest / TagUpdateRequest — CRUD справочника тегов компании.
type TagCreateRequest struct {
	CompanyID int64
	Name      string
	Color     string
}

type TagUpdateRequest struct {
	CompanyID int64
	TagID     int64
	Name      *string
	Color     *string
}

// SetTaskTagsRequest — полная замена набора тегов задачи.
type SetTaskTagsRequest struct {
	TaskID    int64
	ActorID   int64
	CompanyID *int64
	TagIDs    []int64
}

type CreateUnitRequest struct {
	TaskID     int64
	UserID     int64
	CompanyID  *int64
	Name       string
	UnitTypeID int64
}

type UnitActorRequest struct {
	UnitID     int64
	ActorID    int64
	ActorLevel int
	CompanyID  *int64
}

type UpdateUnitRequest struct {
	UnitID     int64
	ActorID    int64
	ActorLevel int
	CompanyID  *int64
	Body       dto.UnitUpdate
}

type CommentCreateRequest struct {
	TaskID    int64
	AuthorID  int64
	CompanyID *int64
	Text      string
}

type CommentEditRequest struct {
	TaskID     int64
	CommentID  int64
	UserID     int64
	ActorLevel int
	CompanyID  *int64
	Text       string
}

type CompanyNameRequest struct {
	CompanyID int64
	ItemID    int64
	Name      string
}

type StageCreateRequest struct {
	CompanyID int64
	Name      string
	Color     string
}

type StageUpdateRequest struct {
	CompanyID int64
	StageID   int64
	Name      *string
	Color     *string
}

type CompanyItemRequest struct {
	CompanyID int64
	ItemID    int64
}

type ReorderRequest struct {
	CompanyID int64
	IDs       []int64
}

type PeriodRequest struct {
	Start     time.Time
	End       time.Time
	CompanyID *int64
}

type UserTasksRequest struct {
	Actor        *domain.User
	TargetUserID int64
	Start        time.Time
	End          time.Time
}

type ProfileRequest struct {
	UserID int64
	Start  time.Time
	End    time.Time
}

// ── ИИ-ассистент (gRPC TasksService) ──────────────────────────────

// AssistantPeriodRequest — период задаётся человекочитаемым кодом
// (today/this_week/…), см. service.ResolveAssistantPeriod.
type AssistantPeriodRequest struct {
	CompanyID int64
	Period    string
}

type AssistantTopEmployeesRequest struct {
	CompanyID int64
	Period    string
	Limit     int
}

type AssistantSearchRequest struct {
	CompanyID int64
	Query     string
	Limit     int
}

type AssistantTaskLinkRequest struct {
	CompanyID int64
	TaskID    int64
}

// AssistantStatsSummaryResult / AssistantListResult — ответы эндпоинтов
// статистики ассистента: срез данных + человекочитаемый ярлык периода
// (транспорт gRPC кладёт его прямо в response.PeriodLabel).
type AssistantListResult[T any] struct {
	Items       T
	PeriodLabel string
}

// ── YouGile ──────────────────────────────────────────────────────

type YougileConnectRequest struct {
	User     *domain.User
	Login    string
	Password string
	Explicit *string // yg_company_id из payload (учитывается для DIRECTOR+)
}

type YougileRotateRequest struct {
	User     *domain.User
	Password string
}

type YougileCredsRequest struct {
	Login    string
	Password string
}

type YougileCatalogRequest struct {
	ActorID int64
	Param   string // projectId / boardId
}

type YougileSettingsUpdateRequest struct {
	Actor     *domain.User
	CompanyID int64
	Body      dto.YougileSettingsUpdate
}

type YougileCompanyActorRequest struct {
	Actor     *domain.User
	CompanyID int64
}

type YougileImportRequest struct {
	User   *domain.User
	Body   dto.YougileImport
	Origin string
}

type YougileTaskActionRequest struct {
	User   *domain.User
	TaskID int64
	Origin string
}

type YougileWebhookRequest struct {
	CompanyID int64
	Secret    string
	Body      []byte
}

type YougileWebhookResponse struct {
	Results []map[string]any
	Found   bool
}

func New(svc *service.Service, yg *service.Yougile) Endpoints {
	return Endpoints{
		ListTasks: func(ctx context.Context, request any) (any, error) {
			return svc.ListTasks(ctx, request.(domain.TaskListFilter))
		},
		CreateTask: func(ctx context.Context, request any) (any, error) {
			req := request.(CreateTaskRequest)
			return svc.CreateTask(ctx, req.ActorID, req.CompanyID, req.Body)
		},
		GetTask: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskActorRequest)
			return svc.GetTaskInCompany(ctx, req.TaskID, req.ActorID, req.CompanyID)
		},
		UpdateTask: func(ctx context.Context, request any) (any, error) {
			req := request.(UpdateTaskRequest)
			return svc.UpdateTask(ctx, req.TaskID, req.ActorID, req.CompanyID, req.Body)
		},
		DeleteTask: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskActorRequest)
			return nil, svc.DeleteTask(ctx, req.TaskID, req.CompanyID)
		},
		ArchiveTask: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskActorRequest)
			return svc.ArchiveTask(ctx, req.TaskID, req.ActorID, req.CompanyID)
		},
		RestoreTask: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskActorRequest)
			return svc.RestoreTask(ctx, req.TaskID, req.ActorID, req.CompanyID)
		},
		SetTaskColor: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskColorRequest)
			return nil, svc.SetTaskColor(ctx, req.TaskID, req.UserID, req.CompanyID, req.Color)
		},
		ToggleFavorite: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskActorRequest)
			return svc.ToggleFavorite(ctx, req.TaskID, req.ActorID, req.CompanyID)
		},
		SetResponsible: func(ctx context.Context, request any) (any, error) {
			req := request.(SetResponsibleRequest)
			return svc.SetResponsible(ctx, req.TaskID, req.ActorID, req.CompanyID, req.ResponsibleUserID)
		},
		SetStage: func(ctx context.Context, request any) (any, error) {
			req := request.(SetStageRequest)
			return svc.SetStage(ctx, req.TaskID, req.ActorID, req.CompanyID, req.StageID)
		},
		Contributors: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskActorRequest)
			return svc.Contributors(ctx, req.TaskID, req.CompanyID)
		},

		TaskUnits: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskActorRequest)
			return svc.TaskUnits(ctx, req.TaskID, req.CompanyID)
		},
		CreateUnit: func(ctx context.Context, request any) (any, error) {
			req := request.(CreateUnitRequest)
			return svc.CreateUnit(ctx, req.TaskID, req.UserID, req.CompanyID, req.Name, req.UnitTypeID)
		},
		ActiveUnit: func(ctx context.Context, request any) (any, error) {
			return svc.ActiveUnit(ctx, request.(int64))
		},
		UpdateUnit: func(ctx context.Context, request any) (any, error) {
			req := request.(UpdateUnitRequest)
			return svc.UpdateUnit(ctx, req.UnitID, req.ActorID, req.ActorLevel, req.CompanyID, req.Body)
		},
		StopUnit: func(ctx context.Context, request any) (any, error) {
			req := request.(UnitActorRequest)
			return svc.StopUnit(ctx, req.UnitID, req.ActorID, req.ActorLevel, req.CompanyID)
		},
		DeleteUnit: func(ctx context.Context, request any) (any, error) {
			req := request.(UnitActorRequest)
			return nil, svc.DeleteUnit(ctx, req.UnitID, req.ActorID, req.ActorLevel, req.CompanyID)
		},

		ListComments: func(ctx context.Context, request any) (any, error) {
			req := request.(TaskActorRequest)
			return svc.ListComments(ctx, req.TaskID, req.CompanyID)
		},
		CreateComment: func(ctx context.Context, request any) (any, error) {
			req := request.(CommentCreateRequest)
			return svc.CreateComment(ctx, req.TaskID, req.AuthorID, req.CompanyID, req.Text)
		},
		UpdateComment: func(ctx context.Context, request any) (any, error) {
			req := request.(CommentEditRequest)
			return svc.UpdateComment(ctx, req.CommentID, req.UserID, req.ActorLevel, req.CompanyID, req.Text)
		},
		DeleteComment: func(ctx context.Context, request any) (any, error) {
			req := request.(CommentEditRequest)
			return nil, svc.DeleteComment(ctx, req.TaskID, req.CommentID, req.UserID, req.ActorLevel, req.CompanyID)
		},

		ListUnitTypes: func(ctx context.Context, request any) (any, error) {
			return svc.ListUnitTypes(ctx, request.(int64))
		},
		CreateUnitType: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyNameRequest)
			return svc.CreateUnitType(ctx, req.CompanyID, req.Name)
		},
		UpdateUnitType: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyNameRequest)
			return svc.UpdateUnitType(ctx, req.CompanyID, req.ItemID, req.Name)
		},
		DeleteUnitType: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyItemRequest)
			return nil, svc.DeleteUnitType(ctx, req.CompanyID, req.ItemID)
		},

		ListDepartments: func(ctx context.Context, request any) (any, error) {
			return svc.ListDepartments(ctx, request.(int64))
		},
		CreateDepartment: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyNameRequest)
			return svc.CreateDepartment(ctx, req.CompanyID, req.Name)
		},
		UpdateDepartment: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyNameRequest)
			return svc.UpdateDepartment(ctx, req.CompanyID, req.ItemID, req.Name)
		},
		DeleteDepartment: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyItemRequest)
			return nil, svc.DeleteDepartment(ctx, req.CompanyID, req.ItemID)
		},

		ListTags: func(ctx context.Context, request any) (any, error) {
			return svc.ListTags(ctx, request.(int64))
		},
		CreateTag: func(ctx context.Context, request any) (any, error) {
			req := request.(TagCreateRequest)
			return svc.CreateTag(ctx, req.CompanyID, req.Name, req.Color)
		},
		UpdateTag: func(ctx context.Context, request any) (any, error) {
			req := request.(TagUpdateRequest)
			return svc.UpdateTag(ctx, req.CompanyID, req.TagID, req.Name, req.Color)
		},
		DeleteTag: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyItemRequest)
			return nil, svc.DeleteTag(ctx, req.CompanyID, req.ItemID)
		},
		SetTaskTags: func(ctx context.Context, request any) (any, error) {
			req := request.(SetTaskTagsRequest)
			return svc.SetTaskTags(ctx, req.TaskID, req.ActorID, req.CompanyID, req.TagIDs)
		},

		ListStages: func(ctx context.Context, request any) (any, error) {
			return svc.ListStages(ctx, request.(int64))
		},
		CreateStage: func(ctx context.Context, request any) (any, error) {
			req := request.(StageCreateRequest)
			return svc.CreateStage(ctx, req.CompanyID, req.Name, req.Color)
		},
		UpdateStage: func(ctx context.Context, request any) (any, error) {
			req := request.(StageUpdateRequest)
			return svc.UpdateStage(ctx, req.CompanyID, req.StageID, req.Name, req.Color)
		},
		DeleteStage: func(ctx context.Context, request any) (any, error) {
			req := request.(CompanyItemRequest)
			return nil, svc.DeleteStage(ctx, req.CompanyID, req.ItemID)
		},
		ReorderStages: func(ctx context.Context, request any) (any, error) {
			req := request.(ReorderRequest)
			return svc.ReorderStages(ctx, req.CompanyID, req.IDs)
		},

		StatsCommon: func(ctx context.Context, request any) (any, error) {
			req := request.(PeriodRequest)
			return svc.StatsCommon(ctx, req.Start, req.End, req.CompanyID)
		},
		StatsExtended: func(ctx context.Context, request any) (any, error) {
			req := request.(PeriodRequest)
			return svc.StatsExtended(ctx, req.Start, req.End, req.CompanyID)
		},
		ExportCommonXLSX: func(ctx context.Context, request any) (any, error) {
			req := request.(PeriodRequest)
			return svc.ExportCommonXLSX(ctx, req.Start, req.End, req.CompanyID)
		},
		ExportExtendedXLSX: func(ctx context.Context, request any) (any, error) {
			req := request.(PeriodRequest)
			return svc.ExportExtendedXLSX(ctx, req.Start, req.End, req.CompanyID)
		},
		StatsUserTasks: func(ctx context.Context, request any) (any, error) {
			req := request.(UserTasksRequest)
			return svc.StatsUserTasks(ctx, req.Actor, req.TargetUserID, req.Start, req.End)
		},
		StatsProfile: func(ctx context.Context, request any) (any, error) {
			req := request.(ProfileRequest)
			return svc.StatsProfile(ctx, req.UserID, req.Start, req.End)
		},
		StatsEmployees: func(ctx context.Context, request any) (any, error) {
			return svc.StatsEmployees(ctx, request.(*int64))
		},
		StatsResponsibles: func(ctx context.Context, request any) (any, error) {
			return svc.StatsResponsibles(ctx, request.(*int64))
		},

		AssistantStatsSummary: func(ctx context.Context, request any) (any, error) {
			req := request.(AssistantPeriodRequest)
			return svc.AssistantStatsSummary(ctx, req.CompanyID, req.Period)
		},
		AssistantDepartments: func(ctx context.Context, request any) (any, error) {
			req := request.(AssistantPeriodRequest)
			items, label, err := svc.AssistantDepartments(ctx, req.CompanyID, req.Period)
			if err != nil {
				return nil, err
			}
			return AssistantListResult[[]dto.DeptStats]{Items: items, PeriodLabel: label}, nil
		},
		AssistantTopEmployees: func(ctx context.Context, request any) (any, error) {
			req := request.(AssistantTopEmployeesRequest)
			items, label, err := svc.AssistantTopEmployees(ctx, req.CompanyID, req.Period, req.Limit)
			if err != nil {
				return nil, err
			}
			return AssistantListResult[[]dto.TaskByEmployee]{Items: items, PeriodLabel: label}, nil
		},
		AssistantByUnitTypes: func(ctx context.Context, request any) (any, error) {
			req := request.(AssistantPeriodRequest)
			items, label, err := svc.AssistantByUnitTypes(ctx, req.CompanyID, req.Period)
			if err != nil {
				return nil, err
			}
			return AssistantListResult[[]dto.UnitTypeStats]{Items: items, PeriodLabel: label}, nil
		},
		AssistantCalendar: func(ctx context.Context, request any) (any, error) {
			req := request.(AssistantPeriodRequest)
			items, label, err := svc.AssistantCalendar(ctx, req.CompanyID, req.Period)
			if err != nil {
				return nil, err
			}
			return AssistantListResult[[]dto.CalendarDay]{Items: items, PeriodLabel: label}, nil
		},
		AssistantSearchTasks: func(ctx context.Context, request any) (any, error) {
			req := request.(AssistantSearchRequest)
			return svc.AssistantSearchTasks(ctx, req.CompanyID, req.Query, req.Limit)
		},
		AssistantTaskLink: func(ctx context.Context, request any) (any, error) {
			req := request.(AssistantTaskLinkRequest)
			return svc.AssistantTaskLink(ctx, req.CompanyID, req.TaskID)
		},

		YougileStatus: func(ctx context.Context, request any) (any, error) {
			return yg.Status(ctx, request.(*domain.User))
		},
		YougileConnect: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileConnectRequest)
			return yg.Connect(ctx, req.User, req.Login, req.Password, req.Explicit)
		},
		YougileDisconnect: func(ctx context.Context, request any) (any, error) {
			return nil, yg.Disconnect(ctx, request.(int64))
		},
		YougileRotate: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileRotateRequest)
			return yg.Rotate(ctx, req.User, req.Password)
		},
		YougileLookupCompanies: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileCredsRequest)
			return yg.LookupCompanies(ctx, req.Login, req.Password)
		},
		YougileProjects: func(ctx context.Context, request any) (any, error) {
			return yg.Projects(ctx, request.(int64))
		},
		YougileBoards: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileCatalogRequest)
			return yg.Boards(ctx, req.ActorID, req.Param)
		},
		YougileColumns: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileCatalogRequest)
			return yg.Columns(ctx, req.ActorID, req.Param)
		},
		YougileGetSettings: func(ctx context.Context, request any) (any, error) {
			return yg.CompanySettings(ctx, request.(*int64))
		},
		YougileUpdateSettings: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileSettingsUpdateRequest)
			return yg.UpdateCompanySettings(ctx, req.Actor, req.CompanyID, req.Body)
		},
		YougileReset: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileCompanyActorRequest)
			return yg.ResetIntegration(ctx, req.Actor, req.CompanyID)
		},
		YougileImport: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileImportRequest)
			return yg.ImportTask(ctx, req.User, req.Body, req.Origin)
		},
		YougileExport: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileTaskActionRequest)
			return yg.ExportTask(ctx, req.User, req.TaskID, req.Origin)
		},
		YougileUnlink: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileTaskActionRequest)
			return yg.UnlinkTask(ctx, req.User, req.TaskID)
		},
		YougileWebhook: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileWebhookRequest)
			results, found, err := yg.HandleWebhook(ctx, req.CompanyID, req.Secret, req.Body)
			if err != nil {
				return nil, err
			}
			return YougileWebhookResponse{Results: results, Found: found}, nil
		},
		YougileRegisterWebhook: func(ctx context.Context, request any) (any, error) {
			req := request.(YougileCompanyActorRequest)
			return yg.RegisterWebhook(ctx, req.Actor, req.CompanyID)
		},
	}
}
