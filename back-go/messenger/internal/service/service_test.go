package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// ── Фейки портов (без БД/Redis/диска, как в callsvc/authsvc) ─────

type fakeRepo struct {
	convs map[int64]*domain.Conversation
	msgs  map[int64]*domain.Message
	atts  map[int64]*domain.Attachment
	calls map[int64]*domain.CallInfo
	tasks map[int64]*domain.TaskPreview
	pets  map[int64]string
	users *fakeUsers // ФИО для reply/forwarded_from

	nextConv, nextMsg, nextAtt int64
	now                        time.Time
}

func newFakeRepo(users *fakeUsers) *fakeRepo {
	return &fakeRepo{
		convs: map[int64]*domain.Conversation{},
		msgs:  map[int64]*domain.Message{},
		atts:  map[int64]*domain.Attachment{},
		calls: map[int64]*domain.CallInfo{},
		tasks: map[int64]*domain.TaskPreview{},
		pets:  map[int64]string{},
		users: users,
		// Автоответ техподдержки сверяется с реальными часами сервиса —
		// фейковое «сейчас» держим возле time.Now().
		now: time.Now().UTC().Add(-time.Hour),
	}
}

func (r *fakeRepo) tick() time.Time {
	r.now = r.now.Add(time.Second)
	return r.now
}

