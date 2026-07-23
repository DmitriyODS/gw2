package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// Создание группы: владелец + участники, системная плашка, веер сообщения по
// участникам, notify_ids без автора.
func TestCreateGroupAndBroadcast(t *testing.T) {
	svc, _, _, pub := newTestEnv()
	ctx := context.Background()

	g, err := svc.CreateGroup(ctx, 2, "Команда", nil, []int64{3})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if !g.IsGroup || g.MyRole != domain.RoleOwner || g.MemberCount != 2 {
		t.Fatalf("группа неверна: %+v", g)
	}
	// Системная плашка «создал группу».
	if news := pub.byName("message:new"); len(news) == 0 {
		t.Fatal("нет события создания (системная плашка)")
	}

	msg, err := svc.SendMessage(ctx, g.ID, 2, dto.MessageCreate{Text: str("привет")})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	_ = msg
	news := pub.byName("message:new")
	last := news[len(news)-1]
	ev := last.Payload.(dto.MessageNewEvent)
	// Веер — по всем участникам (комнаты user_2, user_3).
	if len(last.Rooms) != 2 {
		t.Fatalf("веер по %d комнатам, ожидалось 2: %v", len(last.Rooms), last.Rooms)
	}
	// notify_ids — без автора (только Боб=3).
	if len(ev.NotifyIDs) != 1 || ev.NotifyIDs[0] != 3 {
		t.Fatalf("notify_ids = %v, ожидалось [3]", ev.NotifyIDs)
	}
	if ev.ConversationTitle == nil || *ev.ConversationTitle != "Команда" {
		t.Fatalf("conversation_title = %v", ev.ConversationTitle)
	}
}

// Пересылка сообщения В ГРУППУ: у группы нет «второго» участника
// (OtherUserID == nil), поэтому веер должен идти всей аудитории через
// broadcastGroupMessage — раньше здесь был nil-разыменование и паника (500).
func TestForwardIntoGroupFansOut(t *testing.T) {
	svc, _, _, pub := newTestEnv()
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, 2, "Команда", nil, []int64{3})
	src, _ := svc.OpenConversation(ctx, 2, 4)
	msg, err := svc.SendMessage(ctx, src.ID, 2, dto.MessageCreate{Text: str("к пересылке")})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	results, err := svc.ForwardMessage(ctx, 2, msg.ID, []int64{g.ID}, nil)
	if err != nil {
		t.Fatalf("forward в группу: %v", err)
	}
	if len(results) != 1 || results[0].ConversationID != g.ID {
		t.Fatalf("цель пересылки неверна: %+v", results)
	}
	// Последнее message:new — веер по участникам группы (user_2, user_3)
	// с заголовком и notify_ids (Боб=3, без автора).
	news := pub.byName("message:new")
	last := news[len(news)-1]
	ev := last.Payload.(dto.MessageNewEvent)
	if last.Rooms == nil || len(last.Rooms) != 2 {
		t.Fatalf("веер группы по %d комнатам, ожидалось 2: %v", len(last.Rooms), last.Rooms)
	}
	if ev.ConversationTitle == nil || *ev.ConversationTitle != "Команда" {
		t.Fatalf("нет заголовка группы в событии пересылки: %v", ev.ConversationTitle)
	}
	if len(ev.NotifyIDs) != 1 || ev.NotifyIDs[0] != 3 {
		t.Fatalf("notify_ids пересылки = %v, ожидалось [3]", ev.NotifyIDs)
	}
}

// Права: обычный участник не может переименовать/добавлять; владелец может;
// повышение до админа даёт право управлять участниками.
func TestGroupPermissions(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, 2, "Команда", nil, []int64{3})

	if err := svc.RenameGroup(ctx, g.ID, 3, "Хак"); err == nil {
		t.Fatal("Боб (участник) не должен переименовывать группу")
	}
	if err := svc.RenameGroup(ctx, g.ID, 2, "Проект"); err != nil {
		t.Fatalf("владелец не смог переименовать: %v", err)
	}
	if err := svc.AddGroupMembers(ctx, g.ID, 3, []int64{4}); err == nil {
		t.Fatal("участник не должен добавлять")
	}
	// Повышаем Боба до админа — теперь может добавлять.
	if err := svc.SetMemberRole(ctx, g.ID, 2, 3, domain.RoleAdmin); err != nil {
		t.Fatalf("promote: %v", err)
	}
	if err := svc.AddGroupMembers(ctx, g.ID, 3, []int64{4}); err != nil {
		t.Fatalf("админ не смог добавить: %v", err)
	}
	got, _ := svc.GetGroup(ctx, g.ID, 2)
	if got.MemberCount != 3 {
		t.Fatalf("участников %d, ожидалось 3", got.MemberCount)
	}
}

