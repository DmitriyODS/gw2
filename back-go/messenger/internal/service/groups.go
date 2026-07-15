package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// MaxGroupTitle — предел длины названия группы (совпадает с колонкой БД).
const MaxGroupTitle = 120

var mentionRe = regexp.MustCompile(`@([A-Za-z0-9_.\-]{2,})`)

// ── Создание и чтение группы ─────────────────────────────────────

// CreateGroup — новая группа: создатель — владелец, memberIDs — участники.
// avatarAttID — id уже загруженного через /uploads вложения-картинки (его путь
// становится аватаром группы). Первое системное сообщение фиксирует создание.
func (s *Service) CreateGroup(ctx context.Context, creatorID int64, title string,
	avatarAttID *int64, memberIDs []int64) (*dto.Conversation, error) {

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, domain.NewError("EMPTY_TITLE", "Введите название группы", 422)
	}
	title = truncateString(title, MaxGroupTitle)
	avatarPath, err := s.avatarPathFromAttachment(ctx, avatarAttID, creatorID)
	if err != nil {
		return nil, err
	}

	conv, err := s.repo.CreateGroup(ctx, title, avatarPath, creatorID, memberIDs)
	if err != nil {
		return nil, err
	}
	s.log.Info("group.create", "conversation_id", conv.ID, "creator_id", creatorID)

	s.postGroupSystem(ctx, conv, creatorID, s.userName(ctx, creatorID)+" создал(а) группу")
	// Все участники должны увидеть группу в списке — просим их клиенты подтянуть.
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, conv.ID, ids)
	return s.GetGroup(ctx, conv.ID, creatorID)
}

// GetGroup — карточка группы: участники + проекция зрителя. invite_code виден
// только тем, кто может управлять участниками.
func (s *Service) GetGroup(ctx context.Context, convID, userID int64) (*dto.Conversation, error) {
	conv, err := s.conversationForUser(ctx, convID, userID)
	if err != nil {
		return nil, err
	}
	if !conv.IsGroup {
		return nil, errConvNotFound()
	}
	members, err := s.repo.ListMembers(ctx, convID)
	if err != nil {
		return nil, err
	}
	conv.Members = members
	conv.MemberCount = len(members)
	me := findMember(members, userID)
	if me != nil {
		conv.MyRole = me.Role
		conv.MyMuted = me.Muted
		conv.MyPinnedAt = me.PinnedAt
	}
	if me == nil || !me.CanManage("members") {
		conv.InviteCode = nil // ссылку видят только управляющие участниками
	}
	return dto.NewConversation(conv), nil
}

// ── Управление участниками ───────────────────────────────────────

func (s *Service) AddGroupMembers(ctx context.Context, convID, actorID int64, userIDs []int64) error {
	conv, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return err
	}
	if !actor.CanManage("members") {
		return errNoRights()
	}
	added := make([]int64, 0, len(userIDs))
	for _, uid := range userIDs {
		if uid == 0 {
			continue
		}
		if m, err := s.repo.GetMember(ctx, convID, uid); err != nil {
			return err
		} else if m != nil {
			continue // уже участник
		}
		if err := s.repo.AddMember(ctx, convID, uid, domain.RoleMember); err != nil {
			return err
		}
		added = append(added, uid)
	}
	if len(added) == 0 {
		return nil
	}
	s.postGroupSystem(ctx, conv, actorID,
		s.userName(ctx, actorID)+" добавил(а): "+s.userNames(ctx, added))
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, convID, append(ids, added...))
	return nil
}

func (s *Service) RemoveGroupMember(ctx context.Context, convID, actorID, memberID int64) error {
	conv, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return err
	}
	target, err := s.repo.GetMember(ctx, convID, memberID)
	if err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	// Владельца удалить нельзя; админа может удалить только владелец; обычного —
	// владелец или админ с правом «участники».
	if target.Role == domain.RoleOwner ||
		(target.Role == domain.RoleAdmin && actor.Role != domain.RoleOwner) ||
		!actor.CanManage("members") {
		return errNoRights()
	}
	if err := s.repo.RemoveMember(ctx, convID, memberID); err != nil {
		return err
	}
	s.postGroupSystem(ctx, conv, actorID,
		s.userName(ctx, actorID)+" удалил(а) "+s.userName(ctx, memberID))
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, convID, ids)
	// Удалённому — убрать группу из списка.
	s.pub.Publish(ctx, "conversation:deleted", rooms(memberID),
		dto.ConversationDeletedEvent{ConversationID: convID})
	return nil
}

