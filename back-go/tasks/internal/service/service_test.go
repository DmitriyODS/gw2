package service

import (
	"context"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// ── Фейки портов (без БД/Redis/gRPC, как в остальных сервисах) ───

type fakeStore struct {
	seq       int64
	tasks     map[int64]*domain.Task
	units     map[int64]*domain.Unit
	unitTypes map[int64]*domain.UnitType
	depts     map[int64]*domain.Department
	stages    map[int64]*domain.Stage
	comments  map[int64]*domain.Comment
	favorites map[[2]int64]bool
	colors    map[[2]int64]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		tasks: map[int64]*domain.Task{}, units: map[int64]*domain.Unit{},
		unitTypes: map[int64]*domain.UnitType{}, depts: map[int64]*domain.Department{},
		stages: map[int64]*domain.Stage{}, comments: map[int64]*domain.Comment{},
		favorites: map[[2]int64]bool{}, colors: map[[2]int64]string{},
	}
}

func (f *fakeStore) next() int64 { f.seq++; return f.seq }

// — TaskRepository —

func (f *fakeStore) GetTask(_ context.Context, id int64) (*domain.Task, error) {
	return f.tasks[id], nil
}

func (f *fakeStore) ListTasks(_ context.Context, fl domain.TaskListFilter) ([]*domain.Task, int, error) {
	if fl.OrderedSet && len(fl.OrderedIDs) == 0 {
		return []*domain.Task{}, 0, nil
	}
	out := []*domain.Task{}
	ids := make([]int64, 0, len(f.tasks))
	for id := range f.tasks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		out = append(out, f.tasks[id])
	}
	return out, len(out), nil
}

func (f *fakeStore) CreateTask(_ context.Context, t *domain.Task) error {
	t.ID = f.next()
	t.CreatedAt = time.Now().UTC()
	if t.ReceivedAt.IsZero() {
		t.ReceivedAt = t.CreatedAt
	}
	t.Author = &domain.UserRef{ID: t.AuthorID, FIO: "Автор"}
	t.Department = &domain.DeptRef{ID: t.DepartmentID, Name: "Отдел"}
	f.tasks[t.ID] = t
	return nil
}

// updOptStr — значение текстовой колонки из fields (string / *string / nil).
func updOptStr(v any) *string {
	switch s := v.(type) {
	case string:
		return &s
	case *string:
		return s
	}
	return nil
}

func (f *fakeStore) UpdateTaskFields(_ context.Context, id int64, fields map[string]any) error {
	t := f.tasks[id]
	for k, v := range fields {
		switch k {
		case "name":
			t.Name = v.(string)
		case "is_archived":
			t.IsArchived = v.(bool)
		case "archived_at":
			if v == nil {
				t.ArchivedAt = nil
			} else {
				at := v.(time.Time)
				t.ArchivedAt = &at
			}
		case "responsible_user_id":
			t.ResponsibleUserID, _ = v.(*int64)
		case "stage_id":
			t.StageID, _ = v.(*int64)
		case "deadline":
			if v == nil {
				t.Deadline = nil
			} else {
				dl := v.(time.Time)
				t.Deadline = &dl
			}
		case "link_yougile":
			t.LinkYougile = updOptStr(v)
		case "yougile_task_id":
			t.YougileTaskID = updOptStr(v)
		case "yougile_id_short":
			t.YougileIDShort = updOptStr(v)
		case "yougile_project_id":
			t.YougileProjectID = updOptStr(v)
		case "yougile_board_id":
			t.YougileBoardID = updOptStr(v)
		case "yougile_column_id":
			t.YougileColumnID = updOptStr(v)
		case "yougile_sync_hash":
			t.YougileSyncHash = updOptStr(v)
		}
	}
	return nil
}

func (f *fakeStore) DeleteTask(_ context.Context, id int64) error {
	delete(f.tasks, id)
	return nil
}