// Прочтение группы: watermark участника + «кто прочитал».
func TestGroupReadWatermarkAndReadBy(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, 2, "Команда", nil, []int64{3})
	msg, _ := svc.SendMessage(ctx, g.ID, 2, dto.MessageCreate{Text: str("привет")})

	// До прочтения Бобом — читателей нет.
	readers, _ := svc.ReadBy(ctx, msg.ID, 2)
	if len(readers) != 0 {
		t.Fatalf("читателей %d, ожидалось 0", len(readers))
	}
	// Боб читает.
	if _, err := svc.MarkRead(ctx, g.ID, 3); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	readers, _ = svc.ReadBy(ctx, msg.ID, 2)
	if len(readers) != 1 || readers[0].ID != 3 {
		t.Fatalf("readby = %+v, ожидался [Боб]", readers)
	}
	// У Боба непрочитанных больше нет (своё сообщение автора не считается).
	items, _ := svc.ListConversations(ctx, 3, nil)
	for _, it := range items {
		if it.ID == g.ID && it.UnreadCount != 0 {
			t.Fatalf("unread у Боба = %d, ожидалось 0", it.UnreadCount)
		}
	}
}

// Явная передача владения: новый владелец — назначенный, прежний — админ.
func TestTransferOwnership(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, 2, "Команда", nil, []int64{3})
	if err := svc.TransferOwnership(ctx, g.ID, 2, 3); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	// Не владелец передать не может.
	if err := svc.TransferOwnership(ctx, g.ID, 2, 3); err == nil {
		t.Fatal("бывший владелец (теперь админ) не должен передавать владение")
	}
	got, _ := svc.GetGroup(ctx, g.ID, 3)
	if got.MyRole != domain.RoleOwner {
		t.Fatalf("новый владелец роль=%s, ожидался owner", got.MyRole)
	}
	old, _ := svc.GetGroup(ctx, g.ID, 2)
	if old.MyRole != domain.RoleAdmin {
		t.Fatalf("прежний владелец роль=%s, ожидался admin", old.MyRole)
	}
}

// Выход владельца передаёт владение самому раннему участнику.
func TestLeaveGroupTransfersOwnership(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, 2, "Команда", nil, []int64{3})
	if err := svc.LeaveGroup(ctx, g.ID, 2); err != nil {
		t.Fatalf("leave: %v", err)
	}
	// Боб теперь владелец.
	got, err := svc.GetGroup(ctx, g.ID, 3)
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	if got.MyRole != domain.RoleOwner || got.MemberCount != 1 {
		t.Fatalf("владение не передано: role=%s count=%d", got.MyRole, got.MemberCount)
	}
	// Алиса больше не участник.
	if _, err := svc.GetGroup(ctx, g.ID, 2); err == nil {
		t.Fatal("Алиса всё ещё видит группу после выхода")
	}
}

// Ссылка-приглашение: генерация и вступление по коду.
func TestGroupInviteLinkJoin(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, 2, "Команда", nil, nil)
	code, err := svc.GroupInviteLink(ctx, g.ID, 2)
	if err != nil || code == "" {
		t.Fatalf("invite link: %v code=%q", err, code)
	}
	// Кэрол вступает по коду.
	joined, err := svc.JoinGroupByCode(ctx, code, 4)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if joined.MyRole != domain.RoleMember || joined.MemberCount != 2 {
		t.Fatalf("вступление неверно: role=%s count=%d", joined.MyRole, joined.MemberCount)
	}
	// Повторное вступление не плодит участника.
	joined2, _ := svc.JoinGroupByCode(ctx, code, 4)
	if joined2.MemberCount != 2 {
		t.Fatalf("повторное вступление изменило состав: %d", joined2.MemberCount)
	}
}