func (r *fakeRepo) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (r *fakeRepo) GetConversation(_ context.Context, id int64) (*domain.Conversation, error) {
	c, ok := r.convs[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *fakeRepo) GetPair(_ context.Context, a, b int64) (*domain.Conversation, error) {
	for _, c := range r.convs {
		if !c.IsSolo() && c.UserAID == a && c.UserBID != nil && *c.UserBID == b {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) CreatePair(_ context.Context, a, b, companyID int64) (*domain.Conversation, error) {
	r.nextConv++
	c := &domain.Conversation{
		ID: r.nextConv, UserAID: a, UserBID: &b, CompanyID: companyID, CreatedAt: r.tick(),
	}
	r.convs[c.ID] = c
	cp := *c
	return &cp, nil
}

func (r *fakeRepo) GetSolo(_ context.Context, userID int64, pet bool) (*domain.Conversation, error) {
	for _, c := range r.convs {
		if c.UserAID == userID && ((pet && c.IsPetChat) || (!pet && c.IsDevChat)) {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) CreateSolo(_ context.Context, userID, companyID int64, pet bool) (*domain.Conversation, error) {
	r.nextConv++
	c := &domain.Conversation{
		ID: r.nextConv, UserAID: userID, CompanyID: companyID,
		IsDevChat: !pet, IsPetChat: pet, CreatedAt: r.tick(),
	}
	r.convs[c.ID] = c
	cp := *c
	return &cp, nil
}

func (r *fakeRepo) ListPairConversations(_ context.Context, userID int64) ([]*domain.Conversation, error) {
	var out []*domain.Conversation
	for _, c := range r.convs {
		if c.IsSolo() {
			continue
		}
		if c.UserAID == userID && !c.HiddenForA {
			cp := *c
			out = append(out, &cp)
		} else if c.UserBID != nil && *c.UserBID == userID && !c.HiddenForB {
			cp := *c
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		li, lj := out[i].LastMessageAt, out[j].LastMessageAt
		if (li == nil) != (lj == nil) {
			return lj == nil
		}
		if li != nil && lj != nil && !li.Equal(*lj) {
			return li.After(*lj)
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (r *fakeRepo) ListDevChats(_ context.Context) ([]*domain.Conversation, error) {
	var out []*domain.Conversation
	for _, c := range r.convs {
		if c.IsDevChat {
			cp := *c
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *fakeRepo) HideConversation(_ context.Context, id int64, side string) (bool, error) {
	c := r.convs[id]
	if side == domain.SideA {
		c.HiddenForA = true
	} else {
		c.HiddenForB = true
	}
	for _, m := range r.msgs {
		if m.ConversationID == id {
			if side == domain.SideA {
				m.HiddenForA = true
			} else {
				m.HiddenForB = true
			}
		}
	}
	return c.HiddenForA && c.HiddenForB, nil
}

func (r *fakeRepo) DeleteConversation(_ context.Context, id int64) error {
	delete(r.convs, id)
	for mid, m := range r.msgs {
		if m.ConversationID == id {
			for aid, a := range r.atts {
				if a.MessageID != nil && *a.MessageID == mid {
					delete(r.atts, aid)
				}
			}
			delete(r.msgs, mid)
		}
	}
	return nil
}

func (r *fakeRepo) SetConversationPin(_ context.Context, id int64, side string, pinned bool) error {
	c := r.convs[id]
	var at *time.Time
	if pinned {
		t := r.tick()
		at = &t
	}
	if side == domain.SideA {
		c.PinnedAtA = at
	} else {
		c.PinnedAtB = at
	}
	return nil
}

// snapshot — полный снапшот, как msgCols+JOIN'ы в postgres-репозитории.
func (r *fakeRepo) snapshot(m *domain.Message) *domain.Message {
	cp := *m
	conv := r.convs[m.ConversationID]
	if conv != nil {
		cp.ConvIsDevChat = conv.IsDevChat
		cp.ConvOwnerID = conv.UserAID
	}
	cp.Attachments = []domain.Attachment{}
	var attIDs []int64
	for id, a := range r.atts {
		if a.MessageID != nil && *a.MessageID == m.ID {
			attIDs = append(attIDs, id)
		}
	}
	sort.Slice(attIDs, func(i, j int) bool { return attIDs[i] < attIDs[j] })
	for _, id := range attIDs {
		cp.Attachments = append(cp.Attachments, *r.atts[id])
	}
	if m.ReplyToID != nil {
		if t, ok := r.msgs[*m.ReplyToID]; ok {
			rp := &domain.ReplyPreview{ID: t.ID, SenderID: t.SenderID, Text: t.Text, Kind: t.Kind}
			if t.SenderID != nil {
				if u := r.users.users[*t.SenderID]; u != nil {
					fio := u.FIO
					rp.SenderFIO = &fio
				}
			}
			for _, a := range r.atts {
				if a.MessageID != nil && *a.MessageID == t.ID {
					rp.HasAttachments = true
				}
			}
			cp.ReplyTo = rp
		}
	}
	if m.ForwardedFromUserID != nil {
		if u := r.users.users[*m.ForwardedFromUserID]; u != nil {
			cp.ForwardedFrom = &domain.UserRef{ID: u.ID, FIO: u.FIO}
		}
	}
	if m.CallID != nil {
		if c, ok := r.calls[*m.CallID]; ok {
			ci := *c
			cp.Call = &ci
		}
	}
	if m.TaskID != nil {
		if t, ok := r.tasks[*m.TaskID]; ok {
			tp := *t
			cp.Task = &tp
		}
	}
	return &cp
}

func (r *fakeRepo) GetMessage(_ context.Context, id int64) (*domain.Message, error) {
	m, ok := r.msgs[id]
	if !ok {
		return nil, nil
	}
	return r.snapshot(m), nil
}

func hiddenFor(m *domain.Message, side string) bool {
	if side == domain.SideA {
		return m.HiddenForA
	}
	return m.HiddenForB
}

func (r *fakeRepo) convMessages(convID int64) []*domain.Message {
	var out []*domain.Message
	for _, m := range r.msgs {
		if m.ConversationID == convID {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *fakeRepo) ListMessages(_ context.Context, convID int64, side string,
	beforeID, afterID *int64, limit int) ([]*domain.Message, error) {

	var filtered []*domain.Message
	for _, m := range r.convMessages(convID) {
		if hiddenFor(m, side) {
			continue
		}
		if beforeID != nil && m.ID >= *beforeID {
			continue
		}
		if afterID != nil && m.ID <= *afterID {
			continue
		}
		filtered = append(filtered, m)
	}
	if afterID == nil && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	} else if afterID != nil && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	out := make([]*domain.Message, 0, len(filtered))
	for _, m := range filtered {
		out = append(out, r.snapshot(m))
	}
	return out, nil
}

func (r *fakeRepo) ListPinned(_ context.Context, convID int64, side string) ([]*domain.Message, error) {
	var out []*domain.Message
	for _, m := range r.convMessages(convID) {
		if m.PinnedAt != nil && !hiddenFor(m, side) {
			out = append(out, r.snapshot(m))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PinnedAt.After(*out[j].PinnedAt) })
	return out, nil
}

func (r *fakeRepo) LastVisibleMessages(_ context.Context, convIDs []int64, side string) (map[int64]*domain.Message, error) {
	out := map[int64]*domain.Message{}
	for _, convID := range convIDs {
		for _, m := range r.convMessages(convID) {
			if side != "" && hiddenFor(m, side) {
				continue
			}
			out[convID] = r.snapshot(m) // последний по id
		}
	}
	return out, nil
}

// CountUnread — инвариант: (sender IS NULL OR sender != userID), иначе
// бот-сообщения теряются.
func (r *fakeRepo) CountUnread(_ context.Context, convIDs []int64, userID int64, side string) (map[int64]int, error) {
	out := map[int64]int{}
	for _, convID := range convIDs {
		for _, m := range r.convMessages(convID) {
			if m.SenderID != nil && *m.SenderID == userID {
				continue
			}
			if m.ReadAt != nil || (side != "" && hiddenFor(m, side)) {
				continue
			}
			out[convID]++
		}
	}
	return out, nil
}

func (r *fakeRepo) CountUnreadFromSenders(_ context.Context, convIDs, senderIDs []int64) (map[int64]int, error) {
	senders := map[int64]bool{}
	for _, id := range senderIDs {
		senders[id] = true
	}
	out := map[int64]int{}
	for _, convID := range convIDs {
		for _, m := range r.convMessages(convID) {
			if m.SenderID != nil && senders[*m.SenderID] && m.ReadAt == nil {
				out[convID]++
			}
		}
	}
	return out, nil
}

func (r *fakeRepo) TotalUnread(_ context.Context, userID int64) (int, error) {
	n := 0
	for _, c := range r.convs {
		var side string
		switch {
		case c.UserAID == userID && !c.HiddenForA:
			side = domain.SideA
		case c.UserBID != nil && *c.UserBID == userID && !c.HiddenForB:
			side = domain.SideB
		default:
			continue
		}
		for _, m := range r.convMessages(c.ID) {
			if m.SenderID != nil && *m.SenderID == userID {
				continue
			}
			if m.ReadAt == nil && !hiddenFor(m, side) {
				n++
			}
		}
	}
	return n, nil
}

func (r *fakeRepo) CreateMessage(_ context.Context, nm domain.NewMessage) (*domain.Message, error) {
	r.nextMsg++
	kind := nm.Kind
	if kind == "" {
		kind = domain.KindText
	}
	m := &domain.Message{
		ID:                  r.nextMsg,
		ConversationID:      nm.ConversationID,
		SenderID:            nm.SenderID,
		IsBot:               nm.IsBot,
		Text:                nm.Text,
		CreatedAt:           r.tick(),
		ReplyToID:           nm.ReplyToID,
		ForwardedFromUserID: nm.ForwardedFromUserID,
		Kind:                kind,
		TaskID:              nm.TaskID,
		CallID:              nm.CallID,
	}
	r.msgs[m.ID] = m
	if nm.SenderID != nil {
		for _, attID := range nm.AttachmentIDs {
			if a, ok := r.atts[attID]; ok && a.UploaderID == *nm.SenderID && a.MessageID == nil {
				id := m.ID
				a.MessageID = &id
			}
		}
	}
	conv := r.convs[nm.ConversationID]
	conv.LastMessageAt = &m.CreatedAt
	conv.HiddenForA, conv.HiddenForB = false, false
	return r.snapshot(m), nil
}

func (r *fakeRepo) MarkRead(_ context.Context, convID, readerID int64) (int, error) {
	n := 0
	for _, m := range r.convMessages(convID) {
		if m.SenderID != nil && *m.SenderID == readerID {
			continue
		}
		if m.ReadAt == nil {
			t := r.tick()
			m.ReadAt = &t
			n++
		}
	}
	return n, nil
}

func (r *fakeRepo) HideMessage(_ context.Context, id int64, side string) (bool, error) {
	m := r.msgs[id]
	if side == domain.SideA {
		m.HiddenForA = true
	} else {
		m.HiddenForB = true
	}
	return m.HiddenForA && m.HiddenForB, nil
}

func (r *fakeRepo) DeleteMessage(_ context.Context, id int64) error {
	for aid, a := range r.atts {
		if a.MessageID != nil && *a.MessageID == id {
			delete(r.atts, aid)
		}
	}
	delete(r.msgs, id)
	return nil
}

func (r *fakeRepo) RecomputeLastMessageAt(_ context.Context, convID int64) error {
	conv := r.convs[convID]
	conv.LastMessageAt = nil
	for _, m := range r.convMessages(convID) {
		t := m.CreatedAt
		conv.LastMessageAt = &t
	}
	return nil
}

func (r *fakeRepo) SetMessagePin(_ context.Context, id int64, pinned bool, byID *int64) error {
	m := r.msgs[id]
	if pinned {
		t := r.tick()
		m.PinnedAt = &t
		m.PinnedByID = byID
	} else {
		m.PinnedAt = nil
		m.PinnedByID = nil
	}
	return nil
}

func (r *fakeRepo) HasHumanMessageSince(_ context.Context, convID int64, since time.Time, beforeID int64) (bool, error) {
	for _, m := range r.convMessages(convID) {
		if m.ID < beforeID && !m.IsBot && !m.CreatedAt.Before(since) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) ListRecent(_ context.Context, convID int64, limit int) ([]*domain.Message, error) {
	all := r.convMessages(convID)
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	out := make([]*domain.Message, 0, len(all))
	for _, m := range all {
		out = append(out, r.snapshot(m))
	}
	return out, nil
}

func (r *fakeRepo) FindCallMessage(_ context.Context, callID, convID int64) (*domain.Message, error) {
	var found *domain.Message
	for _, m := range r.convMessages(convID) {
		if m.Kind == domain.KindCall && m.CallID != nil && *m.CallID == callID {
			found = m
		}
	}
	if found == nil {
		return nil, nil
	}
	return r.snapshot(found), nil
}

func (r *fakeRepo) ListAttachmentPathsOfConversation(_ context.Context, convID int64) ([]string, error) {
	var out []string
	for _, m := range r.convMessages(convID) {
		for _, a := range r.atts {
			if a.MessageID != nil && *a.MessageID == m.ID {
				out = append(out, a.FilePath)
			}
		}
	}
	return out, nil
}

func (r *fakeRepo) CreateAttachment(_ context.Context, att *domain.Attachment) error {
	r.nextAtt++
	att.ID = r.nextAtt
	att.CreatedAt = r.tick()
	cp := *att
	r.atts[att.ID] = &cp
	return nil
}

func (r *fakeRepo) GetAttachment(_ context.Context, id int64) (*domain.Attachment, error) {
	a, ok := r.atts[id]
	if !ok {
		return nil, nil
	}
	cp := *a
	return &cp, nil
}

func (r *fakeRepo) GetCall(_ context.Context, id int64) (*domain.CallInfo, error) {
	c, ok := r.calls[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *fakeRepo) GetTask(_ context.Context, id int64) (*domain.TaskPreview, error) {
	t, ok := r.tasks[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (r *fakeRepo) PetName(_ context.Context, ownerID int64) (*string, error) {
	name, ok := r.pets[ownerID]
	if !ok {
		return nil, nil
	}
	return &name, nil
}

type fakeUsers struct {
	users map[int64]*domain.User
}

func (f *fakeUsers) GetUser(_ context.Context, id int64) (*domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUsers) CompanyActive(_ context.Context, _ *int64) (bool, error) { return true, nil }

func (f *fakeUsers) ListUsers(_ context.Context, ids []int64) ([]*domain.User, error) {
	var out []*domain.User
	for _, id := range ids {
		if u, ok := f.users[id]; ok {
			cp := *u
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeUsers) DevChatUserIDs(_ context.Context, ownerID int64) ([]int64, error) {
	var out []int64
	for _, u := range f.users {
		if !u.IsHidden && (u.ID == ownerID || u.CompanyID == nil) {
			out = append(out, u.ID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

type fakeFiles struct {
	n       int
	saved   []string
	copied  map[string]string // src → dst
	removed []string
}

func newFakeFiles() *fakeFiles {
	return &fakeFiles{copied: map[string]string{}}
}

func (f *fakeFiles) Save(_ []byte, ext string) (string, error) {
	f.n++
	p := fmt.Sprintf("messages/2026/06/file%d%s", f.n, ext)
	f.saved = append(f.saved, p)
	return p, nil
}

func (f *fakeFiles) Copy(src string) (string, error) {
	f.n++
	p := fmt.Sprintf("messages/2026/06/copy%d", f.n)
	f.copied[src] = p
	return p, nil
}

func (f *fakeFiles) Remove(paths []string) {
	f.removed = append(f.removed, paths...)
}

type pubEvent struct {
	Event   string
	Rooms   []string
	Payload any
}

// noopGroove — pet-хук groovesvc в тестах не дёргаем.
type noopGroove struct{}

func (noopGroove) OnPetMessage(int64) {}

type fakePub struct {
	events []pubEvent
}

func (p *fakePub) Publish(_ context.Context, event string, rooms []string, payload any) {
	p.events = append(p.events, pubEvent{Event: event, Rooms: rooms, Payload: payload})
}

func (p *fakePub) byName(event string) []pubEvent {
	var out []pubEvent
	for _, e := range p.events {
		if e.Event == event {
			out = append(out, e)
		}
	}
	return out
}

// ── Setup ────────────────────────────────────────────────────────

func i64(v int64) *int64   { return &v }
func str(s string) *string { return &s }

func newTestEnv() (*Service, *fakeRepo, *fakeFiles, *fakePub) {
	c10, c20 := int64(10), int64(20)
	users := &fakeUsers{users: map[int64]*domain.User{
		1: {ID: 1, FIO: "Админ Системы", Login: "admin", RoleLevel: 4, CompanyActive: true},
		2: {ID: 2, FIO: "Алиса", Login: "alice", CompanyID: &c10, RoleLevel: 1, CompanyActive: true},
		3: {ID: 3, FIO: "Боб", Login: "bob", CompanyID: &c10, RoleLevel: 1, CompanyActive: true},
		4: {ID: 4, FIO: "Кэрол", Login: "carol", CompanyID: &c20, RoleLevel: 1, CompanyActive: true},
		5: {ID: 5, FIO: "Скрытый", Login: "ghost", CompanyID: &c10, IsHidden: true, CompanyActive: true},
	}}
	repo := newFakeRepo(users)
	files := newFakeFiles()
	pub := &fakePub{}
	svc := New(repo, users, files, pub, noopGroove{}, slog.New(slog.DiscardHandler))
	return svc, repo, files, pub
}

// ── Тесты ────────────────────────────────────────────────────────

// Уникальная пара нормализуется a<b независимо от инициатора.
func TestOpenConversationPairOrder(t *testing.T) {
	svc, repo, _, _ := newTestEnv()
	ctx := context.Background()

	conv1, err := svc.OpenConversation(ctx, 3, 2)
	if err != nil {
		t.Fatalf("open 3→2: %v", err)
	}
	if conv1.UserAID != 2 || conv1.UserBID == nil || *conv1.UserBID != 3 {
		t.Fatalf("пара не нормализована: a=%d b=%v", conv1.UserAID, conv1.UserBID)
	}
	if conv1.CompanyID != 10 {
		t.Fatalf("company_id = %d, ожидалось 10", conv1.CompanyID)
	}
	if conv1.OtherUser == nil || conv1.OtherUser.ID != 2 {
		t.Fatalf("other_user = %+v, ожидался id=2", conv1.OtherUser)
	}

	conv2, err := svc.OpenConversation(ctx, 2, 3)
	if err != nil {
		t.Fatalf("open 2→3: %v", err)
	}
	if conv2.ID != conv1.ID {
		t.Fatalf("создался второй диалог: %d != %d", conv2.ID, conv1.ID)
	}
	if len(repo.convs) != 1 {
		t.Fatalf("диалогов в БД: %d, ожидался 1", len(repo.convs))
	}
}

func TestOpenConversationGuards(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	cases := []struct {
		me, other int64
		code      string
	}{
		{2, 2, "SELF_CONVERSATION"},
		{2, 5, "USER_NOT_FOUND"}, // скрытый
		{2, 99, "USER_NOT_FOUND"},
	}
	for _, tc := range cases {
		_, err := svc.OpenConversation(ctx, tc.me, tc.other)
		de := domain.AsDomainError(err)
		if de == nil || de.Code != tc.code {
			t.Fatalf("open %d→%d: ожидался %s, получено %v", tc.me, tc.other, tc.code, err)
		}
	}
	// Барьер компаний для чата снят: сотрудник может писать в другую компанию
	// (2 — компания 10, 4 — компания 20).
	cross, err := svc.OpenConversation(ctx, 2, 4)
	if err != nil {
		t.Fatalf("кросс-компанийный чат должен проходить: %v", err)
	}
	if cross.CompanyID == 0 {
		t.Fatalf("у кросс-компанийного диалога должен быть company_id: %+v", cross)
	}
	// Администратор системы тоже может писать сотруднику любой компании.
	if _, err := svc.OpenConversation(ctx, 1, 4); err != nil {
		t.Fatalf("админ → сотрудник: %v", err)
	}
}

// Unread-инвариант: бот-сообщения (sender NULL) обязаны попадать в счётчик.
func TestPetChatUnreadCountsBotMessages(t *testing.T) {
	svc, _, _, pub := newTestEnv()
	ctx := context.Background()

	pet, err := svc.OpenPetChat(ctx, 2)
	if err != nil {
		t.Fatalf("open pet chat: %v", err)
	}
	if _, err := svc.PostBotMessage(ctx, pet.ID, "Привет! Я Грувик"); err != nil {
		t.Fatalf("post bot message: %v", err)
	}

	items, err := svc.ListConversations(ctx, 2)
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if len(items) == 0 || !items[0].IsPetChat {
		t.Fatalf("pet-чат не первый в списке: %+v", items)
	}
	if items[0].UnreadCount != 1 {
		t.Fatalf("unread_count = %d, бот-сообщение потерялось", items[0].UnreadCount)
	}
	if items[0].LastMessage == nil || !items[0].LastMessage.IsBot {
		t.Fatalf("last_message не бот: %+v", items[0].LastMessage)
	}

	// message:new ушло владельцу.
	news := pub.byName("message:new")
	if len(news) != 1 || news[0].Rooms[0] != "user_2" {
		t.Fatalf("message:new события: %+v", news)
	}
	ev := news[0].Payload.(dto.MessageNewEvent)
	if ev.FromUserID != nil {
		t.Fatalf("from_user_id у бота должен быть null, получено %v", *ev.FromUserID)
	}

	n, err := svc.MarkRead(ctx, pet.ID, 2)
	if err != nil || n != 1 {
		t.Fatalf("mark read: n=%d err=%v", n, err)
	}
	items, _ = svc.ListConversations(ctx, 2)
	if items[0].UnreadCount != 0 {
		t.Fatalf("после прочтения unread_count = %d", items[0].UnreadCount)
	}
}

// Pet-чат принимает только текст.
func TestPetChatTextOnly(t *testing.T) {
	svc, repo, _, _ := newTestEnv()
	ctx := context.Background()

	pet, _ := svc.OpenPetChat(ctx, 2)
	att, err := svc.UploadAttachment(ctx, 2, "doc.pdf", "application/pdf", []byte("data"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	_, err = svc.SendMessage(ctx, pet.ID, 2, dto.MessageCreate{AttachmentIDs: []int64{att.ID}})
	if de := domain.AsDomainError(err); de == nil || de.Code != "PET_CHAT_TEXT_ONLY" {
		t.Fatalf("вложение в pet-чат: ожидался PET_CHAT_TEXT_ONLY, получено %v", err)
	}

	repo.tasks[7] = &domain.TaskPreview{ID: 7, Name: "Задача", CompanyID: 10}
	_, err = svc.SendMessage(ctx, pet.ID, 2, dto.MessageCreate{TaskID: i64(7)})
	if de := domain.AsDomainError(err); de == nil || de.Code != "PET_CHAT_TEXT_ONLY" {
		t.Fatalf("задача в pet-чат: ожидался PET_CHAT_TEXT_ONLY, получено %v", err)
	}

	if _, err := svc.SendMessage(ctx, pet.ID, 2, dto.MessageCreate{Text: str("привет")}); err != nil {
		t.Fatalf("текст в pet-чат: %v", err)
	}
}

// Soft-delete сообщения обеими сторонами → физическое удаление + файлы.
func TestMessageHiddenByBothSidesIsPhysicallyDeleted(t *testing.T) {
	svc, repo, files, pub := newTestEnv()
	ctx := context.Background()

	conv, _ := svc.OpenConversation(ctx, 2, 3)
	att, _ := svc.UploadAttachment(ctx, 2, "pic.png", "image/png", []byte("img"))
	msg, err := svc.SendMessage(ctx, conv.ID, 2, dto.MessageCreate{
		Text: str("смотри"), AttachmentIDs: []int64{att.ID},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	forAll, err := svc.DeleteMessage(ctx, msg.ID, 2, "me")
	if err != nil || forAll {
		t.Fatalf("скрытие первой стороной: forAll=%v err=%v", forAll, err)
	}
	if _, ok := repo.msgs[msg.ID]; !ok {
		t.Fatal("сообщение удалено раньше времени")
	}

	forAll, err = svc.DeleteMessage(ctx, msg.ID, 3, "me")
	if err != nil || !forAll {
		t.Fatalf("скрытие второй стороной: forAll=%v err=%v", forAll, err)
	}
	if _, ok := repo.msgs[msg.ID]; ok {
		t.Fatal("сообщение не удалено физически")
	}
	if len(files.removed) != 1 || files.removed[0] != att.URL[len("/uploads/"):] {
		t.Fatalf("файлы вложений не зачищены: %v", files.removed)
	}
	if len(pub.byName("message:deleted")) != 1 {
		t.Fatalf("message:deleted не разослан: %+v", pub.events)
	}
}

// scope=all — только своё сообщение.
func TestDeleteMessageForAllOnlyOwn(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	conv, _ := svc.OpenConversation(ctx, 2, 3)
	msg, _ := svc.SendMessage(ctx, conv.ID, 2, dto.MessageCreate{Text: str("моё")})

	_, err := svc.DeleteMessage(ctx, msg.ID, 3, "all")
	if de := domain.AsDomainError(err); de == nil || de.Code != "FORBIDDEN" {
		t.Fatalf("чужое сообщение scope=all: ожидался FORBIDDEN, получено %v", err)
	}
	forAll, err := svc.DeleteMessage(ctx, msg.ID, 2, "all")
	if err != nil || !forAll {
		t.Fatalf("своё сообщение scope=all: forAll=%v err=%v", forAll, err)
	}
}

// Soft-delete диалога обеими сторонами → физическое удаление + файлы.
func TestConversationHiddenByBothSidesIsPhysicallyDeleted(t *testing.T) {
	svc, repo, files, pub := newTestEnv()
	ctx := context.Background()

	conv, _ := svc.OpenConversation(ctx, 2, 3)
	att, _ := svc.UploadAttachment(ctx, 3, "v.mp4", "video/mp4", []byte("vid"))
	if _, err := svc.SendMessage(ctx, conv.ID, 3, dto.MessageCreate{AttachmentIDs: []int64{att.ID}}); err != nil {
		t.Fatalf("send: %v", err)
	}

	physical, err := svc.DeleteConversation(ctx, conv.ID, 2, "me")
	if err != nil || physical {
		t.Fatalf("первая сторона: physical=%v err=%v", physical, err)
	}
	physical, err = svc.DeleteConversation(ctx, conv.ID, 3, "me")
	if err != nil || !physical {
		t.Fatalf("вторая сторона: physical=%v err=%v", physical, err)
	}
	if _, ok := repo.convs[conv.ID]; ok {
		t.Fatal("диалог не удалён физически")
	}
	if len(files.removed) != 1 {
		t.Fatalf("файлы не зачищены: %v", files.removed)
	}
	// Второй удаливший получает эхо в свои вкладки.
	evs := pub.byName("conversation:deleted")
	if len(evs) != 1 || len(evs[0].Rooms) != 1 || evs[0].Rooms[0] != "user_3" {
		t.Fatalf("conversation:deleted: %+v", evs)
	}
}

// Чат техподдержки удалять нельзя.
func TestDevChatUndeletable(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	dev, _ := svc.OpenDevChat(ctx, 2)
	_, err := svc.DeleteConversation(ctx, dev.ID, 2, "all")
	if de := domain.AsDomainError(err); de == nil || de.Code != "DEV_CHAT_UNDELETABLE" {
		t.Fatalf("dev-чат: %v", err)
	}
}

// Чат с Грувиком — соло-диалог: удаляется физически при любом scope,
// владелец получает conversation:deleted в свои вкладки.
func TestPetChatDeletesPhysically(t *testing.T) {
	svc, repo, _, pub := newTestEnv()
	ctx := context.Background()

	pet, _ := svc.OpenPetChat(ctx, 2)
	if _, err := svc.SendMessage(ctx, pet.ID, 2, dto.MessageCreate{Text: str("привет")}); err != nil {
		t.Fatalf("send: %v", err)
	}

	physical, err := svc.DeleteConversation(ctx, pet.ID, 2, "me")
	if err != nil || !physical {
		t.Fatalf("pet-чат: physical=%v err=%v", physical, err)
	}
	if _, ok := repo.convs[pet.ID]; ok {
		t.Fatal("pet-чат не удалён физически")
	}
	evs := pub.byName("conversation:deleted")
	if len(evs) != 1 || len(evs[0].Rooms) != 1 || evs[0].Rooms[0] != "user_2" {
		t.Fatalf("conversation:deleted: %+v", evs)
	}
}

// Пересылка: соло-чаты пропускаются, вложения копируются физически.
func TestForwardSkipsSoloAndCopiesAttachments(t *testing.T) {
	svc, repo, files, _ := newTestEnv()
	ctx := context.Background()

	dev, _ := svc.OpenDevChat(ctx, 2)
	src, _ := svc.OpenConversation(ctx, 2, 3)
	att, _ := svc.UploadAttachment(ctx, 3, "report.pdf", "application/pdf", []byte("pdf"))
	msg, err := svc.SendMessage(ctx, src.ID, 3, dto.MessageCreate{
		Text: str("отчёт"), AttachmentIDs: []int64{att.ID},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// Только соло-цель → NO_TARGET.
	_, err = svc.ForwardMessage(ctx, 2, msg.ID, []int64{dev.ID}, nil)
	if de := domain.AsDomainError(err); de == nil || de.Code != "NO_TARGET" {
		t.Fatalf("пересылка только в dev-чат: ожидался NO_TARGET, получено %v", err)
	}

	// Dev-чат пропущен, пользователь 1 (админ) получает копию.
	results, err := svc.ForwardMessage(ctx, 2, msg.ID, []int64{dev.ID}, []int64{1})
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("целей: %d, ожидалась 1", len(results))
	}
	fwd := results[0].Message
	if fwd.ForwardedFrom == nil || fwd.ForwardedFrom.ID != 3 {
		t.Fatalf("forwarded_from = %+v, ожидался автор оригинала id=3", fwd.ForwardedFrom)
	}
	if len(fwd.Attachments) != 1 || fwd.Attachments[0].ID == att.ID {
		t.Fatalf("вложение не скопировано отдельной записью: %+v", fwd.Attachments)
	}
	srcPath := att.URL[len("/uploads/"):]
	if _, ok := files.copied[srcPath]; !ok {
		t.Fatalf("файл не скопирован физически: %v", files.copied)
	}
	// Оригинал не задет.
	if orig, _ := repo.GetMessage(ctx, msg.ID); len(orig.Attachments) != 1 || orig.Attachments[0].ID != att.ID {
		t.Fatalf("оригинальное вложение пострадало: %+v", orig.Attachments)
	}
}

// Автоответ техподдержки — один раз в сутки, бот с kind=system_dev_reply.
func TestSupportAutoReplyOncePerDay(t *testing.T) {
	svc, repo, _, pub := newTestEnv()
	ctx := context.Background()

	dev, _ := svc.OpenDevChat(ctx, 2)

	countBots := func() int {
		n := 0
		for _, m := range repo.msgs {
			if m.ConversationID == dev.ID && m.IsBot {
				n++
			}
		}
		return n
	}

	if _, err := svc.SendMessage(ctx, dev.ID, 2, dto.MessageCreate{Text: str("помогите")}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if countBots() != 1 {
		t.Fatalf("автоответ не создан: ботов %d", countBots())
	}
	for _, m := range repo.msgs {
		if m.IsBot {
			if m.Kind != domain.KindDevReply || m.SenderID != nil || *m.Text != SupportAutoReplyText {
				t.Fatalf("неверная форма автоответа: %+v", m)
			}
		}
	}
	// message:new (своё + автоответ) уходят владельцу и админу.
	news := pub.byName("message:new")
	if len(news) != 2 {
		t.Fatalf("message:new событий: %d, ожидалось 2", len(news))
	}
	for _, ev := range news {
		if len(ev.Rooms) != 2 || ev.Rooms[0] != "user_1" || ev.Rooms[1] != "user_2" {
			t.Fatalf("rooms dev-чата: %v", ev.Rooms)
		}
	}
	if ev := news[1].Payload.(dto.MessageNewEvent); ev.FromUserID != nil {
		t.Fatalf("from_user_id автоответа должен быть null")
	}

	// Повторное сообщение в те же сутки — без автоответа.
	if _, err := svc.SendMessage(ctx, dev.ID, 2, dto.MessageCreate{Text: str("ещё вопрос")}); err != nil {
		t.Fatalf("send #2: %v", err)
	}
	if countBots() != 1 {
		t.Fatalf("автоответ продублировался: ботов %d", countBots())
	}

	// Сутки тишины — бот отвечает снова.
	for _, m := range repo.msgs {
		m.CreatedAt = m.CreatedAt.Add(-25 * time.Hour)
	}
	if _, err := svc.SendMessage(ctx, dev.ID, 2, dto.MessageCreate{Text: str("я снова тут")}); err != nil {
		t.Fatalf("send #3: %v", err)
	}
	if countBots() != 2 {
		t.Fatalf("автоответ после суток не создан: ботов %d", countBots())
	}

	// Ответ админа автоответа не вызывает, но получает spec-kind.
	adminMsg, err := svc.SendMessage(ctx, dev.ID, 1, dto.MessageCreate{Text: str("чиним")})
	if err != nil {
		t.Fatalf("send admin: %v", err)
	}
	if adminMsg.Kind != domain.KindDevReply || !adminMsg.IsFromSupport {
		t.Fatalf("ответ админа: kind=%s is_from_support=%v", adminMsg.Kind, adminMsg.IsFromSupport)
	}
	if countBots() != 2 {
		t.Fatalf("ответ админа вызвал автоответ: ботов %d", countBots())
	}
}

// Правила закрепления: чужие kind'ы нельзя, событие уходит обоим участникам.
func TestMessagePinRules(t *testing.T) {
	svc, repo, _, pub := newTestEnv()
	ctx := context.Background()

	conv, _ := svc.OpenConversation(ctx, 2, 3)
	msg, _ := svc.SendMessage(ctx, conv.ID, 2, dto.MessageCreate{Text: str("важно")})

	pinnedMsg, pinned, err := svc.ToggleMessagePin(ctx, msg.ID, 3)
	if err != nil || !pinned {
		t.Fatalf("pin: pinned=%v err=%v", pinned, err)
	}
	if pinnedMsg.PinnedAt == nil || pinnedMsg.PinnedByID == nil || *pinnedMsg.PinnedByID != 3 {
		t.Fatalf("снапшот закрепления: %+v", pinnedMsg)
	}
	evs := pub.byName("message:pin")
	if len(evs) != 1 || len(evs[0].Rooms) != 2 {
		t.Fatalf("message:pin: %+v", evs)
	}

	_, pinned, err = svc.ToggleMessagePin(ctx, msg.ID, 2)
	if err != nil || pinned {
		t.Fatalf("unpin: pinned=%v err=%v", pinned, err)
	}

	// Плашку задачи закрепить нельзя.
	repo.tasks[7] = &domain.TaskPreview{ID: 7, Name: "Задача", CompanyID: 10}
	taskMsg, _ := svc.SendMessage(ctx, conv.ID, 2, dto.MessageCreate{TaskID: i64(7)})
	if taskMsg.Kind != domain.KindTask {
		t.Fatalf("kind = %s, ожидался task", taskMsg.Kind)
	}
	_, _, err = svc.ToggleMessagePin(ctx, taskMsg.ID, 2)
	if de := domain.AsDomainError(err); de == nil || de.Code != "BAD_PIN" {
		t.Fatalf("pin задачи: ожидался BAD_PIN, получено %v", err)
	}
}

// Личное закрепление диалога + сортировка списка (pinned первыми).
func TestConversationPinPersonal(t *testing.T) {
	svc, _, _, pub := newTestEnv()
	ctx := context.Background()

	convA, _ := svc.OpenConversation(ctx, 2, 3)
	convB, _ := svc.OpenConversation(ctx, 2, 1)
	// Свежее сообщение поднимает convB наверх…
	if _, err := svc.SendMessage(ctx, convB.ID, 1, dto.MessageCreate{Text: str("привет")}); err != nil {
		t.Fatalf("send: %v", err)
	}

	pinned, err := svc.ToggleConversationPin(ctx, convA.ID, 2)
	if err != nil || !pinned {
		t.Fatalf("pin: %v", err)
	}
	evs := pub.byName("conversation:pin")
	if len(evs) != 1 || evs[0].Rooms[0] != "user_2" {
		t.Fatalf("conversation:pin: %+v", evs)
	}

	// …но закреплённый convA всё равно первый среди пар (выше только
	// автосозданный dev-чат, он всегда сверху).
	items, _ := svc.ListConversations(ctx, 2)
	var pairs []*dto.ConversationListItem
	for _, it := range items {
		if !it.IsDevChat && !it.IsPetChat {
			pairs = append(pairs, it)
		}
	}
	if len(pairs) != 2 || pairs[0].ID != convA.ID || !pairs[0].IsPinned {
		t.Fatalf("закреплённый диалог не первый: %+v", pairs)
	}
	// А у пользователя 3 закрепление не видно (оно личное).
	items3, _ := svc.ListConversations(ctx, 3)
	for _, it := range items3 {
		if it.ID == convA.ID && it.IsPinned {
			t.Fatal("закрепление утекло собеседнику")
		}
	}

	if pinned, _ = svc.ToggleConversationPin(ctx, convA.ID, 2); pinned {
		t.Fatal("повторный toggle не снял закрепление")
	}
}

// gRPC-плашки звонков: создание и актуальный снапшот со статусом из calls.
func TestCallMessageLifecycle(t *testing.T) {
	svc, repo, _, _ := newTestEnv()
	ctx := context.Background()

	convID, err := svc.EnsureDialog(ctx, 2, 3)
	if err != nil {
		t.Fatalf("ensure dialog: %v", err)
	}
	started := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	repo.calls[42] = &domain.CallInfo{
		ID: 42, Kind: "p2p", Media: "video", Status: "ringing",
		StartedAt: started, InitiatorID: 2, ConversationID: &convID,
	}

	msg, notify, err := svc.CreateCallMessage(ctx, convID, 2, 42)
	if err != nil {
		t.Fatalf("create call message: %v", err)
	}
	if msg.Kind != domain.KindCall || msg.Call == nil || msg.Call.Status != "ringing" {
		t.Fatalf("плашка: %+v", msg)
	}
	if len(notify) != 2 {
		t.Fatalf("notify: %v", notify)
	}

	// Звонок завершился — снапшот читает статус заново и считает длительность.
	ended := started.Add(90 * time.Second)
	repo.calls[42].Status = "ended"
	repo.calls[42].EndedAt = &ended

	gotConvID, updated, notify2, err := svc.GetCallMessage(ctx, 42)
	if err != nil || gotConvID != convID || len(notify2) != 2 {
		t.Fatalf("get call message: conv=%d notify=%v err=%v", gotConvID, notify2, err)
	}
	if updated.Call.Status != "ended" || updated.Call.DurationSec == nil || *updated.Call.DurationSec != 90 {
		t.Fatalf("обновлённая плашка: %+v", updated.Call)
	}
}

// ListRecentMessages — хронологический порядок и лимит.
func TestListRecentMessages(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	pet, _ := svc.OpenPetChat(ctx, 2)
	for i := 0; i < 5; i++ {
		if _, err := svc.SendMessage(ctx, pet.ID, 2, dto.MessageCreate{Text: str(fmt.Sprintf("м%d", i))}); err != nil {
			t.Fatalf("send: %v", err)
		}
	}
	msgs, err := svc.ListRecentMessages(ctx, pet.ID, 3)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(msgs) != 3 || *msgs[0].Text != "м2" || *msgs[2].Text != "м4" {
		t.Fatalf("порядок/лимит: %+v", msgs)
	}
}

// Невидимые мелочи отправки: пустые сообщения, чужие вложения, чужие ответы.
func TestSendMessageValidation(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	conv, _ := svc.OpenConversation(ctx, 2, 3)

	_, err := svc.SendMessage(ctx, conv.ID, 2, dto.MessageCreate{Text: str("   ")})
	if de := domain.AsDomainError(err); de == nil || de.Code != "EMPTY_MESSAGE" {
		t.Fatalf("пробельный текст: ожидался EMPTY_MESSAGE, получено %v", err)
	}

	// Чужое вложение использовать нельзя.
	att, _ := svc.UploadAttachment(ctx, 3, "x.txt", "text/plain", []byte("x"))
	_, err = svc.SendMessage(ctx, conv.ID, 2, dto.MessageCreate{AttachmentIDs: []int64{att.ID}})
	if de := domain.AsDomainError(err); de == nil || de.Code != "BAD_ATTACHMENT" {
		t.Fatalf("чужое вложение: ожидался BAD_ATTACHMENT, получено %v", err)
	}

	// Ответ на сообщение чужого диалога запрещён.
	other, _ := svc.OpenConversation(ctx, 1, 3)
	foreign, _ := svc.SendMessage(ctx, other.ID, 1, dto.MessageCreate{Text: str("из другого чата")})
	_, err = svc.SendMessage(ctx, conv.ID, 2, dto.MessageCreate{Text: str("re"), ReplyToID: &foreign.ID})
	if de := domain.AsDomainError(err); de == nil || de.Code != "BAD_REPLY" {
		t.Fatalf("ответ в чужой диалог: ожидался BAD_REPLY, получено %v", err)
	}

	// Доступ к чужому диалогу закрыт.
	_, err = svc.SendMessage(ctx, other.ID, 2, dto.MessageCreate{Text: str("я тут лишний")})
	if de := domain.AsDomainError(err); de == nil || de.Code != "FORBIDDEN" {
		t.Fatalf("не участник: ожидался FORBIDDEN, получено %v", err)
	}

	// Задача чужой компании к сообщению не прикрепляется.
	svcRepoTask(t, svc, ctx, conv.ID)
}

func svcRepoTask(t *testing.T, svc *Service, ctx context.Context, convID int64) {
	t.Helper()
	repo := svc.repo.(*fakeRepo)
	repo.tasks[8] = &domain.TaskPreview{ID: 8, Name: "Чужая", CompanyID: 20}
	_, err := svc.SendMessage(ctx, convID, 2, dto.MessageCreate{TaskID: i64(8)})
	if de := domain.AsDomainError(err); de == nil || de.Code != "TASK_WRONG_COMPANY" {
		t.Fatalf("чужая задача: ожидался TASK_WRONG_COMPANY, получено %v", err)
	}
}

// Support-inbox доступен только Администратору системы и считает только
// сообщения владельцев.
func TestSupportInbox(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	dev, _ := svc.OpenDevChat(ctx, 2)
	if _, err := svc.SendMessage(ctx, dev.ID, 2, dto.MessageCreate{Text: str("вопрос")}); err != nil {
		t.Fatalf("send: %v", err)
	}

	_, err := svc.SupportInbox(ctx, 2)
	if de := domain.AsDomainError(err); de == nil || de.Code != "FORBIDDEN" {
		t.Fatalf("инбокс не для сотрудника: %v", err)
	}

	items, err := svc.SupportInbox(ctx, 1)
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	if len(items) != 1 || items[0].ID != dev.ID {
		t.Fatalf("items: %+v", items)
	}
	// Сообщение владельца — 1; автоответ бота в счётчик НЕ входит.
	if items[0].UnreadCount != 1 {
		t.Fatalf("unread_count = %d, ожидался 1 (только владелец)", items[0].UnreadCount)
	}
	if items[0].OwnerUser == nil || items[0].OwnerUser.ID != 2 {
		t.Fatalf("owner_user: %+v", items[0].OwnerUser)
	}
}