// LeaveGroup — выход из группы. Владелец передаёт владение самому раннему
// участнику; если он был последним — группа удаляется.
func (s *Service) LeaveGroup(ctx context.Context, convID, userID int64) error {
	conv, me, err := s.groupActor(ctx, convID, userID)
	if err != nil {
		return err
	}
	members, err := s.repo.ListMembers(ctx, convID)
	if err != nil {
		return err
	}
	if len(members) <= 1 {
		if err := s.destroyConversation(ctx, convID); err != nil {
			return err
		}
		s.pub.Publish(ctx, "conversation:deleted", rooms(userID),
			dto.ConversationDeletedEvent{ConversationID: convID})
		return nil
	}
	if err := s.repo.RemoveMember(ctx, convID, userID); err != nil {
		return err
	}
	// Владелец ушёл — передаём владение самому раннему по joined_at из оставшихся.
	if me.Role == domain.RoleOwner {
		var heir *domain.Member
		for _, m := range members {
			if m.UserID == userID {
				continue
			}
			if heir == nil || m.JoinedAt.Before(heir.JoinedAt) {
				heir = m
			}
		}
		if heir != nil {
			if err := s.repo.UpdateMemberRole(ctx, convID, heir.UserID, domain.RoleOwner); err != nil {
				return err
			}
		}
	}
	s.postGroupSystem(ctx, conv, userID, s.userName(ctx, userID)+" вышел(а) из группы")
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, convID, ids)
	s.pub.Publish(ctx, "conversation:deleted", rooms(userID),
		dto.ConversationDeletedEvent{ConversationID: convID})
	return nil
}

// ── Информация о группе ──────────────────────────────────────────

func (s *Service) RenameGroup(ctx context.Context, convID, actorID int64, title string) error {
	conv, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return err
	}
	if !actor.CanManage("info") {
		return errNoRights()
	}
	title = truncateString(strings.TrimSpace(title), MaxGroupTitle)
	if title == "" {
		return domain.NewError("EMPTY_TITLE", "Введите название группы", 422)
	}
	if err := s.repo.RenameGroup(ctx, convID, title); err != nil {
		return err
	}
	conv.Title = &title
	s.postGroupSystem(ctx, conv, actorID,
		s.userName(ctx, actorID)+" переименовал(а) группу в «"+title+"»")
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, convID, ids)
	return nil
}

func (s *Service) SetGroupAvatar(ctx context.Context, convID, actorID int64, avatarAttID *int64) error {
	conv, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return err
	}
	if !actor.CanManage("info") {
		return errNoRights()
	}
	path, err := s.avatarPathFromAttachment(ctx, avatarAttID, actorID)
	if err != nil {
		return err
	}
	if err := s.repo.SetGroupAvatar(ctx, convID, path); err != nil {
		return err
	}
	s.postGroupSystem(ctx, conv, actorID, s.userName(ctx, actorID)+" обновил(а) аватар группы")
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, convID, ids)
	return nil
}

// ── Роли и права ─────────────────────────────────────────────────

func (s *Service) SetMemberRole(ctx context.Context, convID, actorID, memberID int64, role string) error {
	_, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return err
	}
	if actor.Role != domain.RoleOwner {
		return errNoRights() // менять роли может только владелец
	}
	if role != domain.RoleAdmin && role != domain.RoleMember {
		return domain.NewError("BAD_ROLE", "Недопустимая роль", 400)
	}
	target, err := s.repo.GetMember(ctx, convID, memberID)
	if err != nil {
		return err
	}
	if target == nil || target.Role == domain.RoleOwner {
		return errNoRights()
	}
	if err := s.repo.UpdateMemberRole(ctx, convID, memberID, role); err != nil {
		return err
	}
	conv, _ := s.repo.GetConversation(ctx, convID)
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, convID, ids)
	return nil
}