func (f *fakeStore) HasActiveUnit(_ context.Context, taskID int64) (bool, error) {
	for _, u := range f.units {
		if u.TaskID == taskID && u.DatetimeEnd == nil {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) HasAnyUnits(_ context.Context, taskID int64) (bool, error) {
	for _, u := range f.units {
		if u.TaskID == taskID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) IsFavorite(_ context.Context, taskID, userID int64) (bool, error) {
	return f.favorites[[2]int64{taskID, userID}], nil
}

func (f *fakeStore) ToggleFavorite(_ context.Context, taskID, userID int64) (bool, error) {
	key := [2]int64{taskID, userID}
	if f.favorites[key] {
		delete(f.favorites, key)
		return false, nil
	}
	f.favorites[key] = true
	return true, nil
}

func (f *fakeStore) ActiveUsers(_ context.Context, _ int64) ([]domain.UserRef, error) {
	return nil, nil
}

func (f *fakeStore) Contributors(_ context.Context, _ int64) ([]domain.UserRef, error) {
	return nil, nil
}

func (f *fakeStore) UserColor(_ context.Context, taskID, userID int64) (*string, error) {
	if c, ok := f.colors[[2]int64{taskID, userID}]; ok {
		return &c, nil
	}
	return nil, nil
}

func (f *fakeStore) SetUserColor(_ context.Context, taskID, userID int64, color *string) error {
	key := [2]int64{taskID, userID}
	if color == nil {
		delete(f.colors, key)
	} else {
		f.colors[key] = *color
	}
	return nil
}

func (f *fakeStore) Enrichment(_ context.Context, _ []int64, _ int64) (*domain.TaskEnrichment, error) {
	return &domain.TaskEnrichment{
		ActiveUsers: map[int64][]domain.UserRef{}, UserColors: map[int64]string{},
		FavoriteIDs: map[int64]bool{}, WithUnits: map[int64]bool{},
	}, nil
}

// — UnitRepository —

func (f *fakeStore) GetUnit(_ context.Context, id int64) (*domain.Unit, error) {
	return f.units[id], nil
}

func (f *fakeStore) UnitsByTask(_ context.Context, taskID int64) ([]*domain.Unit, error) {
	out := []*domain.Unit{}
	for _, u := range f.units {
		if u.TaskID == taskID {
			out = append(out, u)
		}
	}
	return out, nil
}

func (f *fakeStore) ActiveUnitForUser(_ context.Context, userID int64) (*domain.Unit, error) {
	for _, u := range f.units {
		if u.UserID == userID && u.DatetimeEnd == nil {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeStore) CreateUnit(_ context.Context, u *domain.Unit) error {
	u.ID = f.next()
	u.DatetimeStart = time.Now().UTC()
	u.CreatedAt = u.DatetimeStart
	u.User = &domain.UserRef{ID: u.UserID, FIO: "Сотрудник"}
	u.UnitType = &domain.UnitTypeRef{ID: u.UnitTypeID, Name: "Тип"}
	f.units[u.ID] = u
	return nil
}

func (f *fakeStore) UpdateUnitFields(_ context.Context, id int64, fields map[string]any) error {
	u := f.units[id]
	u.IsEdited = true
	if v, ok := fields["name"]; ok {
		u.Name = v.(string)
	}
	return nil
}

func (f *fakeStore) StopUnit(_ context.Context, id int64) (time.Time, error) {
	end := time.Now().UTC()
	f.units[id].DatetimeEnd = &end
	return end, nil
}

func (f *fakeStore) DeleteUnit(_ context.Context, id int64) error {
	delete(f.units, id)
	return nil
}

// — UnitTypeRepository (минимум для тестов) —

func (f *fakeStore) ListUnitTypes(_ context.Context, _ int64) ([]*domain.UnitType, error) {
	return nil, nil
}
func (f *fakeStore) GetUnitType(_ context.Context, id int64) (*domain.UnitType, error) {
	return f.unitTypes[id], nil
}
func (f *fakeStore) GetUnitTypeByName(_ context.Context, _ string, _ int64) (*domain.UnitType, error) {
	return nil, nil
}
func (f *fakeStore) CreateUnitType(_ context.Context, ut *domain.UnitType) error {
	ut.ID = f.next()
	f.unitTypes[ut.ID] = ut
	return nil
}
func (f *fakeStore) UpdateUnitTypeName(_ context.Context, id int64, name string) error {
	f.unitTypes[id].Name = name
	return nil
}
func (f *fakeStore) DeleteUnitType(_ context.Context, id int64) error {
	delete(f.unitTypes, id)
	// Каскад юнитов — как FK ON DELETE CASCADE.
	for uid, u := range f.units {
		if u.UnitTypeID == id {
			delete(f.units, uid)
		}
	}
	return nil
}

// — DepartmentRepository —

func (f *fakeStore) ListDepartments(_ context.Context, _ int64) ([]*domain.Department, error) {
	return nil, nil
}
func (f *fakeStore) GetDepartment(_ context.Context, id int64) (*domain.Department, error) {
	return f.depts[id], nil
}
func (f *fakeStore) GetDepartmentByName(_ context.Context, _ string, _ int64) (*domain.Department, error) {
	return nil, nil
}
func (f *fakeStore) CreateDepartment(_ context.Context, d *domain.Department) error {
	d.ID = f.next()
	f.depts[d.ID] = d
	return nil
}
func (f *fakeStore) UpdateDepartmentName(_ context.Context, id int64, name string) error {
	f.depts[id].Name = name
	return nil
}
func (f *fakeStore) DeleteDepartment(_ context.Context, id int64) error {
	delete(f.depts, id)
	return nil
}

// — StageRepository —

func (f *fakeStore) ListStages(_ context.Context, _ int64) ([]*domain.Stage, error) { return nil, nil }
func (f *fakeStore) GetStage(_ context.Context, id int64) (*domain.Stage, error) {
	return f.stages[id], nil
}
func (f *fakeStore) GetStageByName(_ context.Context, _ string, _ int64) (*domain.Stage, error) {
	return nil, nil
}
func (f *fakeStore) NextStageOrder(_ context.Context, _ int64) (int, error) { return 1, nil }
func (f *fakeStore) CreateStage(_ context.Context, s *domain.Stage) error {
	s.ID = f.next()
	f.stages[s.ID] = s
	return nil
}
func (f *fakeStore) UpdateStageFields(_ context.Context, _ int64, _ map[string]any) error {
	return nil
}
func (f *fakeStore) DeleteStage(_ context.Context, id int64) error {
	delete(f.stages, id)
	return nil
}
func (f *fakeStore) ReorderStages(_ context.Context, _ int64, _ []int64) error { return nil }

// — CommentRepository —

func (f *fakeStore) GetComment(_ context.Context, id int64) (*domain.Comment, error) {
	return f.comments[id], nil
}
func (f *fakeStore) ListComments(_ context.Context, _ int64) ([]*domain.Comment, error) {
	return nil, nil
}
func (f *fakeStore) CreateComment(_ context.Context, c *domain.Comment) error {
	c.ID = f.next()
	c.CreatedAt = time.Now().UTC()
	c.Author = &domain.UserRef{ID: c.AuthorID, FIO: "Автор"}
	f.comments[c.ID] = c
	return nil
}
func (f *fakeStore) UpdateCommentText(_ context.Context, id int64, text string, at time.Time) error {
	f.comments[id].Text, f.comments[id].UpdatedAt = text, &at
	return nil
}
func (f *fakeStore) SoftDeleteComment(_ context.Context, id int64, at time.Time) error {
	f.comments[id].DeletedAt = &at
	return nil
}

// — Остальные порты —

type fakeUsers struct{ users map[int64]*domain.User }

func (f *fakeUsers) GetUser(_ context.Context, id int64) (*domain.User, error) {
	return f.users[id], nil
}

func (f *fakeUsers) YougileEnabled(_ context.Context, _ int64) (bool, error) { return true, nil }

type fakeGroove struct {
	started, stopped int
	closedHero       int64
}

func (f *fakeGroove) OnUnitStarted(*domain.Unit, string) { f.started++ }
func (f *fakeGroove) OnUnitStopped(*domain.Unit, string) { f.stopped++ }
func (f *fakeGroove) OnTaskClosed(_ *domain.Task, actorID int64) {
	f.closedHero = actorID
}

type fakeAI struct {
	enabled   bool
	hits      []int64
	reindexed []int64
}

func (f *fakeAI) Enabled(context.Context, int64) bool { return f.enabled }
func (f *fakeAI) SemanticSearch(context.Context, int64, string) []int64 {
	return f.hits
}
func (f *fakeAI) ScheduleReindex(taskID int64) { f.reindexed = append(f.reindexed, taskID) }

type busEvent struct {
	Event   string
	Rooms   []string
	Payload any
}

type fakeBus struct{ events []busEvent }

func (f *fakeBus) Publish(_ context.Context, event string, rooms []string, payload any) {
	f.events = append(f.events, busEvent{Event: event, Rooms: rooms, Payload: payload})
}

func (f *fakeBus) names() []string {
	out := make([]string, 0, len(f.events))
	for _, e := range f.events {
		out = append(out, e.Event)
	}
	return out
}

// ── Хелперы ──────────────────────────────────────────────────────

func newTestService() (*Service, *fakeStore, *fakeGroove, *fakeAI, *fakeBus, *fakeUsers) {
	store := newFakeStore()
	groove := &fakeGroove{}
	ai := &fakeAI{}
	bus := &fakeBus{}
	users := &fakeUsers{users: map[int64]*domain.User{}}
	svc := New(Deps{
		Tasks: store, Units: store, UnitTypes: store, Depts: store, Stages: store,
		Comments: store, Stats: nil, Users: users, Companies: users,
		Groove: groove, AI: ai, Bus: bus, Log: slog.New(slog.DiscardHandler),
	})
	return svc, store, groove, ai, bus, users
}

func seedTask(store *fakeStore, companyID int64) *domain.Task {
	task := &domain.Task{Name: "Задача", AuthorID: 1, DepartmentID: 1, CompanyID: companyID}
	_ = store.CreateTask(context.Background(), task)
	return task
}

func employee(users *fakeUsers, id, companyID int64) *domain.User {
	u := &domain.User{ID: id, FIO: "Тест", RoleLevel: domain.LevelEmployee,
		CompanyID: &companyID, CompanyActive: true}
	users.users[id] = u
	return u
}

// ── Тесты инвариантов ────────────────────────────────────────────

func TestCreateUnitSecondActiveForbidden(t *testing.T) {
	svc, store, groove, _, _, _ := newTestService()
	task := seedTask(store, 1)
	store.unitTypes[10] = &domain.UnitType{ID: 10, Name: "Код", CompanyID: 1}

	if _, err := svc.CreateUnit(context.Background(), task.ID, 5, "первый", 10); err != nil {
		t.Fatalf("первый юнит: %v", err)
	}
	if groove.started != 1 {
		t.Fatal("groove.OnUnitStarted не вызван")
	}
	_, err := svc.CreateUnit(context.Background(), task.ID, 5, "второй", 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "ACTIVE_UNIT_EXISTS" || de.HTTPStatus != 409 {
		t.Fatalf("ожидался ACTIVE_UNIT_EXISTS 409, получено %v", err)
	}
}

func TestCreateUnitForeignTypeForbidden(t *testing.T) {
	svc, store, _, _, _, _ := newTestService()
	task := seedTask(store, 1)
	store.unitTypes[10] = &domain.UnitType{ID: 10, Name: "Чужой", CompanyID: 2}

	_, err := svc.CreateUnit(context.Background(), task.ID, 5, "юнит", 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "TYPE_FOREIGN" || de.HTTPStatus != 422 {
		t.Fatalf("ожидался TYPE_FOREIGN 422, получено %v", err)
	}
}

func TestArchiveTaskWithActiveUnitForbidden(t *testing.T) {
	svc, store, groove, _, bus, _ := newTestService()
	task := seedTask(store, 1)
	store.unitTypes[10] = &domain.UnitType{ID: 10, Name: "Код", CompanyID: 1}
	if _, err := svc.CreateUnit(context.Background(), task.ID, 5, "работа", 10); err != nil {
		t.Fatalf("юнит: %v", err)
	}

	_, err := svc.ArchiveTask(context.Background(), task.ID, 5)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "HAS_ACTIVE_UNIT" || de.HTTPStatus != 422 {
		t.Fatalf("ожидался HAS_ACTIVE_UNIT 422, получено %v", err)
	}

	// Останавливаем юнит — архивирование проходит, хук и события на месте.
	for id := range store.units {
		if _, err := svc.StopUnit(context.Background(), id, 5, domain.LevelEmployee); err != nil {
			t.Fatalf("stop: %v", err)
		}
	}
	if _, err := svc.ArchiveTask(context.Background(), task.ID, 5); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if groove.closedHero != 5 {
		t.Fatalf("OnTaskClosed hero = %d", groove.closedHero)
	}
	names := bus.names()
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["task:archived"] {
		t.Fatalf("событие архивирования не опубликовано: %v", names)
	}

	_, err = svc.ArchiveTask(context.Background(), task.ID, 5)
	if de := domain.AsDomainError(err); de == nil || de.Code != "ALREADY_ARCHIVED" {
		t.Fatalf("повторный архив: %v", err)
	}
}

func TestStopForeignUnitNeedsManager(t *testing.T) {
	svc, store, _, _, bus, users := newTestService()
	task := seedTask(store, 1)
	store.unitTypes[10] = &domain.UnitType{ID: 10, Name: "Код", CompanyID: 1}
	employee(users, 7, 1)
	unit, err := svc.CreateUnit(context.Background(), task.ID, 5, "чужой", 10)
	if err != nil {
		t.Fatalf("юнит: %v", err)
	}

	_, err = svc.StopUnit(context.Background(), unit.ID, 7, domain.LevelEmployee)
	if de := domain.AsDomainError(err); de == nil || de.Code != "FORBIDDEN" {
		t.Fatalf("сотрудник остановил чужой юнит: %v", err)
	}

	if _, err := svc.StopUnit(context.Background(), unit.ID, 7, domain.LevelManager); err != nil {
		t.Fatalf("менеджер не смог остановить: %v", err)
	}
	var forced *busEvent
	for i := range bus.events {
		if bus.events[i].Event == "unit:force_stopped" {
			forced = &bus.events[i]
		}
	}
	if forced == nil || len(forced.Rooms) != 1 || forced.Rooms[0] != "user_5" {
		t.Fatalf("unit:force_stopped не ушёл владельцу: %+v", forced)
	}
}

func TestUpdateTaskReindexAndBroadcast(t *testing.T) {
	svc, store, _, ai, bus, _ := newTestService()
	task := seedTask(store, 1)

	newName := "Новое имя"
	if _, err := svc.UpdateTask(context.Background(), task.ID, 1,
		dto.TaskUpdate{Name: &newName}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(ai.reindexed) == 0 || ai.reindexed[len(ai.reindexed)-1] != task.ID {
		t.Fatal("изменение name должно перегенерить эмбеддинг")
	}
	var updated *busEvent
	for i := range bus.events {
		if bus.events[i].Event == "task:updated" {
			updated = &bus.events[i]
		}
	}
	if updated == nil {
		t.Fatal("событие task:updated не опубликовано")
	}
}

func TestSemanticSearchTakesOverList(t *testing.T) {
	svc, store, _, ai, _, _ := newTestService()
	seedTask(store, 1)
	ai.enabled = true
	ai.hits = []int64{} // пустая семантическая выдача

	companyID := int64(1)
	out, err := svc.ListTasks(context.Background(), domain.TaskListFilter{
		CurrentUserID: 1, CompanyID: &companyID, Tab: "active",
		Search: "логика авторизации", Page: 1, PerPage: 30,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// При включённом AI пустая выдача честно отдаётся пустой (без LIKE).
	if out.Total != 0 || len(out.Items) != 0 {
		t.Fatalf("ожидалась пустая семантическая выдача, получено %d", out.Total)
	}
}
