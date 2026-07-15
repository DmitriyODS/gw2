package service

import (
	"context"
	"sort"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// ListConversations — список диалогов пользователя: закреплённые → остальные
// (по last_message_at), сверху pet-чат и личный dev-чат (для пользователей
// с активной компанией; без активной компании своих соло-чатов нет).
// companyID — активная компания сессии ИЗ ТОКЕНА: в users её нет (идентичность
// развязана с компаниями), поэтому передаётся транспортом.
func (s *Service) ListConversations(ctx context.Context, userID int64, companyID *int64) ([]*dto.ConversationListItem, error) {
	me, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Личный чат техподдержки должен существовать у сотрудника компании
	// всегда, даже без переписки. Бизнес-блокировку (доменную ошибку)
	// глотаем — не валим листинг.
	if me != nil && companyID != nil {
		if _, err := s.getOrCreateSolo(ctx, userID, *companyID); err != nil {
			if domain.AsDomainError(err) == nil {
				return nil, err
			}
		}
	}

	convs, err := s.repo.ListPairConversations(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Закреплённые первыми (pinned_at DESC); не закреплённые сохраняют
	// SQL-порядок (стабильная сортировка, как в Python).
	sort.SliceStable(convs, func(i, j int) bool {
		pi, pj := convs[i].PinnedAtFor(userID), convs[j].PinnedAtFor(userID)
		if (pi == nil) != (pj == nil) {
			return pj == nil
		}
		if pi != nil && pj != nil {
			return pi.After(*pj)
		}
		return false
	})

	var aIDs, bIDs, otherIDs []int64
	for _, c := range convs {
		if c.Side(userID) == domain.SideA {
			aIDs = append(aIDs, c.ID)
		} else {
			bIDs = append(bIDs, c.ID)
		}
		if oid := c.OtherUserID(userID); oid != nil {
			otherIDs = append(otherIDs, *oid)
		}
	}

	lastByConv := map[int64]*domain.Message{}
	unreadByConv := map[int64]int{}
	for side, ids := range map[string][]int64{domain.SideA: aIDs, domain.SideB: bIDs} {
		if len(ids) == 0 {
			continue
		}
		last, err := s.repo.LastVisibleMessages(ctx, ids, side)
		if err != nil {
			return nil, err
		}
		for k, v := range last {
			lastByConv[k] = v
		}
		unread, err := s.repo.CountUnread(ctx, ids, userID, side)
		if err != nil {
			return nil, err
		}
		for k, v := range unread {
			unreadByConv[k] = v
		}
	}

	others, err := s.users.ListUsers(ctx, otherIDs)
	if err != nil {
		return nil, err
	}
	otherByID := make(map[int64]*domain.User, len(others))
	for _, u := range others {
		otherByID[u.ID] = u
	}

	result := make([]*dto.ConversationListItem, 0, len(convs)+2)
	for _, c := range convs {
		var other *domain.User
		if oid := c.OtherUserID(userID); oid != nil {
			other = otherByID[*oid]
		}
		pinnedAt := c.PinnedAtFor(userID)
		result = append(result, &dto.ConversationListItem{
			ID:            c.ID,
			OtherUser:     dto.NewDirectoryUser(other),
			LastMessage:   dto.NewMessage(lastByConv[c.ID]),
			UnreadCount:   unreadByConv[c.ID],
			LastMessageAt: dto.JSONTimePtr(c.LastMessageAt),
			IsPinned:      pinnedAt != nil,
			PinnedAt:      dto.JSONTimePtr(pinnedAt),
			CompanyID:     c.CompanyID,
			CompanyName:   c.CompanyName,
		})
	}

	// Группы пользователя — свой источник (conversation_members), досыпаем
	// в общий список (порядок клиент пересобирает сам).
	groups, err := s.groupListItems(ctx, userID)
	if err != nil {
		return nil, err
	}
	result = append(result, groups...)

	// Личный dev-чат владельца исключён из ListPairConversations — досыпаем
	// его отдельно первым, если уже создан.
	dev, err := s.soloListItem(ctx, userID)
	if err != nil {
		return nil, err
	}
	if dev != nil {
		result = append([]*dto.ConversationListItem{dev}, result...)
	}
	return result, nil
}

// groupListItems — элементы списка для групп пользователя.
func (s *Service) groupListItems(ctx context.Context, userID int64) ([]*dto.ConversationListItem, error) {
	convs, err := s.repo.ListGroupConversations(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(convs) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(convs))
	for _, c := range convs {
		ids = append(ids, c.ID)
	}
	lastByConv, err := s.repo.LastVisibleMessages(ctx, ids, "")
	if err != nil {
		return nil, err
	}
	unreadByConv, err := s.repo.CountGroupUnread(ctx, ids, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.ConversationListItem, 0, len(convs))
	for _, c := range convs {
		out = append(out, &dto.ConversationListItem{
			ID:            c.ID,
			LastMessage:   dto.NewMessage(lastByConv[c.ID]),
			UnreadCount:   unreadByConv[c.ID],
			LastMessageAt: dto.JSONTimePtr(c.LastMessageAt),
			IsPinned:      c.MyPinnedAt != nil,
			PinnedAt:      dto.JSONTimePtr(c.MyPinnedAt),
			IsGroup:       true,
			Title:         c.Title,
			AvatarPath:    c.AvatarPath,
			MemberCount:   c.MemberCount,
			MyRole:        c.MyRole,
			Muted:         c.MyMuted,
		})
	}
	return out, nil
}

// soloListItem — элемент списка для dev-чата владельца; nil — чата нет.
func (s *Service) soloListItem(ctx context.Context, userID int64) (*dto.ConversationListItem, error) {
	conv, err := s.repo.GetSolo(ctx, userID)
	if err != nil || conv == nil {
		return nil, err
	}
	last, err := s.repo.LastVisibleMessages(ctx, []int64{conv.ID}, "")
	if err != nil {
		return nil, err
	}
	unread, err := s.repo.CountUnread(ctx, []int64{conv.ID}, userID, "")
	if err != nil {
		return nil, err
	}
	return &dto.ConversationListItem{
		ID:            conv.ID,
		LastMessage:   dto.NewMessage(last[conv.ID]),
		UnreadCount:   unread[conv.ID],
		LastMessageAt: dto.JSONTimePtr(conv.LastMessageAt),
		IsDevChat:     conv.IsDevChat,
		CompanyID:     conv.CompanyID,
		CompanyName:   conv.CompanyName,
	}, nil
}

// OpenConversation — найти или создать диалог с пользователем.
func (s *Service) OpenConversation(ctx context.Context, meID, otherUserID int64) (*dto.ConversationWithOther, error) {
	conv, err := s.ensureConversation(ctx, meID, otherUserID)
	if err != nil {
		return nil, err
	}
	var other *domain.User
	if oid := conv.OtherUserID(meID); oid != nil {
		if other, err = s.users.GetUser(ctx, *oid); err != nil {
			return nil, err
		}
	}
	return &dto.ConversationWithOther{
		Conversation: *dto.NewConversation(conv),
		OtherUser:    dto.NewDirectoryUser(other),
	}, nil
}

// getOrCreateSolo — dev-чат владельца (get_or_create_dev_chat_for_user).
func (s *Service) getOrCreateSolo(ctx context.Context, userID, companyID int64) (*domain.Conversation, error) {
	conv, err := s.repo.GetSolo(ctx, userID)
	if err != nil || conv != nil {
		return conv, err
	}
	return s.repo.CreateSolo(ctx, userID, companyID)
}

// OpenDevChat — личный чат с техподдержкой (нужна активная компания).
// companyID — активная компания из токена (в users её нет — идентичность
// развязана с компаниями; брать из me.CompanyID нельзя).
func (s *Service) OpenDevChat(ctx context.Context, userID int64, companyID *int64) (*dto.Conversation, error) {
	me, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if me == nil {
		return nil, domain.NewError("USER_NOT_FOUND", "Пользователь не найден", 404)
	}
	if companyID == nil {
		return nil, domain.NewError("NO_ACTIVE_COMPANY",
			"Нет активной компании для чата с техподдержкой", 400)
	}
	conv, err := s.getOrCreateSolo(ctx, userID, *companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewConversation(conv), nil
}

// SupportInbox — все личные dev-чаты пользователей (вкладка «Техподдержка»
// супер-админа).
func (s *Service) SupportInbox(ctx context.Context, userID int64) ([]*dto.ConversationListItem, error) {
	me, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if me == nil || !me.IsSuperAdmin {
		return nil, domain.NewError("FORBIDDEN", "Только супер-админ", 403)
	}

	convs, err := s.repo.ListDevChats(ctx)
	if err != nil {
		return nil, err
	}
	convIDs := make([]int64, 0, len(convs))
	ownerIDs := make([]int64, 0, len(convs))
	for _, c := range convs {
		convIDs = append(convIDs, c.ID)
		ownerIDs = append(ownerIDs, c.UserAID)
	}

	lastByConv := map[int64]*domain.Message{}
	unreadByConv := map[int64]int{}
	if len(convIDs) > 0 {
		if lastByConv, err = s.repo.LastVisibleMessages(ctx, convIDs, ""); err != nil {
			return nil, err
		}
		// Непрочитанные — только сообщения владельцев (ни бот, ни админы).
		if unreadByConv, err = s.repo.CountUnreadFromSenders(ctx, convIDs, ownerIDs); err != nil {
			return nil, err
		}
	}

	owners, err := s.users.ListUsers(ctx, ownerIDs)
	if err != nil {
		return nil, err
	}
	ownerByID := make(map[int64]*domain.User, len(owners))
	for _, u := range owners {
		ownerByID[u.ID] = u
	}

	items := make([]*dto.ConversationListItem, 0, len(convs))
	for _, c := range convs {
		items = append(items, &dto.ConversationListItem{
			ID:            c.ID,
			OwnerUser:     dto.NewDirectoryUser(ownerByID[c.UserAID]),
			LastMessage:   dto.NewMessage(lastByConv[c.ID]),
			UnreadCount:   unreadByConv[c.ID],
			LastMessageAt: dto.JSONTimePtr(c.LastMessageAt),
			IsDevChat:     true,
			CompanyID:     c.CompanyID,
			CompanyName:   c.CompanyName,
		})
	}
	return items, nil
}

// DeleteConversation — scope=me (скрыть у себя) или all (у обоих).
// Возвращает true, если диалог удалён физически.
func (s *Service) DeleteConversation(ctx context.Context, convID, userID int64, scope string) (bool, error) {
	conv, err := s.conversationForUser(ctx, convID, userID)
	if err != nil {
		return false, err
	}
	// Чат техподдержки живёт сколько живёт владелец — удалять нельзя.
	if conv.IsDevChat {
		return false, domain.NewError("DEV_CHAT_UNDELETABLE", "Чат техподдержки удалить нельзя", 400)
	}

	// Группа: «у себя» — скрыть (hidden_at); «у всех» — только владелец
	// распускает группу целиком (выход участника — отдельный LeaveGroup).
	if conv.IsGroup {
		if scope == "all" {
			mem, err := s.repo.GetMember(ctx, convID, userID)
			if err != nil {
				return false, err
			}
			if mem == nil || mem.Role != domain.RoleOwner {
				return false, errNoRights()
			}
			ids, _ := s.audience(ctx, conv)
			if err := s.destroyConversation(ctx, convID); err != nil {
				return false, err
			}
			s.pub.Publish(ctx, "conversation:deleted", rooms(ids...),
				dto.ConversationDeletedEvent{ConversationID: convID})
			return true, nil
		}
		if err := s.repo.HideConversationMember(ctx, convID, userID, true); err != nil {
			return false, err
		}
		s.pub.Publish(ctx, "conversation:deleted", rooms(userID),
			dto.ConversationDeletedEvent{ConversationID: convID})
		return false, nil
	}

	otherID := conv.OtherUserID(userID)

	var physical bool
	switch scope {
	case "all":
		if err := s.destroyConversation(ctx, convID); err != nil {
			return false, err
		}
		physical = true
	case "me":
		both, err := s.repo.HideConversation(ctx, convID, conv.Side(userID))
		if err != nil {
			return false, err
		}
		if both {
			if err := s.destroyConversation(ctx, convID); err != nil {
				return false, err
			}
			physical = true
		}
	default:
		return false, domain.NewError("BAD_SCOPE", "Неверный scope", 400)
	}

	s.log.Info("conversation.delete", "conversation_id", convID,
		"user_id", userID, "scope", scope, "physical", physical)

	payload := dto.ConversationDeletedEvent{ConversationID: convID}
	if scope == "all" && otherID != nil {
		s.pub.Publish(ctx, "conversation:deleted", rooms(*otherID, userID), payload)
	} else if physical && otherID != nil {
		// Обе стороны независимо нажали «у себя» — уведомим другие вкладки
		// самого пользователя (собеседнику уже не нужно).
		s.pub.Publish(ctx, "conversation:deleted", rooms(userID), payload)
	}
	return physical, nil
}

// destroyConversation — физическое удаление диалога с файлами вложений.
func (s *Service) destroyConversation(ctx context.Context, convID int64) error {
	paths, err := s.repo.ListAttachmentPathsOfConversation(ctx, convID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteConversation(ctx, convID); err != nil {
		return err
	}
	s.files.Remove(paths)
	return nil
}

// ToggleConversationPin — личное закрепление диалога; возвращает новое
// состояние.
func (s *Service) ToggleConversationPin(ctx context.Context, convID, userID int64) (bool, error) {
	conv, err := s.conversationForUser(ctx, convID, userID)
	if err != nil {
		return false, err
	}
	var pinned bool
	if conv.IsGroup {
		// GetConversation не заполняет проекцию зрителя — берём pin из member.
		mem, err := s.repo.GetMember(ctx, convID, userID)
		if err != nil {
			return false, err
		}
		pinned = mem.PinnedAt == nil
		if err := s.repo.SetMemberPin(ctx, convID, userID, pinned); err != nil {
			return false, err
		}
	} else {
		pinned = conv.PinnedAtFor(userID) == nil
		if err := s.repo.SetConversationPin(ctx, convID, conv.Side(userID), pinned); err != nil {
			return false, err
		}
	}
	// Эхо в другие вкладки этого же пользователя.
	s.pub.Publish(ctx, "conversation:pin", rooms(userID),
		dto.ConversationPinEvent{ConversationID: convID, IsPinned: pinned})
	return pinned, nil
}

// TotalUnread — общее число непрочитанных по всем не скрытым диалогам и группам.
func (s *Service) TotalUnread(ctx context.Context, userID int64) (int, error) {
	pairs, err := s.repo.TotalUnread(ctx, userID)
	if err != nil {
		return 0, err
	}
	groups, err := s.repo.TotalGroupUnread(ctx, userID)
	if err != nil {
		return 0, err
	}
	return pairs + groups, nil
}