func (s *Service) SetMemberRights(ctx context.Context, convID, actorID, memberID int64,
	manageMembers, editInfo, pinMessages bool) error {

	_, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return err
	}
	if actor.Role != domain.RoleOwner {
		return errNoRights()
	}
	target, err := s.repo.GetMember(ctx, convID, memberID)
	if err != nil {
		return err
	}
	if target == nil || target.Role != domain.RoleAdmin {
		return domain.NewError("NOT_ADMIN", "Права настраиваются только для админов", 400)
	}
	if err := s.repo.UpdateMemberRights(ctx, convID, memberID, manageMembers, editInfo, pinMessages); err != nil {
		return err
	}
	conv, _ := s.repo.GetConversation(ctx, convID)
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, convID, ids)
	return nil
}

// TransferOwnership — передать владение другому участнику: он становится
// владельцем, прежний владелец — админом. Только текущий владелец.
func (s *Service) TransferOwnership(ctx context.Context, convID, actorID, newOwnerID int64) error {
	conv, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return err
	}
	if actor.Role != domain.RoleOwner {
		return errNoRights()
	}
	if newOwnerID == actorID {
		return nil // уже владелец
	}
	target, err := s.repo.GetMember(ctx, convID, newOwnerID)
	if err != nil {
		return err
	}
	if target == nil {
		return errNoAccess()
	}
	if err := s.repo.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateMemberRole(ctx, convID, newOwnerID, domain.RoleOwner); err != nil {
			return err
		}
		return s.repo.UpdateMemberRole(ctx, convID, actorID, domain.RoleAdmin)
	}); err != nil {
		return err
	}
	s.postGroupSystem(ctx, conv, actorID,
		s.userName(ctx, actorID)+" передал(а) владение: "+s.userName(ctx, newOwnerID))
	ids, _ := s.audience(ctx, conv)
	s.emitGroupUpdated(ctx, convID, ids)
	return nil
}

// SetGroupMute — личное отключение уведомлений группы; возвращает новое значение.
func (s *Service) SetGroupMute(ctx context.Context, convID, userID int64, muted bool) (bool, error) {
	if _, _, err := s.groupActor(ctx, convID, userID); err != nil {
		return false, err
	}
	if err := s.repo.SetMemberMute(ctx, convID, userID, muted); err != nil {
		return false, err
	}
	s.pub.Publish(ctx, "group:updated", rooms(userID),
		dto.GroupUpdatedEvent{ConversationID: convID})
	return muted, nil
}

// ── Ссылка-приглашение ───────────────────────────────────────────

func (s *Service) GroupInviteLink(ctx context.Context, convID, actorID int64) (string, error) {
	conv, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return "", err
	}
	if !actor.CanManage("members") {
		return "", errNoRights()
	}
	if conv.InviteCode != nil && *conv.InviteCode != "" {
		return *conv.InviteCode, nil
	}
	code, err := records.NewShareCode()
	if err != nil {
		return "", err
	}
	if err := s.repo.SetInviteCode(ctx, convID, &code); err != nil {
		return "", err
	}
	return code, nil
}

func (s *Service) RevokeGroupInviteLink(ctx context.Context, convID, actorID int64) error {
	_, actor, err := s.groupActor(ctx, convID, actorID)
	if err != nil {
		return err
	}
	if !actor.CanManage("members") {
		return errNoRights()
	}
	return s.repo.SetInviteCode(ctx, convID, nil)
}

// GroupInvitePreview — превью группы по коду приглашения (название/аватар/счётчик)
// для экрана вступления. Требует авторизации, но не членства.
func (s *Service) GroupInvitePreview(ctx context.Context, code string) (*dto.Conversation, error) {
	conv, err := s.repo.FindByInviteCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, domain.NewError("INVITE_NOT_FOUND", "Ссылка недействительна", 404)
	}
	members, err := s.repo.ListMembers(ctx, conv.ID)
	if err != nil {
		return nil, err
	}
	conv.MemberCount = len(members)
	conv.InviteCode = nil // в превью код не отдаём
	return dto.NewConversation(conv), nil
}

