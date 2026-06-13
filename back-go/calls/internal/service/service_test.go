package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/ringstate"
)

// ── Фейки портов ─────────────────────────────────────────────────

type fakeRepo struct {
	calls  map[int64]*domain.Call
	parts  map[int64][]*domain.Participant
	nextID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{calls: map[int64]*domain.Call{}, parts: map[int64][]*domain.Participant{}}
}

func (r *fakeRepo) CreateCall(_ context.Context, c *domain.Call, ps []*domain.Participant) error {
	r.nextID++
	c.ID = r.nextID
	c.RoomName = domain.RoomNameFor(c.ID)
	cp := *c
	r.calls[c.ID] = &cp
	for _, p := range ps {
		p.CallID = c.ID
		r.addPart(p)
	}
	return nil
}

func (r *fakeRepo) addPart(p *domain.Participant) {
	r.nextID++
	p.ID = r.nextID
	cp := *p
	r.parts[p.CallID] = append(r.parts[p.CallID], &cp)
}

func (r *fakeRepo) GetCall(_ context.Context, id int64) (*domain.Call, error) {
	c, ok := r.calls[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *fakeRepo) GetCallByShareCode(_ context.Context, code string) (*domain.Call, error) {
	for _, c := range r.calls {
		if c.ShareCode == code {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) UpdateCall(_ context.Context, c *domain.Call) error {
	cp := *c
	r.calls[c.ID] = &cp
	return nil
}

func (r *fakeRepo) DeleteCall(_ context.Context, id int64) error {
	delete(r.calls, id)
	delete(r.parts, id)
	return nil
}

func (r *fakeRepo) GetParticipant(_ context.Context, callID, userID int64) (*domain.Participant, error) {
	for _, p := range r.parts[callID] {
		if p.UserID == userID {
			cp := *p
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) ListParticipants(_ context.Context, callID int64) ([]*domain.Participant, error) {
	out := make([]*domain.Participant, 0, len(r.parts[callID]))
	for _, p := range r.parts[callID] {
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func (r *fakeRepo) CreateParticipant(_ context.Context, p *domain.Participant) error {
	r.addPart(p)
	return nil
}

func (r *fakeRepo) UpdateParticipant(_ context.Context, p *domain.Participant) error {
	for i, old := range r.parts[p.CallID] {
		if old.ID == p.ID {
			cp := *p
			r.parts[p.CallID][i] = &cp
		}
	}
	return nil
}

func (r *fakeRepo) CloseOpenParticipants(_ context.Context, callID int64, leftAt time.Time) error {
	for _, p := range r.parts[callID] {
		if p.LeftAt == nil {
			ts := leftAt
			p.LeftAt = &ts
		}
	}
	return nil
}

func (r *fakeRepo) ListUnfinishedCalls(context.Context) ([]*domain.Call, error) {
	var out []*domain.Call
	for _, c := range r.calls {
		if c.Status == domain.StatusRinging || c.Status == domain.StatusActive {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListHistoryForUser(context.Context, int64, int) ([]*domain.Call, error) {
	return nil, nil
}

type fakeUsers map[int64]*domain.User

func (f fakeUsers) GetUser(_ context.Context, id int64) (*domain.User, error) {
	u, ok := f[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (f fakeUsers) CompanyActive(_ context.Context, _ *int64) (bool, error) { return true, nil }

func (f fakeUsers) ListVisibleUsers(_ context.Context, ids []int64) ([]*domain.User, error) {
	var out []*domain.User
	for _, id := range ids {
		if u, ok := f[id]; ok && !u.IsHidden {
			out = append(out, u)
		}
	}
	return out, nil
}

type fakeMedia struct {
	created, deleted []string
	occupants        map[string][]string // room → identities (nil = недоступен)
	tokenErr         error
}

func (m *fakeMedia) AccessToken(identity, _, room string, _ map[string]any) (string, error) {
	if m.tokenErr != nil {
		return "", m.tokenErr
	}
	return "tok-" + identity + "@" + room, nil
}
func (m *fakeMedia) CreateRoom(_ context.Context, name string, _ int) {
	m.created = append(m.created, name)
}
func (m *fakeMedia) DeleteRoom(_ context.Context, name string) {
	m.deleted = append(m.deleted, name)
}
func (m *fakeMedia) ListParticipantIdentities(_ context.Context, room string) ([]string, bool) {
	ids, ok := m.occupants[room]
	return ids, ok
}
func (m *fakeMedia) ClientURL() string { return "/livekit" }

type pubEvent struct {
	kind   string
	callID int64
	status string
	notify []int64
}

type fakePub struct{ events []pubEvent }

func (p *fakePub) CallEnded(_ context.Context, callID int64, status string, notify []int64) {
	p.events = append(p.events, pubEvent{"call_ended", callID, status, notify})
}
func (p *fakePub) PillCreated(_ context.Context, conversationID, senderID, callID int64) {
	p.events = append(p.events, pubEvent{"pill_created", callID, "", []int64{conversationID, senderID}})
}
func (p *fakePub) PillUpdated(_ context.Context, callID int64) {
	p.events = append(p.events, pubEvent{"pill_updated", callID, "", nil})
}

// fakeMessenger — msgsvc: парный диалог для p2p-звонков.
type fakeMessenger struct {
	nextConvID int64
	calls      [][2]int64
	err        error
}

func (m *fakeMessenger) EnsureDialog(_ context.Context, a, b int64) (int64, error) {
	m.calls = append(m.calls, [2]int64{a, b})
	if m.err != nil {
		return 0, m.err
	}
	if m.nextConvID == 0 {
		m.nextConvID = 700
	}
	return m.nextConvID, nil
}

func company(id int64) *int64 { return &id }

func newTestService() (*Service, *fakeRepo, *fakeMedia, *fakePub) {
	repo := newFakeRepo()
	media := &fakeMedia{occupants: map[string][]string{}}
	pub := &fakePub{}
	users := fakeUsers{
		10: {ID: 10, FIO: "Инициатор", CompanyID: company(1), CompanyActive: true},
		20: {ID: 20, FIO: "Собеседник", CompanyID: company(1), CompanyActive: true},
		30: {ID: 30, FIO: "Третий", CompanyID: company(1), CompanyActive: true},
		99: {ID: 99, FIO: "Чужой", CompanyID: company(2), CompanyActive: true},
	}
	svc := New(repo, users, ringstate.New(), media, pub, &fakeMessenger{}, slog.Default())
	svc.leftGrace = 0 // сверка после participant_left — синхронно
	return svc, repo, media, pub
}

// ── Тесты ────────────────────────────────────────────────────────

func TestStartCallP2P(t *testing.T) {
	svc, repo, media, _ := newTestService()
	resp, err := svc.StartCall(context.Background(), dto.StartCallRequest{
		InitiatorID: 10, InviteeIDs: []int64{20}, Media: "video", ConversationID: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	c := resp.Call
	if c.Kind != domain.KindP2P || c.Status != domain.StatusRinging {
		t.Errorf("kind/status: %s/%s", c.Kind, c.Status)
	}
	if c.ShareCode == nil || *c.ShareCode == "" {
		t.Error("нет share_code")
	}
	if c.ConversationID == nil || *c.ConversationID != 5 {
		t.Error("conversation_id не сохранился")
	}
	if len(c.Participants) != 2 {
		t.Fatalf("участников %d, ожидалось 2", len(c.Participants))
	}
	if resp.Livekit.Token == "" || resp.Livekit.URL != "/livekit" {
		t.Error("нет LiveKit-токена инициатора")
	}
	if len(media.created) != 1 || media.created[0] != domain.RoomNameFor(c.ID) {
		t.Errorf("комната не создана: %v", media.created)
	}
	if !svc.ring.IsUserBusy(10) || !svc.ring.IsUserBusy(20) {
		t.Error("участники должны быть заняты")
	}
	stored, _ := repo.GetCall(context.Background(), c.ID)
	if stored.RoomName != domain.RoomNameFor(c.ID) {
		t.Error("room_name не записан")
	}
}

// p2p без переданного conversation_id: callsvc сам создаёт парный диалог
// через msgsvc и заводит плашку (PillCreated). Недоступный msgsvc звонок
// не блокирует — он пройдёт без привязки к чату.
func TestStartCallEnsuresDialogAndPill(t *testing.T) {
	repo := newFakeRepo()
	media := &fakeMedia{occupants: map[string][]string{}}
	pub := &fakePub{}
	msgr := &fakeMessenger{nextConvID: 700}
	users := fakeUsers{
		10: {ID: 10, FIO: "Инициатор", CompanyID: company(1), CompanyActive: true},
		20: {ID: 20, FIO: "Собеседник", CompanyID: company(1), CompanyActive: true},
	}
	svc := New(repo, users, ringstate.New(), media, pub, msgr, slog.Default())

	resp, err := svc.StartCall(context.Background(), dto.StartCallRequest{
		InitiatorID: 10, InviteeIDs: []int64{20}, Media: "audio",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgr.calls) != 1 || msgr.calls[0] != [2]int64{10, 20} {
		t.Fatalf("EnsureDialog не вызван: %v", msgr.calls)
	}
	if resp.Call.ConversationID == nil || *resp.Call.ConversationID != 700 {
		t.Fatalf("conversation_id = %v", resp.Call.ConversationID)
	}
	var pill *pubEvent
	for i := range pub.events {
		if pub.events[i].kind == "pill_created" {
			pill = &pub.events[i]
		}
	}
	if pill == nil || pill.notify[0] != 700 || pill.notify[1] != 10 {
		t.Fatalf("PillCreated не опубликован: %+v", pub.events)
	}

	// msgsvc лежит — звонок всё равно создаётся, без привязки.
	svc2, _, _, pub2 := newTestService()
	svc2.msgr = &fakeMessenger{err: context.DeadlineExceeded}
	resp2, err := svc2.StartCall(context.Background(), dto.StartCallRequest{
		InitiatorID: 10, InviteeIDs: []int64{20}, Media: "audio",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Call.ConversationID != nil {
		t.Fatal("привязка к диалогу при недоступном msgsvc")
	}
	for _, e := range pub2.events {
		if e.kind == "pill_created" {
			t.Fatal("плашка без диалога")
		}
	}
}

func TestStartCallValidation(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.StartCall(ctx, dto.StartCallRequest{
		InitiatorID: 10, InviteeIDs: []int64{99},
	}); domainCode(err) != "CROSS_COMPANY" {
		t.Errorf("чужая компания: %v", err)
	}

	if _, err := svc.StartCall(ctx, dto.StartCallRequest{InitiatorID: 10, InviteeIDs: []int64{20}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartCall(ctx, dto.StartCallRequest{
		InitiatorID: 30, InviteeIDs: []int64{20},
	}); domainCode(err) != "INVITEE_BUSY" {
		t.Errorf("занятый приглашённый: %v", err)
	}
	if _, err := svc.StartCall(ctx, dto.StartCallRequest{
		InitiatorID: 10, InviteeIDs: []int64{30},
	}); domainCode(err) != "BUSY" {
		t.Errorf("занятый инициатор: %v", err)
	}
}

func TestStartEmptyCall(t *testing.T) {
	// «Пустой звонок»: комната с одним инициатором, без приглашённых —
	// людей зовут уже из звонка (invite / ссылка).
	svc, repo, media, _ := newTestService()
	ctx := context.Background()

	resp, err := svc.StartCall(ctx, dto.StartCallRequest{InitiatorID: 10})
	if err != nil {
		t.Fatal(err)
	}
	c := resp.Call
	if c.Kind != domain.KindGroup || c.Status != domain.StatusRinging {
		t.Errorf("ожидался group+ringing, получено %s/%s", c.Kind, c.Status)
	}
	if len(c.Participants) != 1 || c.Participants[0].UserID != 10 {
		t.Errorf("в звонке должен быть только инициатор: %+v", c.Participants)
	}
	if resp.Livekit.Token == "" {
		t.Error("нет LiveKit-токена инициатора")
	}
	if !svc.ring.IsUserBusy(10) {
		t.Error("инициатор должен быть занят")
	}
	if len(media.created) != 1 {
		t.Error("комната не создана")
	}

	// Пригласить коллегу можно уже из звонка.
	inv, err := svc.InviteToCall(ctx, dto.InviteRequest{
		CallID: c.ID, InviterID: 10, InviteeIDs: []int64{20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.NewInviteeIDs) != 1 || inv.NewInviteeIDs[0] != 20 {
		t.Errorf("приглашение не прошло: %v", inv.NewInviteeIDs)
	}

	// Инициатор положил трубку — сам он освобождается сразу (звонок может
	// ещё ждать приглашённого, его финализируют decline или вебхуки LiveKit).
	if _, err := svc.LeaveCall(ctx, dto.HangupRequest{CallID: c.ID, UserID: 10}); err != nil {
		t.Fatal(err)
	}
	if svc.ring.IsUserBusy(10) {
		t.Error("после выхода инициатор должен освободиться")
	}
	if stored, _ := repo.GetCall(ctx, c.ID); stored == nil {
		t.Error("запись звонка должна остаться в истории")
	}
}

func TestStartCallRollbackOnTokenError(t *testing.T) {
	svc, repo, media, _ := newTestService()
	media.tokenErr = errors.New("sign failed")

	if _, err := svc.StartCall(context.Background(), dto.StartCallRequest{
		InitiatorID: 10, InviteeIDs: []int64{20},
	}); err == nil {
		t.Fatal("ожидалась ошибка выписки токена")
	}
	if svc.ring.IsUserBusy(10) || svc.ring.IsUserBusy(20) {
		t.Error("после отката участники должны освободиться")
	}
	if len(repo.calls) != 0 || len(repo.parts) != 0 {
		t.Error("запись несостоявшегося звонка не удалена")
	}
	if len(media.deleted) != 1 {
		t.Error("комната должна быть погашена")
	}
}

func TestInviteCrossCompany(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := context.Background()
	started, err := svc.StartCall(ctx, dto.StartCallRequest{InitiatorID: 10, InviteeIDs: []int64{20}})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.InviteToCall(ctx, dto.InviteRequest{
		CallID: started.Call.ID, InviterID: 10, InviteeIDs: []int64{99},
	}); domainCode(err) != "CROSS_COMPANY" {
		t.Errorf("приглашение из чужой компании: %v", err)
	}

	resp, err := svc.InviteToCall(ctx, dto.InviteRequest{
		CallID: started.Call.ID, InviterID: 10, InviteeIDs: []int64{30},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.NewInviteeIDs) != 1 || resp.NewInviteeIDs[0] != 30 {
		t.Errorf("свой коллега должен приглашаться: %v", resp.NewInviteeIDs)
	}
}

func TestDeclineP2PBecomesMissed(t *testing.T) {
	svc, repo, media, _ := newTestService()
	ctx := context.Background()
	started, err := svc.StartCall(ctx, dto.StartCallRequest{InitiatorID: 10, InviteeIDs: []int64{20}})
	if err != nil {
		t.Fatal(err)
	}
	callID := started.Call.ID

	resp, err := svc.DeclineCall(ctx, dto.HangupRequest{CallID: callID, UserID: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Ended || resp.Call.Status != domain.StatusMissed {
		t.Errorf("ожидался missed+ended, получено %s ended=%v", resp.Call.Status, resp.Ended)
	}
	// Завершение уведомляет всех участников звонка, включая отклонившего
	// (его другие вкладки тоже должны сбросить состояние).
	if !domain.Has(resp.NotifyUserIDs, 10) || !domain.Has(resp.NotifyUserIDs, 20) {
		t.Errorf("notify = %v, ожидались оба участника [10 20]", resp.NotifyUserIDs)
	}
	if len(media.deleted) != 1 {
		t.Error("комната должна быть погашена")
	}
	if svc.ring.IsUserBusy(10) || svc.ring.IsUserBusy(20) {
		t.Error("все должны освободиться")
	}
	stored, _ := repo.GetCall(ctx, callID)
	if stored.Status != domain.StatusMissed || stored.EndedAt == nil {
		t.Error("в БД звонок не финализирован")
	}
}

func TestWebhookJoinActivatesAndLeftFinalizes(t *testing.T) {
	svc, repo, media, pub := newTestService()
	ctx := context.Background()
	started, _ := svc.StartCall(ctx, dto.StartCallRequest{InitiatorID: 10, InviteeIDs: []int64{20}})
	callID := started.Call.ID
	room := domain.RoomNameFor(callID)

	if _, err := svc.AcceptCall(ctx, dto.AcceptRequest{CallID: callID, UserID: 20}); err != nil {
		t.Fatal(err)
	}
	must(t, svc.HandleWebhook(ctx, dto.WebhookEvent{Event: "participant_joined", Room: room, Identity: "u10"}))
	must(t, svc.HandleWebhook(ctx, dto.WebhookEvent{Event: "participant_joined", Room: room, Identity: "u20"}))

	stored, _ := repo.GetCall(ctx, callID)
	if stored.Status != domain.StatusActive {
		t.Errorf("после двоих в комнате статус %s, ожидался active", stored.Status)
	}

	// Собеседник вышел (в комнате остался один инициатор) → после сверки с
	// фактическим составом LiveKit звонок завершается, событие — клиентам.
	media.occupants[room] = []string{"u10"}
	must(t, svc.HandleWebhook(ctx, dto.WebhookEvent{Event: "participant_left", Room: room, Identity: "u20"}))
	stored, _ = repo.GetCall(ctx, callID)
	if stored.Status != domain.StatusEnded {
		t.Errorf("статус %s, ожидался ended", stored.Status)
	}
	var ended *pubEvent
	for i := range pub.events {
		if pub.events[i].kind == "call_ended" {
			ended = &pub.events[i]
		}
	}
	if ended == nil {
		t.Fatal("событие call_ended не опубликовано")
	}
	if ended.status != domain.StatusEnded || !domain.Has(ended.notify, 10) {
		t.Errorf("call_ended: %+v", *ended)
	}
}

// Перезагрузка страницы / перехват identity второй вкладкой: participant_left
// приходит, но к моменту сверки участник уже снова в комнате — звонок живёт,
// membership сохраняется (нужен для авто-возврата).
func TestWebhookLeftThenReturnedKeepsCall(t *testing.T) {
	svc, repo, media, pub := newTestService()
	ctx := context.Background()
	started, _ := svc.StartCall(ctx, dto.StartCallRequest{InitiatorID: 10, InviteeIDs: []int64{20}})
	callID := started.Call.ID
	room := domain.RoomNameFor(callID)

	if _, err := svc.AcceptCall(ctx, dto.AcceptRequest{CallID: callID, UserID: 20}); err != nil {
		t.Fatal(err)
	}
	must(t, svc.HandleWebhook(ctx, dto.WebhookEvent{Event: "participant_joined", Room: room, Identity: "u10"}))
	must(t, svc.HandleWebhook(ctx, dto.WebhookEvent{Event: "participant_joined", Room: room, Identity: "u20"}))

	// На момент сверки собеседник уже вернулся в комнату.
	media.occupants[room] = []string{"u10", "u20"}
	must(t, svc.HandleWebhook(ctx, dto.WebhookEvent{Event: "participant_left", Room: room, Identity: "u20"}))

	stored, _ := repo.GetCall(ctx, callID)
	if stored.Status != domain.StatusActive || stored.EndedAt != nil {
		t.Errorf("живой звонок финализирован: %s", stored.Status)
	}
	if id, ok := svc.ring.UserActiveCall(20); !ok || id != callID {
		t.Error("участник должен остаться в звонке (для авто-возврата)")
	}
	for _, e := range pub.events {
		if e.kind == "call_ended" {
			t.Fatal("call_ended не должен публиковаться")
		}
	}
}

// Ринг-state потерян (рестарт при упавшем ReconcileStartup) — вебхуки должны
// восстановить его по БД и LiveKit, а не финализировать живой звонок.
func TestWebhookLeftWithoutRingStateKeepsLiveCall(t *testing.T) {
	svc, repo, media, _ := newTestService()
	ctx := context.Background()
	ts := time.Now().UTC()

	call := &domain.Call{InitiatorID: 10, CompanyID: 1, Kind: domain.KindGroup,
		Status: domain.StatusActive, Media: domain.MediaVideo, StartedAt: ts}
	must(t, repo.CreateCall(ctx, call, []*domain.Participant{
		{UserID: 10, Role: domain.RoleInitiator, InvitedAt: ts, JoinedAt: &ts},
		{UserID: 20, Role: domain.RoleInvitee, InvitedAt: ts, JoinedAt: &ts},
		{UserID: 30, Role: domain.RoleInvitee, InvitedAt: ts, JoinedAt: &ts},
	}))
	// Фактический состав комнаты после ухода u30.
	media.occupants[call.RoomName] = []string{"u10", "u20"}

	// Один из троих вышел — звонок продолжается.
	must(t, svc.HandleWebhook(ctx, dto.WebhookEvent{
		Event: "participant_left", Room: call.RoomName, Identity: "u30",
	}))
	stored, _ := repo.GetCall(ctx, call.ID)
	if stored.Status != domain.StatusActive {
		t.Errorf("живой звонок финализирован: %s", stored.Status)
	}
	if got := svc.ring.OccupantsCount(call.ID); got != 2 {
		t.Errorf("после restore occupants = %d, ожидалось 2", got)
	}
}

func TestWebhookJoinWithoutRingStateActivates(t *testing.T) {
	svc, repo, media, _ := newTestService()
	ctx := context.Background()
	ts := time.Now().UTC()

	call := &domain.Call{InitiatorID: 10, CompanyID: 1, Kind: domain.KindP2P,
		Status: domain.StatusRinging, Media: domain.MediaAudio, StartedAt: ts}
	must(t, repo.CreateCall(ctx, call, []*domain.Participant{
		{UserID: 10, Role: domain.RoleInitiator, InvitedAt: ts, JoinedAt: &ts},
		{UserID: 20, Role: domain.RoleInvitee, InvitedAt: ts},
	}))
	media.occupants[call.RoomName] = []string{"u10", "u20"}

	must(t, svc.HandleWebhook(ctx, dto.WebhookEvent{
		Event: "participant_joined", Room: call.RoomName, Identity: "u20",
	}))
	stored, _ := repo.GetCall(ctx, call.ID)
	if stored.Status != domain.StatusActive {
		t.Errorf("статус %s, ожидался active (двое в комнате)", stored.Status)
	}
	if !svc.ring.IsUserBusy(20) {
		t.Error("участник должен быть занят после восстановления state")
	}
}

func TestGuestJoinByCode(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := context.Background()
	started, _ := svc.StartCall(ctx, dto.StartCallRequest{InitiatorID: 10, InviteeIDs: []int64{20}})

	resp, err := svc.JoinByCode(ctx, dto.JoinByCodeRequest{
		Code: *started.Call.ShareCode, GuestName: "  Внешний Гость  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Guest || len(resp.Identity) < 3 || resp.Identity[:2] != "g-" {
		t.Errorf("гостевая identity неверна: %+v", resp)
	}
	if resp.Livekit.Token == "" {
		t.Error("гость не получил токен")
	}
	if _, err := svc.JoinByCode(ctx, dto.JoinByCodeRequest{Code: "несуществующий"}); domainCode(err) != "CALL_NOT_FOUND" {
		t.Errorf("неизвестный код: %v", err)
	}
}

func TestReconcileStartup(t *testing.T) {
	svc, repo, media, _ := newTestService()
	ctx := context.Background()

	// Звонок «завис» в ringing, комнаты в LiveKit нет → missed.
	dead := &domain.Call{InitiatorID: 10, CompanyID: 1, Kind: domain.KindP2P,
		Status: domain.StatusRinging, Media: domain.MediaVideo, StartedAt: time.Now().UTC()}
	must(t, repo.CreateCall(ctx, dead, []*domain.Participant{
		{UserID: 10, Role: domain.RoleInitiator, InvitedAt: time.Now().UTC()},
	}))

	// Живой active-звонок: комната существует, в ней двое → restore.
	alive := &domain.Call{InitiatorID: 20, CompanyID: 1, Kind: domain.KindP2P,
		Status: domain.StatusActive, Media: domain.MediaVideo, StartedAt: time.Now().UTC()}
	must(t, repo.CreateCall(ctx, alive, []*domain.Participant{
		{UserID: 20, Role: domain.RoleInitiator, InvitedAt: time.Now().UTC()},
		{UserID: 30, Role: domain.RoleInvitee, InvitedAt: time.Now().UTC()},
	}))
	media.occupants[alive.RoomName] = []string{"u20", "u30", "g-cafe01"}

	must(t, svc.ReconcileStartup(ctx))

	deadStored, _ := repo.GetCall(ctx, dead.ID)
	if deadStored.Status != domain.StatusMissed || deadStored.EndedAt == nil {
		t.Errorf("мёртвый звонок не финализирован: %+v", deadStored)
	}
	aliveStored, _ := repo.GetCall(ctx, alive.ID)
	if aliveStored.Status != domain.StatusActive {
		t.Error("живой звонок не должен меняться")
	}
	if got := svc.ring.OccupantsCount(alive.ID); got != 3 {
		t.Errorf("restore: occupants = %d, ожидалось 3 (двое + гость)", got)
	}
	if !svc.ring.IsUserBusy(30) {
		t.Error("участник живого звонка должен быть занят после restore")
	}
}

func domainCode(err error) string {
	if de := domain.AsDomainError(err); de != nil {
		return de.Code
	}
	return ""
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
