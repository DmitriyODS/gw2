package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/token"
)

// ── Фейки портов (без БД/Redis, как в callsvc) ───────────────────

type fakeRepo struct {
	users     map[int64]*domain.User
	roles     map[int64]*domain.Role
	directors map[int64]bool                         // userID → корневой Руководитель компании
	members   map[int64]map[int64]*domain.Membership // userID → companyID → членство
	nextID    int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users: map[int64]*domain.User{},
		roles: map[int64]*domain.Role{
			1: {ID: 1, Name: "Сотрудник", Level: domain.LevelEmployee},
			2: {ID: 2, Name: "Менеджер", Level: domain.LevelManager},
			3: {ID: 3, Name: "Руководитель", Level: domain.LevelDirector},
			4: {ID: 4, Name: "Администратор", Level: domain.LevelAdmin},
		},
		directors: map[int64]bool{},
		members:   map[int64]map[int64]*domain.Membership{},
	}
}

func (r *fakeRepo) add(u *domain.User) *domain.User {
	r.nextID++
	u.ID = r.nextID
	u.CreatedAt = time.Now()
	r.users[u.ID] = u
	// Авто-членство из первичной компании пользователя (Company-указатель общий,
	// чтобы мутация u.Company.IsActive в тестах отражалась и в членстве).
	if u.CompanyID != nil {
		if r.members[u.ID] == nil {
			r.members[u.ID] = map[int64]*domain.Membership{}
		}
		r.members[u.ID][*u.CompanyID] = &domain.Membership{
			CompanyID: *u.CompanyID, Company: u.Company, Role: u.Role, CreatedAt: u.CreatedAt,
		}
	}
	return u
}

func (r *fakeRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (r *fakeRepo) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Login == login {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email != nil && strings.EqualFold(*u.Email, email) {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) ListVisible(_ context.Context) ([]*domain.User, error) {
	var out []*domain.User
	for _, u := range r.users {
		if !u.IsHidden {
			cp := *u
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeRepo) SearchDirectory(_ context.Context, query string, excludeID int64,
	companyID *int64) ([]*domain.User, error) {
	var out []*domain.User
	for _, u := range r.users {
		if u.IsHidden || u.ID == excludeID {
			continue
		}
		if companyID != nil && (u.CompanyID == nil || *u.CompanyID != *companyID) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(u.FIO), strings.ToLower(query)) &&
			!strings.Contains(strings.ToLower(u.Login), strings.ToLower(query)) {
			continue
		}
		cp := *u
		out = append(out, &cp)
	}
	return out, nil
}

func (r *fakeRepo) Create(_ context.Context, u *domain.User) error {
	r.add(u)
	return nil
}

func (r *fakeRepo) UpdateFields(_ context.Context, id int64, fields map[string]any) error {
	u := r.users[id]
	for k, v := range fields {
		switch k {
		case "login":
			u.Login = v.(string)
		case "fio":
			u.FIO = v.(string)
		case "hash_password":
			u.HashPassword = v.(string)
		case "is_default_pass":
			u.IsDefaultPass = v.(bool)
		case "is_hidden":
			u.IsHidden = v.(bool)
		case "role_id":
			role := r.roles[v.(int64)]
			u.Role = *role
		case "avatar_path":
			if v == nil {
				u.AvatarPath = nil
			} else {
				p := v.(string)
				u.AvatarPath = &p
			}
		case "company_id":
			if v == nil {
				u.CompanyID = nil
			} else {
				cid := v.(int64)
				u.CompanyID = &cid
			}
		}
	}
	return nil
}

func (r *fakeRepo) GetRole(_ context.Context, roleID int64) (*domain.Role, error) {
	role, ok := r.roles[roleID]
	if !ok {
		return nil, nil
	}
	cp := *role
	return &cp, nil
}

func (r *fakeRepo) ListRoles(_ context.Context) ([]*domain.Role, error) {
	out := make([]*domain.Role, 0, len(r.roles))
	for _, role := range r.roles {
		cp := *role
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Level < out[j].Level })
	return out, nil
}

func (r *fakeRepo) CountVisibleByLevel(_ context.Context, level int) (int, error) {
	n := 0
	for _, u := range r.users {
		if !u.IsHidden && u.Role.Level == level {
			n++
		}
	}
	return n, nil
}

func (r *fakeRepo) IsCompanyDirector(_ context.Context, userID int64) (bool, error) {
	return r.directors[userID], nil
}

func (r *fakeRepo) HashPassword(_ context.Context, password string) (string, error) {
	return "hash:" + password, nil
}

func (r *fakeRepo) VerifyPassword(_ context.Context, password, hash string) (bool, error) {
	return hash == "hash:"+password, nil
}

// ── Членство (user_companies) ──

func (r *fakeRepo) ListMemberships(_ context.Context, userID int64) ([]domain.Membership, error) {
	var out []domain.Membership
	for _, m := range r.members[userID] {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].CompanyID < out[j].CompanyID
	})
	return out, nil
}

