package service

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

/* Фейки портов: бизнес-логика проверяется без БД и Redis. */

type fakeRepo struct {
	schedules  map[int64]*domain.Schedule
	categories map[int64][]*domain.Category
	fields     map[int64][]*domain.Field
	items      map[int64][]*domain.Item
	userShares map[int64][]*domain.UserShare
	shares     map[string]*domain.Share
	nextID     int64
	// adjusted — сколько занятий тронула последняя подрезка недель.
	adjusted int
}

func newRepo() *fakeRepo {
	return &fakeRepo{
		schedules: map[int64]*domain.Schedule{}, categories: map[int64][]*domain.Category{},
		fields: map[int64][]*domain.Field{}, items: map[int64][]*domain.Item{},
		userShares: map[int64][]*domain.UserShare{}, shares: map[string]*domain.Share{},
	}
}

func (r *fakeRepo) id() int64 { r.nextID++; return r.nextID }

func (r *fakeRepo) ListOwned(_ domain.Ctx, ownerID int64) ([]*domain.Schedule, error) {
	out := []*domain.Schedule{}
	for _, s := range r.schedules {
		if s.OwnerID == ownerID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListShared(_ domain.Ctx, userID int64, companyIDs []int64) ([]*domain.Schedule, error) {
	out := []*domain.Schedule{}
	for id, shares := range r.userShares {
		for _, sh := range shares {
			if matchesTarget(sh, userID, companyIDs) && r.schedules[id].OwnerID != userID {
				out = append(out, r.schedules[id])
			}
		}
	}
	return out, nil
}

func matchesTarget(sh *domain.UserShare, userID int64, companyIDs []int64) bool {
	if sh.UserID != nil && *sh.UserID == userID {
		return true
	}
	if sh.CompanyID == nil {
		return false
	}
	for _, c := range companyIDs {
		if c == *sh.CompanyID {
			return true
		}
	}
	return false
}

func (r *fakeRepo) GetSchedule(_ domain.Ctx, id int64) (*domain.Schedule, error) {
	return r.schedules[id], nil
}

func (r *fakeRepo) HasAccess(_ domain.Ctx, scheduleID, userID int64, companyIDs []int64) (bool, error) {
	for _, sh := range r.userShares[scheduleID] {
		if matchesTarget(sh, userID, companyIDs) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) NextPosition(domain.Ctx, int64) (int, error) { return 1, nil }

func (r *fakeRepo) CreateSchedule(_ domain.Ctx, s *domain.Schedule) error {
	s.ID = r.id()
	s.CreatedAt, s.UpdatedAt = time.Now(), time.Now()
	r.schedules[s.ID] = s
	return nil
}

func (r *fakeRepo) UpdateSchedule(_ domain.Ctx, s *domain.Schedule) error {
	r.schedules[s.ID] = s
	return nil
}

func (r *fakeRepo) DeleteSchedule(_ domain.Ctx, id int64) error {
	delete(r.schedules, id)
	return nil
}

func (r *fakeRepo) Audience(_ domain.Ctx, scheduleID int64) ([]int64, error) {
	out := []int64{}
	if s := r.schedules[scheduleID]; s != nil {
		out = append(out, s.OwnerID)
	}
	for _, sh := range r.userShares[scheduleID] {
		if sh.UserID != nil {
			out = append(out, *sh.UserID)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListCategories(_ domain.Ctx, scheduleID int64) ([]*domain.Category, error) {
	return r.categories[scheduleID], nil
}

func (r *fakeRepo) CategoriesBySchedules(_ domain.Ctx, ids []int64) (map[int64][]*domain.Category, error) {
	out := map[int64][]*domain.Category{}
	for _, id := range ids {
		out[id] = r.categories[id]
	}
	return out, nil
}

func (r *fakeRepo) CreateCategory(_ domain.Ctx, c *domain.Category) error {
	c.ID = r.id()
	r.categories[c.ScheduleID] = append(r.categories[c.ScheduleID], c)
	return nil
}

func (r *fakeRepo) ReplaceCategories(_ domain.Ctx, scheduleID int64, cats []*domain.Category) error {
	for i, c := range cats {
		c.ID, c.ScheduleID, c.Position = r.id(), scheduleID, i+1
	}
	r.categories[scheduleID] = cats
	return nil
}

func (r *fakeRepo) UpdateCategory(_ domain.Ctx, c *domain.Category) error {
	for i, cur := range r.categories[c.ScheduleID] {
		if cur.ID == c.ID {
			r.categories[c.ScheduleID][i] = c
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func (r *fakeRepo) DeleteCategory(_ domain.Ctx, scheduleID, id int64) error {
	list := r.categories[scheduleID]
	for i, c := range list {
		if c.ID == id {
			r.categories[scheduleID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func (r *fakeRepo) NextCategoryPosition(_ domain.Ctx, scheduleID int64) (int, error) {
	return len(r.categories[scheduleID]) + 1, nil
}

func (r *fakeRepo) ListFields(_ domain.Ctx, scheduleID int64) ([]*domain.Field, error) {
	return r.fields[scheduleID], nil
}

func (r *fakeRepo) FieldsBySchedules(_ domain.Ctx, ids []int64) (map[int64][]*domain.Field, error) {
	out := map[int64][]*domain.Field{}
	for _, id := range ids {
		out[id] = r.fields[id]
	}
	return out, nil
}

func (r *fakeRepo) ReplaceFields(_ domain.Ctx, scheduleID int64, fields []*domain.Field) ([]int64, error) {
	keep := map[int64]bool{}
	for i, f := range fields {
		if f.ID == 0 {
			f.ID = r.id()
		}
		f.ScheduleID, f.Position = scheduleID, i+1
		keep[f.ID] = true
	}
	removed := []int64{}
	for _, old := range r.fields[scheduleID] {
		if !keep[old.ID] {
			removed = append(removed, old.ID)
		}
	}
	r.fields[scheduleID] = fields
	return removed, nil
}

func (r *fakeRepo) ListItems(_ domain.Ctx, scheduleID int64) ([]*domain.Item, error) {
	return r.items[scheduleID], nil
}

func (r *fakeRepo) GetItem(_ domain.Ctx, scheduleID, id int64) (*domain.Item, error) {
	for _, it := range r.items[scheduleID] {
		if it.ID == id {
			return it, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) CreateItem(_ domain.Ctx, i *domain.Item, _ string) error {
	i.ID = r.id()
	r.items[i.ScheduleID] = append(r.items[i.ScheduleID], i)
	return nil
}

func (r *fakeRepo) UpdateItem(_ domain.Ctx, i *domain.Item, _ string) error {
	for idx, cur := range r.items[i.ScheduleID] {
		if cur.ID == i.ID {
			r.items[i.ScheduleID][idx] = i
			return nil
		}
	}
	return domain.ErrItemNotFound
}

func (r *fakeRepo) DeleteItem(_ domain.Ctx, scheduleID, id int64) error {
	list := r.items[scheduleID]
	for i, it := range list {
		if it.ID == id {
			r.items[scheduleID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return domain.ErrItemNotFound
}

func (r *fakeRepo) ReplaceItems(_ domain.Ctx, scheduleID int64, items []*domain.Item,
	_ func(*domain.Item) string) error {
	for _, i := range items {
		i.ID, i.ScheduleID = r.id(), scheduleID
	}
	r.items[scheduleID] = items
	return nil
}

func (r *fakeRepo) NormalizeItemWeeks(_ domain.Ctx, scheduleID int64, cycleWeeks int) (int, error) {
	r.adjusted = 0
	for _, it := range r.items[scheduleID] {
		if it.RepeatEvery != nil {
			continue
		}
		before := len(it.Weeks)
		it.Weeks = domain.NormalizeWeeks(it.Weeks, cycleWeeks)
		if len(it.Weeks) != before {
			r.adjusted++
		}
	}
	return r.adjusted, nil
}

func (r *fakeRepo) ItemsForDay(_ domain.Ctx, userID int64, companyIDs []int64, weekday int) ([]*domain.DayItem, error) {
	out := []*domain.DayItem{}
	for id, sc := range r.schedules {
		visible := sc.OwnerID == userID
		if !visible {
			for _, sh := range r.userShares[id] {
				if matchesTarget(sh, userID, companyIDs) {
					visible = true
				}
			}
		}
		if !visible {
			continue
		}
		for _, it := range r.items[id] {
			if it.Weekday == weekday {
				out = append(out, &domain.DayItem{Item: it, Schedule: sc})
			}
		}
	}
	return out, nil
}

func (r *fakeRepo) SearchItems(domain.Ctx, int64, []int64, string, int) ([]*domain.ItemScope, error) {
	return []*domain.ItemScope{}, nil
}

func (r *fakeRepo) CreateShare(_ domain.Ctx, s *domain.Share) error {
	s.ID = r.id()
	r.shares[s.Code] = s
	return nil
}

func (r *fakeRepo) ListShares(_ domain.Ctx, scheduleID int64) ([]*domain.Share, error) {
	out := []*domain.Share{}
	for _, s := range r.shares {
		if s.ScheduleID == scheduleID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *fakeRepo) GetShareByCode(_ domain.Ctx, code string) (*domain.Share, error) {
	return r.shares[code], nil
}

func (r *fakeRepo) DeleteShare(_ domain.Ctx, id, scheduleID int64) error {
	for code, s := range r.shares {
		if s.ID == id && s.ScheduleID == scheduleID {
			delete(r.shares, code)
			return nil
		}
	}
	return domain.ErrShareNotFound
}

func (r *fakeRepo) ListUserShares(_ domain.Ctx, scheduleID int64) ([]*domain.UserShare, error) {
	return r.userShares[scheduleID], nil
}

func (r *fakeRepo) PutUserShare(_ domain.Ctx, s *domain.UserShare) error {
	s.ID = r.id()
	r.userShares[s.ScheduleID] = append(r.userShares[s.ScheduleID], s)
	return nil
}

func (r *fakeRepo) DeleteUserShare(_ domain.Ctx, scheduleID int64, userID, companyID *int64) error {
	list := r.userShares[scheduleID]
	out := list[:0]
	for _, sh := range list {
		drop := (userID != nil && sh.UserID != nil && *sh.UserID == *userID) ||
			(companyID != nil && sh.CompanyID != nil && *sh.CompanyID == *companyID)
		if !drop {
			out = append(out, sh)
		}
	}
	r.userShares[scheduleID] = out
	return nil
}

type fakeUsers struct {
	companies map[int64][]int64
}

func (u *fakeUsers) GetUser(_ domain.Ctx, id int64) (*domain.User, error) {
	return &domain.User{ID: id, IsActive: true}, nil
}

func (u *fakeUsers) CompaniesOf(_ domain.Ctx, userID int64) ([]int64, error) {
	return u.companies[userID], nil
}

func (u *fakeUsers) Colleagues(domain.Ctx, int64, string, int) ([]*domain.UserRef, error) {
	return []*domain.UserRef{}, nil
}

func (u *fakeUsers) Companies(domain.Ctx, int64) ([]*domain.CompanyRef, error) {
	return []*domain.CompanyRef{}, nil
}

type fakeBus struct{ events []string }

func (b *fakeBus) Publish(_ domain.Ctx, event string, _ []string, _ any) {
	b.events = append(b.events, event)
}

func newService(t *testing.T) (*Service, *fakeRepo, *fakeBus) {
	t.Helper()
	repo, bus := newRepo(), &fakeBus{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(Deps{Repo: repo, Users: &fakeUsers{companies: map[int64][]int64{}}, Bus: bus, Log: log}), repo, bus
}

func ctx() context.Context { return context.Background() }

// ── Тесты ────────────────────────────────────────────────────────

func TestCreateScheduleAnchorsToMonday(t *testing.T) {
	svc, _, bus := newService(t)
	anchor, _ := time.Parse(domain.DateLayout, "2026-09-03") // четверг
	view, err := svc.CreateSchedule(ctx(), 1, ScheduleInput{
		Name: "Учёба", CycleWeeks: 2, CycleAnchor: anchor,
	})
	if err != nil {
		t.Fatalf("создание расписания: %v", err)
	}
	if got := view.Schedule.CycleAnchor.Format(domain.DateLayout); got != "2026-08-31" {
		t.Errorf("якорь цикла приводится к понедельнику, получено %s", got)
	}
	if len(view.Schedule.WeekLabels) != 2 {
		t.Errorf("названий недель столько же, сколько недель в цикле: %v", view.Schedule.WeekLabels)
	}
	if len(bus.events) != 1 || bus.events[0] != "schedule:created" {
		t.Errorf("о создании сообщается событием, получено %v", bus.events)
	}
}

func TestStrangerSeesNothing(t *testing.T) {
	svc, _, _ := newService(t)
	view, err := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Учёба"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetSchedule(ctx(), 2, view.Schedule.ID); err != domain.ErrScheduleNotFound {
		t.Errorf("постороннему расписание не существует, получено %v", err)
	}
	if _, err := svc.CreateItem(ctx(), 2, view.Schedule.ID, ItemInput{
		Title: "Чужое", StartMin: 60, EndMin: 120,
	}); err != domain.ErrScheduleNotFound {
		t.Errorf("посторонний не может завести занятие, получено %v", err)
	}
}

// Тому, кто расписание уже видит, отказ называется честно: «только чтение», а
// не «не найдено» — иначе кнопка правки врала бы про открытое расписание.
func TestSharedUserGetsReadOnlyError(t *testing.T) {
	svc, _, _ := newService(t)
	view, err := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Учёба"})
	if err != nil {
		t.Fatal(err)
	}
	guest := int64(2)
	if _, err := svc.ShareWith(ctx(), 1, view.Schedule.ID, &guest, nil); err != nil {
		t.Fatalf("шаринг: %v", err)
	}
	shared, err := svc.GetSchedule(ctx(), guest, view.Schedule.ID)
	if err != nil {
		t.Fatalf("адресат видит расписание: %v", err)
	}
	if shared.CanEdit {
		t.Error("адресату расписание доступно только на чтение")
	}
	_, err = svc.CreateItem(ctx(), guest, view.Schedule.ID, ItemInput{
		Title: "Своё", StartMin: 60, EndMin: 120,
	})
	if err != domain.ErrReadOnly {
		t.Errorf("правка адресатом запрещена явно, получено %v", err)
	}
}

func TestShareWithSelfRejected(t *testing.T) {
	svc, _, _ := newService(t)
	view, _ := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Учёба"})
	me := int64(1)
	if _, err := svc.ShareWith(ctx(), 1, view.Schedule.ID, &me, nil); err != domain.ErrShareSelf {
		t.Errorf("делиться с собой незачем, получено %v", err)
	}
	if _, err := svc.ShareWith(ctx(), 1, view.Schedule.ID, nil, nil); err != domain.ErrNoAudience {
		t.Errorf("адресат обязателен, получено %v", err)
	}
}

// Укорочение цикла не должно ронять занятия со шкалы: номера недель за его
// пределами отбрасываются, а опустевший набор означает «каждую неделю».
func TestShortenCycleKeepsItemsVisible(t *testing.T) {
	svc, repo, _ := newService(t)
	view, _ := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Учёба", CycleWeeks: 4})
	id := view.Schedule.ID

	if _, err := svc.CreateItem(ctx(), 1, id, ItemInput{
		Title: "Матан", Weekday: 0, StartMin: 600, EndMin: 700, Weeks: []int{3, 4},
	}); err != nil {
		t.Fatalf("занятие: %v", err)
	}
	weeks := 2
	updated, adjusted, err := svc.UpdateSchedule(ctx(), 1, id, ScheduleUpdate{CycleWeeks: &weeks})
	if err != nil {
		t.Fatalf("правка цикла: %v", err)
	}
	if adjusted != 1 {
		t.Errorf("о тронутых занятиях сообщается числом, получено %d", adjusted)
	}
	if len(updated.Schedule.WeekLabels) != 2 {
		t.Errorf("названия недель обрезаются под цикл: %v", updated.Schedule.WeekLabels)
	}
	item := repo.items[id][0]
	if len(item.Weeks) != 0 {
		t.Fatalf("недели вне цикла отброшены, получено %v", item.Weeks)
	}
	monday, _ := time.Parse(domain.DateLayout, "2026-08-31")
	if !item.OccursOn(updated.Schedule, monday) {
		t.Error("занятие без недель идёт каждую неделю, а не пропадает")
	}
}

// Категория чужого расписания к занятию не пристаёт: она бы ссылалась на
// справочник, которого у этого расписания нет.
func TestForeignCategoryDropped(t *testing.T) {
	svc, _, _ := newService(t)
	mine, _ := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Учёба"})
	other, _ := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Работа"})
	cat, err := svc.CreateCategory(ctx(), 1, other.Schedule.ID, CategoryInput{Name: "Смена", Color: "green"})
	if err != nil {
		t.Fatal(err)
	}
	item, err := svc.CreateItem(ctx(), 1, mine.Schedule.ID, ItemInput{
		Title: "Матан", StartMin: 600, EndMin: 700, CategoryID: &cat.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.CategoryID != nil {
		t.Errorf("чужая категория отбрасывается, получено %v", *item.CategoryID)
	}
}

func TestCategoryColorNormalized(t *testing.T) {
	svc, _, _ := newService(t)
	view, _ := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Учёба"})
	cat, err := svc.CreateCategory(ctx(), 1, view.Schedule.ID, CategoryInput{Name: "Пары", Color: "#ff0000"})
	if err != nil {
		t.Fatal(err)
	}
	if cat.Color != domain.CategoryColors[0] {
		t.Errorf("незнакомый цвет приводится к палитре, получено %q", cat.Color)
	}
}

// Импорт формата прототипа: числитель и знаменатель разворачиваются в цикл из
// двух недель, вид и место становятся полями карточки.
func TestImportLegacyPrototype(t *testing.T) {
	svc, _, _ := newService(t)
	raw := []byte(`{"items":[
		{"day":0,"category":"study","week":"num","start":"10:20","end":"11:55",
		 "title":"Вейвлеты","short":"Вейвлеты","kind":"лекция","place":"2-3.10"},
		{"day":2,"category":"work","week":"both","start":"14:00","end":"18:00","title":"Работа"}
	]}`)
	view, err := svc.ImportNew(ctx(), 1, raw, "Учёба")
	if err != nil {
		t.Fatalf("импорт: %v", err)
	}
	sc := view.Schedule
	if sc.CycleWeeks != 2 {
		t.Errorf("числитель и знаменатель — цикл из двух недель, получено %d", sc.CycleWeeks)
	}
	if sc.WeekLabel(1) != "Числитель" || sc.WeekLabel(2) != "Знаменатель" {
		t.Errorf("недели названы как в прототипе, получено %v", sc.WeekLabels)
	}
	if len(sc.Categories) != 2 {
		t.Fatalf("категории заводятся только использованные, получено %d", len(sc.Categories))
	}
	if len(sc.Fields) != 2 {
		t.Fatalf("вид и место становятся полями, получено %d", len(sc.Fields))
	}
	if len(view.Items) != 2 {
		t.Fatalf("перенесены оба занятия, получено %d", len(view.Items))
	}
	first := view.Items[0]
	if first.StartMin != 620 || first.EndMin != 715 {
		t.Errorf("время разобрано из ЧЧ:ММ, получено %d–%d", first.StartMin, first.EndMin)
	}
	if len(first.Weeks) != 1 || first.Weeks[0] != 1 {
		t.Errorf("num — первая неделя цикла, получено %v", first.Weeks)
	}
	if first.CategoryID == nil {
		t.Error("категория занятия сопоставлена по названию")
	}
	if len(first.Data) != 2 {
		t.Errorf("значения полей перенесены, получено %v", first.Data)
	}
}

// Экспорт и импорт — обратные операции: файл возвращает расписание как было.
func TestExportImportRoundTrip(t *testing.T) {
	svc, _, _ := newService(t)
	view, _ := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Учёба", CycleWeeks: 2})
	id := view.Schedule.ID
	cat, _ := svc.CreateCategory(ctx(), 1, id, CategoryInput{Name: "Лекции", Color: "blue"})
	if _, err := svc.ReplaceFields(ctx(), 1, id, []*domain.Field{
		{Label: "Место", Type: domain.FieldText, ShowOnBlock: true},
	}); err != nil {
		t.Fatal(err)
	}
	fields, _ := svc.repo.ListFields(ctx(), id)
	if _, err := svc.CreateItem(ctx(), 1, id, ItemInput{
		Title: "Матан", Weekday: 1, StartMin: 600, EndMin: 695, Weeks: []int{2},
		CategoryID: &cat.ID,
		Data:       map[string]any{fieldKey(fields[0].ID): "2-3.10"},
	}); err != nil {
		t.Fatal(err)
	}

	raw, name, err := svc.Export(ctx(), 1, id)
	if err != nil {
		t.Fatalf("выгрузка: %v", err)
	}
	if name != "Учёба" {
		t.Errorf("имя файла берётся у расписания, получено %q", name)
	}

	restored, err := svc.ImportNew(ctx(), 1, raw, "Копия")
	if err != nil {
		t.Fatalf("загрузка: %v", err)
	}
	if restored.Schedule.CycleWeeks != 2 || len(restored.Items) != 1 {
		t.Fatalf("расписание восстановлено не целиком: %d недель, %d занятий",
			restored.Schedule.CycleWeeks, len(restored.Items))
	}
	item := restored.Items[0]
	if item.Title != "Матан" || item.StartMin != 600 || len(item.Weeks) != 1 || item.Weeks[0] != 2 {
		t.Errorf("занятие восстановлено не полностью: %+v", item)
	}
	if item.CategoryID == nil {
		t.Error("категория сопоставлена по названию — id в другом аккаунте другой")
	}
	if len(item.Data) != 1 {
		t.Errorf("значение поля восстановлено по названию, получено %v", item.Data)
	}
}

// fieldKey — строковый ключ поля в data занятия.
func fieldKey(id int64) string { return strconv.FormatInt(id, 10) }

// Живая плитка считает «сейчас» и «дальше» по присланному дню и минуте.
func TestAgendaNowAndNext(t *testing.T) {
	svc, _, _ := newService(t)
	view, _ := svc.CreateSchedule(ctx(), 1, ScheduleInput{Name: "Учёба"})
	id := view.Schedule.ID
	// Понедельник 2026-08-31: пара 10:20–11:55 и следующая 12:10–13:45.
	if _, err := svc.CreateItem(ctx(), 1, id, ItemInput{
		Title: "Вейвлеты", Weekday: 0, StartMin: 620, EndMin: 715,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateItem(ctx(), 1, id, ItemInput{
		Title: "Физика", Weekday: 0, StartMin: 730, EndMin: 825,
	}); err != nil {
		t.Fatal(err)
	}
	monday, _ := time.Parse(domain.DateLayout, "2026-08-31")

	agenda, err := svc.Agenda(ctx(), 1, monday, 640) // 10:40 — идёт первая пара
	if err != nil {
		t.Fatal(err)
	}
	if agenda.Now == nil || agenda.Now.Title != "Вейвлеты" {
		t.Errorf("сейчас идёт первая пара, получено %+v", agenda.Now)
	}
	if agenda.Next == nil || agenda.Next.Title != "Физика" {
		t.Errorf("следующей идёт вторая пара, получено %+v", agenda.Next)
	}
	if agenda.BusyMin != 95+95 {
		t.Errorf("занято обе пары, получено %d", agenda.BusyMin)
	}
	// Перемена в 15 минут окном не считается: порог расписания — 20 минут.
	if agenda.GapMin != 0 {
		t.Errorf("перемена короче порога окном не считается, получено %d", agenda.GapMin)
	}
	gap := 10
	if _, _, err := svc.UpdateSchedule(ctx(), 1, id, ScheduleUpdate{GapMin: &gap}); err != nil {
		t.Fatal(err)
	}
	agenda, err = svc.Agenda(ctx(), 1, monday, 640)
	if err != nil {
		t.Fatal(err)
	}
	if agenda.GapMin != 15 {
		t.Errorf("со сниженным порогом та же перемена — окно, получено %d", agenda.GapMin)
	}
	if agenda.Total != 2 {
		t.Errorf("сегодня два занятия, получено %d", agenda.Total)
	}
}