func (s *Service) JoinGroupByCode(ctx context.Context, code string, userID int64) (*dto.Conversation, error) {
	conv, err := s.repo.FindByInviteCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, domain.NewError("INVITE_NOT_FOUND", "Ссылка недействительна", 404)
	}
	existing, err := s.repo.GetMember(ctx, conv.ID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		if err := s.repo.AddMember(ctx, conv.ID, userID, domain.RoleMember); err != nil {
			return nil, err
		}
		s.postGroupSystem(ctx, conv, userID,
			s.userName(ctx, userID)+" присоединился(ась) по ссылке")
		ids, _ := s.audience(ctx, conv)
		s.emitGroupUpdated(ctx, conv.ID, ids)
	}
	return s.GetGroup(ctx, conv.ID, userID)
}

// ── «Кто прочитал» ───────────────────────────────────────────────

func (s *Service) ReadBy(ctx context.Context, messageID, userID int64) ([]*dto.DirectoryUser, error) {
	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, errMsgNotFound()
	}
	conv, err := s.repo.GetConversation(ctx, msg.ConversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil || !conv.IsGroup {
		return nil, errConvNotFound()
	}
	if err := s.ensureMember(ctx, conv, userID); err != nil {
		return nil, err
	}
	var authorID int64
	if msg.SenderID != nil {
		authorID = *msg.SenderID
	}
	readers, err := s.repo.ReadersOf(ctx, conv.ID, messageID, authorID)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.DirectoryUser, 0, len(readers))
	for _, m := range readers {
		out = append(out, dto.NewDirectoryUser(m.User))
	}
	return out, nil
}

// ── Хелперы групп ────────────────────────────────────────────────

// broadcastGroupMessage — веер сообщения по участникам: WS всем, пуш — не-muted
// (плюс упомянутым @login, даже если у них mute).
func (s *Service) broadcastGroupMessage(ctx context.Context, conv *domain.Conversation,
	senderID int64, msg *domain.Message, event *dto.MessageNewEvent) error {

	mutes, err := s.repo.ListMemberMutes(ctx, conv.ID)
	if err != nil {
		return err
	}
	memberIDs := make([]int64, 0, len(mutes))
	for id := range mutes {
		memberIDs = append(memberIDs, id)
	}
	mentioned := map[int64]bool{}
	if msg.Text != nil && strings.Contains(*msg.Text, "@") {
		if ids, err := s.mentionedMemberIDs(ctx, conv.ID, *msg.Text); err == nil {
			for _, id := range ids {
				mentioned[id] = true
			}
		}
	}
	notify := make([]int64, 0, len(memberIDs))
	for id, muted := range mutes {
		if id == senderID {
			continue
		}
		if !muted || mentioned[id] {
			notify = append(notify, id)
		}
	}
	event.NotifyIDs = notify
	event.ConversationTitle = conv.Title
	s.pub.Publish(ctx, "message:new", rooms(memberIDs...), *event)
	return nil
}

// markGroupRead — поднять watermark участника до последнего сообщения группы.
func (s *Service) markGroupRead(ctx context.Context, conv *domain.Conversation, userID int64) (int, error) {
	last, err := s.repo.LastVisibleMessages(ctx, []int64{conv.ID}, "")
	if err != nil {
		return 0, err
	}
	lastMsg := last[conv.ID]
	if lastMsg == nil {
		return 0, nil
	}
	cnt, err := s.repo.CountGroupUnread(ctx, []int64{conv.ID}, userID)
	if err != nil {
		return 0, err
	}
	changed, err := s.repo.SetMemberRead(ctx, conv.ID, userID, lastMsg.ID)
	if err != nil {
		return 0, err
	}
	if changed {
		ids, err := s.audience(ctx, conv)
		if err != nil {
			return 0, err
		}
		lid := lastMsg.ID
		s.pub.Publish(ctx, "message:read", rooms(ids...), dto.MessageReadEvent{
			ConversationID: conv.ID, ReaderID: userID, LastReadID: &lid,
		})
	}
	return cnt[conv.ID], nil
}