func (r *fakeRepo) GetMembership(_ context.Context, userID, companyID int64) (*domain.Membership, error) {
	if m, ok := r.members[userID][companyID]; ok {
		cp := *m
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeRepo) AddMembership(_ context.Context, userID, companyID, roleID int64) error {
	if r.members[userID] == nil {
		r.members[userID] = map[int64]*domain.Membership{}
	}
	if _, ok := r.members[userID][companyID]; ok {
		return nil // ON CONFLICT DO NOTHING
	}
	r.members[userID][companyID] = &domain.Membership{
		CompanyID: companyID,
		Company:   &domain.CompanyRef{ID: companyID, Name: "Компания", IsActive: true},
		Role:      *r.roles[roleID],
		CreatedAt: time.Now(),
	}
	return nil
}

func (r *fakeRepo) RemoveMembership(_ context.Context, userID, companyID int64) error {
	delete(r.members[userID], companyID)
	return nil
}

func (r *fakeRepo) UpdateMembershipRole(_ context.Context, userID, companyID, roleID int64) error {
	if m, ok := r.members[userID][companyID]; ok {
		m.Role = *r.roles[roleID]
	}
	return nil
}

func (r *fakeRepo) CountCompanyMembersByLevel(_ context.Context, companyID int64, level int) (int, error) {
	n := 0
	for uid, byCompany := range r.members {
		if m, ok := byCompany[companyID]; ok && m.Role.Level == level && !r.users[uid].IsHidden {
			n++
		}
	}
	return n, nil
}

func (r *fakeRepo) SearchDirectoryMembers(_ context.Context, query string, excludeID, companyID int64) ([]*domain.User, error) {
	var out []*domain.User
	for uid, byCompany := range r.members {
		m, ok := byCompany[companyID]
		if !ok {
			continue
		}
		u := r.users[uid]
		if u == nil || u.IsHidden || u.ID == excludeID {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(u.FIO), strings.ToLower(query)) &&
			!strings.Contains(strings.ToLower(u.Login), strings.ToLower(query)) {
			continue
		}
		cp := *u
		cp.Role = m.Role
		cp.CompanyID = &companyID
		out = append(out, &cp)
	}
	return out, nil
}

func (r *fakeRepo) SearchNonMembers(_ context.Context, query string, companyID int64) ([]*domain.User, error) {
	var out []*domain.User
	for _, u := range r.users {
		if u.IsHidden {
			continue
		}
		if _, ok := r.members[u.ID][companyID]; ok {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(u.FIO), strings.ToLower(query)) &&
			!strings.Contains(strings.ToLower(u.Login), strings.ToLower(query)) {
			continue
		}
		cp := *u
		out = append(out, &cp)
	}
	return out, nil
}

func (r *fakeRepo) SyncPrimaryCompany(_ context.Context, userID int64) error {
	ms, _ := r.ListMemberships(context.Background(), userID)
	u := r.users[userID]
	if u == nil {
		return nil
	}
	if len(ms) == 0 {
		u.CompanyID = nil
		return nil
	}
	primary := ms[0]
	u.CompanyID = &primary.CompanyID
	u.Role = primary.Role
	return nil
}

func (r *fakeRepo) CompanyActive(_ context.Context, _ *int64) (bool, error) { return true, nil }

type fakeThrottle struct {
	locked   map[string]int
	failures map[string]int
}

func newFakeThrottle() *fakeThrottle {
	return &fakeThrottle{locked: map[string]int{}, failures: map[string]int{}}
}

func (t *fakeThrottle) LockRemaining(_ context.Context, login string) int { return t.locked[login] }
func (t *fakeThrottle) RegisterFailure(_ context.Context, login string) int {
	t.failures[login]++
	return 0
}
func (t *fakeThrottle) RegisterSuccess(_ context.Context, login string) { t.failures[login] = 0 }

type fakeAvatars struct{ saved, deleted []string }

func (a *fakeAvatars) Save(_ []byte) (string, error) {
	path := "avatars/test.png"
	a.saved = append(a.saved, path)
	return path, nil
}
func (a *fakeAvatars) Delete(p string) { a.deleted = append(a.deleted, p) }

func (a *fakeAvatars) ListFiles() ([]domain.AvatarFile, error) { return nil, nil }
func (a *fakeAvatars) WriteFile(string, []byte) error          { return nil }

// fakeCompanies — in-memory компании (порт CompanyRepository).
type fakeCompanies struct {
	seq       int64
	companies map[int64]*domain.Company
}

func newFakeCompanies() *fakeCompanies {
	return &fakeCompanies{companies: map[int64]*domain.Company{}}
}

func (f *fakeCompanies) ListCompanies(_ context.Context) ([]*domain.Company, error) {
	out := make([]*domain.Company, 0, len(f.companies))
	for _, c := range f.companies {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (f *fakeCompanies) GetCompany(_ context.Context, id int64) (*domain.Company, error) {
	return f.companies[id], nil
}

func (f *fakeCompanies) GetCompanyByName(_ context.Context, name string) (*domain.Company, error) {
	for _, c := range f.companies {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, nil
}

func (f *fakeCompanies) GetCompanyByInviteCode(_ context.Context, code string) (*domain.Company, error) {
	for _, c := range f.companies {
		if c.InviteCode != nil && *c.InviteCode == code {
			return c, nil
		}
	}
	return nil, nil
}

func (f *fakeCompanies) CreateCompany(_ context.Context, c *domain.Company) error {
	f.seq++
	c.ID = f.seq
	c.IsActive = true
	c.CreatedAt = time.Now().UTC()
	f.companies[c.ID] = c
	return nil
}

func (f *fakeCompanies) UpdateCompanyFields(_ context.Context, id int64, fields map[string]any) error {
	c, ok := f.companies[id]
	if !ok {
		return nil
	}
	for k, v := range fields {
		switch k {
		case "name":
			c.Name = v.(string)
		case "description":
			c.Description, _ = v.(*string)
		case "director_id":
			c.DirectorID, _ = v.(*int64)
		case "is_active":
			c.IsActive = v.(bool)
		case "settings":
			c.Settings = v.(map[string]any)
		case "invite_code":
			switch x := v.(type) {
			case *string:
				c.InviteCode = x
			case string:
				c.InviteCode = &x
			default:
				c.InviteCode = nil
			}
		}
	}
	return nil
}

func (f *fakeCompanies) DeleteCompany(_ context.Context, id int64) error {
	delete(f.companies, id)
	return nil
}

func (f *fakeCompanies) CompanyStats(_ context.Context, ids []int64) (map[int64]domain.CompanyStats, error) {
	out := map[int64]domain.CompanyStats{}
	for _, id := range ids {
		out[id] = domain.CompanyStats{}
	}
	return out, nil
}

// fakeBackup — заглушка BackupStore (export/import в unit-тестах не гоняем).
type fakeBackup struct{}

func (fakeBackup) ExportData(context.Context) (*domain.BackupData, error) {
	return &domain.BackupData{}, nil
}
func (fakeBackup) ImportData(context.Context, *domain.BackupData) error { return nil }

// ── Хелперы ──────────────────────────────────────────────────────

const testPrivateHex = "b4cbfb43df4ce210727d953e4a713307fa19bb7d9f85041438d9e11b942a37741eb9dbbbbc047c03fd70604e0071f0987e16b28b757225c11f00415d0e20b1a2"
const testRefreshHex = "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"

func newTestService(t *testing.T) (*Service, *fakeRepo, *fakeThrottle) {
	t.Helper()
	iss, err := token.NewIssuer(testPrivateHex, testRefreshHex, 15*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	repo := newFakeRepo()
	throttle := newFakeThrottle()
	svc := New(repo, newFakeCompanies(), fakeBackup{}, throttle, iss, &fakeAvatars{}, slog.Default())
	return svc, repo, throttle
}

func employee(repo *fakeRepo, login string, companyID *int64) *domain.User {
	return repo.add(&domain.User{
		FIO: "Тест " + login, Login: login, HashPassword: "hash:secret123",
		Role: *repo.roles[1], CompanyID: companyID,
		Company: companyOf(companyID), IsDefaultPass: false,
	})
}

func companyOf(id *int64) *domain.CompanyRef {
	if id == nil {
		return nil
	}
	return &domain.CompanyRef{ID: *id, Name: "Компания", IsActive: true,
		Settings: map[string]any{"uses_calls": true}}
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	de := domain.AsDomainError(err)
	if de == nil || de.Code != code {
		t.Fatalf("ожидалась ошибка %s, получено: %v", code, err)
	}
}

// ── Auth ─────────────────────────────────────────────────────────

func TestLoginSuccess(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)

	sess, err := svc.Login(context.Background(), dto.LoginRequest{Login: "ivanov", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if sess.UserID != u.ID || sess.ForceChange || sess.AccessToken == "" || sess.RefreshToken == "" {
		t.Fatalf("некорректная сессия: %+v", sess)
	}
	if sess.CompanyID == nil || *sess.CompanyID != cid || sess.CompanyName == nil {
		t.Fatalf("клеймы компании не заполнены: %+v", sess)
	}
	if sess.RoleLevel != domain.LevelEmployee {
		t.Fatalf("role_level=%d", sess.RoleLevel)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc, repo, throttle := newTestService(t)
	employee(repo, "ivanov", nil)

	_, err := svc.Login(context.Background(), dto.LoginRequest{Login: "ivanov", Password: "wrong"})
	wantCode(t, err, "INVALID_CREDENTIALS")
	if throttle.failures["ivanov"] != 1 {
		t.Fatal("неудачная попытка не учтена")
	}
}

func TestLoginLocked(t *testing.T) {
	svc, repo, throttle := newTestService(t)
	employee(repo, "ivanov", nil)
	throttle.locked["ivanov"] = 40

	_, err := svc.Login(context.Background(), dto.LoginRequest{Login: "ivanov", Password: "secret123"})
	wantCode(t, err, "TOO_MANY_ATTEMPTS")
	de := domain.AsDomainError(err)
	if de.HTTPStatus != 429 || de.Extra["retry_after_sec"] != 40 {
		t.Fatalf("ожидался 429 c retry_after_sec=40: %+v", de)
	}
}

func TestLoginCompanyDisabled(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)
	u.Company.IsActive = false

	_, err := svc.Login(context.Background(), dto.LoginRequest{Login: "ivanov", Password: "secret123"})
	wantCode(t, err, "COMPANY_DISABLED")
}

func TestRefreshRoundTrip(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	employee(repo, "ivanov", &cid)

	sess, err := svc.Login(context.Background(), dto.LoginRequest{Login: "ivanov", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	got, err := svc.Refresh(context.Background(), sess.RefreshToken)
	if err != nil || got.UserID != sess.UserID || got.AccessToken == "" {
		t.Fatalf("Refresh: %+v, %v", got, err)
	}
	// Активная компания переживает refresh (зашита в refresh-токен).
	if got.CompanyID == nil || *got.CompanyID != cid {
		t.Fatalf("refresh потерял компанию: %+v", got)
	}
	if _, err := svc.Refresh(context.Background(), "v4.local.garbage"); err == nil {
		t.Fatal("мусорный refresh принят")
	}
}

func TestChangeDefault(t *testing.T) {
	svc, repo, _ := newTestService(t)
	u := repo.add(&domain.User{
		FIO: "Новичок", Login: "novice", HashPassword: "hash:novice123",
		Role: *repo.roles[1], IsDefaultPass: true,
	})

	_, err := svc.ChangeDefault(context.Background(), dto.ChangeDefaultRequest{
		UserID: u.ID, NewLogin: "hero", NewPassword: "supersecret", ConfirmPassword: "nope",
	})
	wantCode(t, err, "PASSWORDS_MISMATCH")

	sess, err := svc.ChangeDefault(context.Background(), dto.ChangeDefaultRequest{
		UserID: u.ID, NewLogin: "hero", NewPassword: "supersecret", ConfirmPassword: "supersecret",
	})
	if err != nil || sess.ForceChange {
		t.Fatalf("ChangeDefault: %+v, %v", sess, err)
	}
	if repo.users[u.ID].Login != "hero" || repo.users[u.ID].IsDefaultPass {
		t.Fatal("логин/флаг не обновлены")
	}

	// Повторная смена — уже нельзя.
	_, err = svc.ChangeDefault(context.Background(), dto.ChangeDefaultRequest{
		UserID: u.ID, NewLogin: "hero2", NewPassword: "supersecret", ConfirmPassword: "supersecret",
	})
	wantCode(t, err, "ALREADY_CHANGED")
}

// ── Users ────────────────────────────────────────────────────────

func TestCreateUserRoleGuard(t *testing.T) {
	svc, repo, _ := newTestService(t)
	director := repo.add(&domain.User{FIO: "Дир", Login: "dir", Role: *repo.roles[3]})

	_, err := svc.CreateUser(context.Background(), director, dto.CreateUserRequest{
		FIO: "Хакер", Login: "hacker", RoleID: 4,
	})
	wantCode(t, err, "ROLE_LEVEL_FORBIDDEN")

	created, err := svc.CreateUser(context.Background(), director, dto.CreateUserRequest{
		FIO: "Новый", Login: "newbie", RoleID: 1,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if !created.IsDefaultPass {
		t.Fatal("без пароля должен быть is_default_pass=true")
	}
	if repo.users[created.ID].HashPassword != "hash:newbie123" {
		t.Fatal("дефолтный пароль должен быть <login>123")
	}
}

func TestCreateUserDuplicateLogin(t *testing.T) {
	svc, repo, _ := newTestService(t)
	admin := repo.add(&domain.User{FIO: "Админ", Login: "admin", Role: *repo.roles[4]})
	employee(repo, "ivanov", nil)

	_, err := svc.CreateUser(context.Background(), admin, dto.CreateUserRequest{
		FIO: "Дубль", Login: "ivanov", RoleID: 1,
	})
	wantCode(t, err, "LOGIN_TAKEN")
}

func TestHideUserGuards(t *testing.T) {
	svc, repo, _ := newTestService(t)
	rootAdmin := repo.add(&domain.User{FIO: "Рут", Login: "root", Role: *repo.roles[4], IsRootAdmin: true})
	admin := repo.add(&domain.User{FIO: "Админ", Login: "admin", Role: *repo.roles[4]})
	director := repo.add(&domain.User{FIO: "Дир", Login: "dir", Role: *repo.roles[3]})
	repo.directors[director.ID] = true

	// Себя — нельзя.
	wantCode(t, svc.HideUser(context.Background(), admin, admin.ID), "SELF_HIDE")
	// Корневого админа — никому.
	wantCode(t, svc.HideUser(context.Background(), admin, rootAdmin.ID), "ROOT_ADMIN")
	// Выше себя — нельзя.
	wantCode(t, svc.HideUser(context.Background(), director, admin.ID), "ROLE_LEVEL_FORBIDDEN")
	// Корневого Руководителя компании — только Админ системы.
	otherDirector := repo.add(&domain.User{FIO: "Дир2", Login: "dir2", Role: *repo.roles[3]})
	repo.directors[otherDirector.ID] = true
	wantCode(t, svc.HideUser(context.Background(), director, otherDirector.ID), "ROOT_DIRECTOR")
	// Админ системы может скрыть корневого Руководителя.
	if err := svc.HideUser(context.Background(), admin, otherDirector.ID); err != nil {
		t.Fatalf("admin hide director: %v", err)
	}
}

func TestHideLastAdmin(t *testing.T) {
	svc, repo, _ := newTestService(t)
	admin1 := repo.add(&domain.User{FIO: "А1", Login: "a1", Role: *repo.roles[4]})
	admin2 := repo.add(&domain.User{FIO: "А2", Login: "a2", Role: *repo.roles[4]})

	if err := svc.HideUser(context.Background(), admin1, admin2.ID); err != nil {
		t.Fatalf("hide admin2: %v", err)
	}
	// admin1 остался единственным видимым админом — третий не скроет его…
	admin3 := repo.add(&domain.User{FIO: "А3", Login: "a3", Role: *repo.roles[4]})
	_ = admin3
	// admin3 теперь тоже админ: их двое видимых, скрыть admin1 можно.
	if err := svc.HideUser(context.Background(), admin3, admin1.ID); err != nil {
		t.Fatalf("hide admin1: %v", err)
	}
	// Остался один admin3 — last-admin guard.
	admin4view := repo.add(&domain.User{FIO: "Дир", Login: "d", Role: *repo.roles[3]})
	wantCode(t, svc.HideUser(context.Background(), admin4view, admin3.ID), "ROLE_LEVEL_FORBIDDEN")
	wantCode(t, svc.HideUser(context.Background(), admin3, admin3.ID), "SELF_HIDE")
}

func TestAssignRoleGuards(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	// Админ системы оперирует в контексте компании (?company_id= → actor.CompanyID
	// проставляет хендлер; в юнит-тесте задаём напрямую).
	admin := repo.add(&domain.User{FIO: "Админ", Login: "admin", Role: *repo.roles[4], CompanyID: &cid})
	repo.add(&domain.User{FIO: "Запас", Login: "admin2", Role: *repo.roles[4]})
	emp := employee(repo, "ivanov", &cid)

	wantCode(t, errOf(svc.AssignRole(context.Background(), admin, admin.ID, 1)), "SELF_ROLE_CHANGE")

	updated, err := svc.AssignRole(context.Background(), admin, emp.ID, 2)
	if err != nil || updated.Role.Level != domain.LevelManager {
		t.Fatalf("AssignRole: %+v, %v", updated, err)
	}
	// Роль сменилась именно в членстве компании cid.
	m, _ := repo.GetMembership(context.Background(), emp.ID, cid)
	if m == nil || m.Role.Level != domain.LevelManager {
		t.Fatalf("членство не обновлено: %+v", m)
	}
}

func TestResetPasswordGuards(t *testing.T) {
	svc, repo, _ := newTestService(t)
	director := repo.add(&domain.User{FIO: "Дир", Login: "dir", Role: *repo.roles[3]})
	admin := repo.add(&domain.User{FIO: "Админ", Login: "admin", Role: *repo.roles[4]})
	emp := employee(repo, "ivanov", nil)

	wantCode(t, svc.ResetPassword(context.Background(), director, admin.ID), "ROLE_LEVEL_FORBIDDEN")
	wantCode(t, svc.ResetPassword(context.Background(), director, director.ID), "SELF_RESET")

	if err := svc.ResetPassword(context.Background(), director, emp.ID); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if !repo.users[emp.ID].IsDefaultPass || repo.users[emp.ID].HashPassword != "hash:ivanov123" {
		t.Fatal("пароль не сброшен на дефолтный")
	}
}

func TestUpdateMePassword(t *testing.T) {
	svc, repo, _ := newTestService(t)
	u := employee(repo, "ivanov", nil)
	newPass := "newsecret99"
	wrong := "badpass"

	_, err := svc.UpdateMe(context.Background(), u.ID, dto.UpdateMeRequest{
		NewPassword: &newPass, ConfirmPassword: &newPass, CurrentPassword: &wrong,
	})
	wantCode(t, err, "WRONG_PASSWORD")

	current := "secret123"
	_, err = svc.UpdateMe(context.Background(), u.ID, dto.UpdateMeRequest{
		NewPassword: &newPass, ConfirmPassword: &newPass, CurrentPassword: &current,
	})
	if err != nil {
		t.Fatalf("UpdateMe: %v", err)
	}
	if repo.users[u.ID].HashPassword != "hash:newsecret99" {
		t.Fatal("пароль не сменился")
	}
}

func TestDirectoryScoping(t *testing.T) {
	svc, repo, _ := newTestService(t)
	c1, c2 := int64(1), int64(2)
	me := employee(repo, "ivanov", &c1)
	employee(repo, "petrov", &c1)
	employee(repo, "sidorov", &c2)

	// Каталог — члены указанной компании (req.CompanyID разрешает хендлер:
	// активная компания актора / ?company_id= админа). Для c1 — только её члены.
	got, err := svc.Directory(context.Background(), dto.DirectoryRequest{
		ActorID: me.ID, CompanyID: &c1,
	})
	if err != nil {
		t.Fatalf("Directory: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ожидалось 2 члена c1, получено %d: %+v", len(got), got)
	}
	for _, u := range got {
		if u.CompanyID == nil || *u.CompanyID != c1 {
			t.Fatalf("в выдаче чужая компания: %+v", u)
		}
	}

	// Администратор системы выбирает компанию явно.
	sysAdmin := repo.add(&domain.User{FIO: "Админ", Login: "admin", Role: *repo.roles[4]})
	got, err = svc.Directory(context.Background(), dto.DirectoryRequest{
		ActorID: sysAdmin.ID, CompanyID: &c2,
	})
	if err != nil || len(got) != 1 || got[0].Login != "sidorov" {
		t.Fatalf("админ по c2: %+v, %v", got, err)
	}
}

func errOf(_ any, err error) error { return err }

// ── Multi-company ────────────────────────────────────────────────

func TestLoginGateMultiCompany(t *testing.T) {
	svc, repo, _ := newTestService(t)
	c1, c2 := int64(1), int64(2)
	u := employee(repo, "ivanov", &c1)
	_ = repo.AddMembership(context.Background(), u.ID, c2, 2) // менеджер в c2

	sess, err := svc.Login(context.Background(), dto.LoginRequest{Login: "ivanov", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !sess.NeedsCompanySelection || sess.SelectToken == "" || sess.AccessToken != "" {
		t.Fatalf("ожидался gate выбора компании: %+v", sess)
	}
	if len(sess.Companies) != 2 {
		t.Fatalf("ожидалось 2 компании: %+v", sess.Companies)
	}

	// Завершаем логин выбором c2 → роль менеджера в этой компании.
	full, err := svc.SelectCompany(context.Background(), sess.SelectToken, c2)
	if err != nil {
		t.Fatalf("SelectCompany: %v", err)
	}
	if full.AccessToken == "" || full.CompanyID == nil || *full.CompanyID != c2 || full.RoleLevel != domain.LevelManager {
		t.Fatalf("select c2: %+v", full)
	}
}

func TestSwitchCompanyRescopesRole(t *testing.T) {
	svc, repo, _ := newTestService(t)
	c1, c2 := int64(1), int64(2)
	u := employee(repo, "ivanov", &c1)                        // сотрудник в c1
	_ = repo.AddMembership(context.Background(), u.ID, c2, 3) // руководитель в c2

	sess, err := svc.SwitchCompany(context.Background(), u.ID, c2)
	if err != nil {
		t.Fatalf("SwitchCompany: %v", err)
	}
	if sess.CompanyID == nil || *sess.CompanyID != c2 || sess.RoleLevel != domain.LevelDirector {
		t.Fatalf("switch c2 не дал роль руководителя: %+v", sess)
	}
	// В компанию без членства — отказ.
	if _, err := svc.SwitchCompany(context.Background(), u.ID, int64(99)); err == nil {
		t.Fatal("switch в чужую компанию прошёл")
	}
}

func TestCompanyMembersAdminOnly(t *testing.T) {
	svc, repo, _ := newTestService(t)
	c1, c2 := int64(1), int64(2)
	admin := repo.add(&domain.User{FIO: "Админ", Login: "admin", Role: *repo.roles[4]})
	director := repo.add(&domain.User{
		FIO: "Дир", Login: "dir", Role: *repo.roles[3], CompanyID: &c1, Company: companyOf(&c1),
	})
	outsider := employee(repo, "petrov", &c2) // в другой компании

	// Руководитель НЕ может управлять участниками — только Администратор системы.
	wantCode(t, svc.AddCompanyMember(context.Background(), director, c1, outsider.ID, 1), "FORBIDDEN")
	wantCode(t, svc.RemoveCompanyMember(context.Background(), director, c1, outsider.ID), "FORBIDDEN")

	// Админ добавляет в c1, повышает до Менеджера, затем убирает.
	if err := svc.AddCompanyMember(context.Background(), admin, c1, outsider.ID, 1); err != nil {
		t.Fatalf("AddCompanyMember: %v", err)
	}
	if err := svc.SetMemberRole(context.Background(), admin, c1, outsider.ID, 2); err != nil {
		t.Fatalf("SetMemberRole: %v", err)
	}
	if m, _ := repo.GetMembership(context.Background(), outsider.ID, c1); m == nil || m.Role.Level != domain.LevelManager {
		t.Fatalf("роль в c1 не выставлена: %+v", m)
	}
	// Роль уровня Администратор системы в компании назначать нельзя.
	wantCode(t, svc.SetMemberRole(context.Background(), admin, c1, outsider.ID, 4), "ROLE_LEVEL_FORBIDDEN")

	if err := svc.RemoveCompanyMember(context.Background(), admin, c1, outsider.ID); err != nil {
		t.Fatalf("RemoveCompanyMember: %v", err)
	}
	if m, _ := repo.GetMembership(context.Background(), outsider.ID, c1); m != nil {
		t.Fatal("членство в c1 не удалено")
	}
	if m, _ := repo.GetMembership(context.Background(), outsider.ID, c2); m == nil {
		t.Fatal("членство в c2 потеряно")
	}
}

func TestInviteAndJoin(t *testing.T) {
	svc, repo, _ := newTestService(t)
	admin := repo.add(&domain.User{FIO: "Админ", Login: "admin", Role: *repo.roles[4]})
	company, err := svc.CreateCompany(context.Background(), dto.CompanyCreate{Name: "Acme", IsActive: true})
	if err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	code, err := svc.RegenerateInvite(context.Background(), admin, company.ID)
	if err != nil || code == "" {
		t.Fatalf("RegenerateInvite: code=%q err=%v", code, err)
	}

	other := int64(50)
	u := employee(repo, "petrov", &other) // уже в другой компании

	sess, err := svc.JoinByCode(context.Background(), u.ID, code)
	if err != nil {
		t.Fatalf("JoinByCode: %v", err)
	}
	// Сессия переключена на компанию приглашения, роль — Сотрудник.
	if sess.CompanyID == nil || *sess.CompanyID != company.ID || sess.RoleLevel != domain.LevelEmployee {
		t.Fatalf("join не переключил на компанию: %+v", sess)
	}
	if m, _ := repo.GetMembership(context.Background(), u.ID, company.ID); m == nil {
		t.Fatal("членство по приглашению не создано")
	}
	// Битый код — отказ.
	if _, err := svc.JoinByCode(context.Background(), u.ID, "deadbeef"); err == nil {
		t.Fatal("вступление по неверному коду прошло")
	}
}

func TestCreateCompanyAssignsDirectorMembership(t *testing.T) {
	svc, repo, _ := newTestService(t)
	// Руководитель уже состоит в одной компании.
	existing := int64(99)
	dir := repo.add(&domain.User{
		FIO: "Дир", Login: "dir", Role: *repo.roles[3], CompanyID: &existing, Company: companyOf(&existing),
	})

	// Супер-админ создаёт вторую компанию и ставит его руководителем.
	created, err := svc.CreateCompany(context.Background(), dto.CompanyCreate{
		Name: "Вторая", DirectorID: &dir.ID, IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	ms, _ := repo.ListMemberships(context.Background(), dir.ID)
	if len(ms) != 2 {
		t.Fatalf("ожидалось 2 членства у руководителя двух компаний, получено %d", len(ms))
	}
	m, _ := repo.GetMembership(context.Background(), dir.ID, created.ID)
	if m == nil || m.Role.Level != domain.LevelDirector {
		t.Fatalf("членство Руководителя во второй компании не создано: %+v", m)
	}
}