// mentionedMemberIDs — id участников, упомянутых в тексте через @login.
func (s *Service) mentionedMemberIDs(ctx context.Context, convID int64, text string) ([]int64, error) {
	matches := mentionRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	logins := map[string]bool{}
	for _, m := range matches {
		logins[strings.ToLower(m[1])] = true
	}
	members, err := s.repo.ListMembers(ctx, convID)
	if err != nil {
		return nil, err
	}
	var out []int64
	for _, m := range members {
		if m.User != nil && logins[strings.ToLower(m.User.Login)] {
			out = append(out, m.UserID)
		}
	}
	return out, nil
}

// postGroupSystem — служебная плашка группы (создан/добавлен/вышел/…).
func (s *Service) postGroupSystem(ctx context.Context, conv *domain.Conversation, actorID int64, text string) {
	msg, err := s.repo.CreateMessage(ctx, domain.NewMessage{
		ConversationID: conv.ID,
		SenderID:       &actorID,
		Text:           &text,
		Kind:           domain.KindSystem,
	})
	if err != nil {
		s.log.Warn("group.system_message_failed", "conversation_id", conv.ID, "error", err)
		return
	}
	ids, err := s.audience(ctx, conv)
	if err != nil {
		return
	}
	s.pub.Publish(ctx, "message:new", rooms(ids...), dto.MessageNewEvent{
		ConversationID: conv.ID, Message: dto.NewMessage(msg), FromUserID: &actorID,
	})
}

// emitGroupUpdated — попросить клиенты перечитать группу (состав/инфо/роли).
func (s *Service) emitGroupUpdated(ctx context.Context, convID int64, targetIDs []int64) {
	if len(targetIDs) == 0 {
		return
	}
	s.pub.Publish(ctx, "group:updated", rooms(targetIDs...),
		dto.GroupUpdatedEvent{ConversationID: convID})
}

// groupActor — группа + строка участника-инициатора (с проверкой членства).
func (s *Service) groupActor(ctx context.Context, convID, userID int64) (*domain.Conversation, *domain.Member, error) {
	conv, err := s.repo.GetConversation(ctx, convID)
	if err != nil {
		return nil, nil, err
	}
	if conv == nil || !conv.IsGroup {
		return nil, nil, errConvNotFound()
	}
	mem, err := s.repo.GetMember(ctx, convID, userID)
	if err != nil {
		return nil, nil, err
	}
	if mem == nil {
		return nil, nil, errNoAccess()
	}
	return conv, mem, nil
}

func (s *Service) avatarPathFromAttachment(ctx context.Context, attID *int64, uploaderID int64) (*string, error) {
	if attID == nil {
		return nil, nil
	}
	att, err := s.repo.GetAttachment(ctx, *attID)
	if err != nil {
		return nil, err
	}
	if att == nil || att.UploaderID != uploaderID {
		return nil, domain.NewError("BAD_ATTACHMENT", "Недопустимое вложение", 400)
	}
	p := att.FilePath
	return &p, nil
}

func (s *Service) userName(ctx context.Context, id int64) string {
	u, err := s.users.GetUser(ctx, id)
	if err != nil || u == nil {
		return "Пользователь"
	}
	return u.FIO
}

func (s *Service) userNames(ctx context.Context, ids []int64) string {
	users, err := s.users.ListUsers(ctx, ids)
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(users))
	for _, u := range users {
		names = append(names, u.FIO)
	}
	return strings.Join(names, ", ")
}

func errNoRights() *domain.Error {
	return domain.NewError("FORBIDDEN", "Недостаточно прав", 403)
}

func findMember(members []*domain.Member, userID int64) *domain.Member {
	for _, m := range members {
		if m.UserID == userID {
			return m
		}
	}
	return nil
}
