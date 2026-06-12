package service

import (
	"context"
	"sort"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// ListConversations — список диалогов пользователя: закреплённые → остальные
// (по last_message_at), сверху pet-чат и личный dev-чат (для сотрудников
// компаний; у Администратора системы своих соло-чатов нет).
func (s *Service) ListConversations(ctx context.Context, userID int64) ([]*dto.ConversationListItem, error) {
	me, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Личный чат техподдержки должен существовать у сотрудника компании
	// всегда, даже без переписки. Бизнес-блокировку (доменную ошибку)
	// глотаем — не валим листинг.
	if me != nil && me.CompanyID != nil {
		if _, err := s.getOrCreateSolo(ctx, userID, *me.CompanyID, false); err != nil {
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

	if me != nil && me.CompanyID != nil {
		// Личный dev-чат — первым…
		dev, err := s.soloListItem(ctx, userID, false)
		if err != nil {
			return nil, err
		}
		if dev != nil {
			result = append([]*dto.ConversationListItem{dev}, result...)
		}
		// …а чат с Грувиком — самым первым (если уже создан в «Моём Groove»).
		pet, err := s.soloListItem(ctx, userID, true)
		if err != nil {
			return nil, err
		}
		if pet != nil {
			result = append([]*dto.ConversationListItem{pet}, result...)
		}
	}
	return result, nil
}

// soloListItem — элемент списка для dev/pet-чата владельца; nil — чата нет.
func (s *Service) soloListItem(ctx context.Context, userID int64, pet bool) (*dto.ConversationListItem, error) {
	conv, err := s.repo.GetSolo(ctx, userID, pet)
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
	var petName *string
	if pet {
		if petName, err = s.repo.PetName(ctx, userID); err != nil {
			return nil, err
		}
	}
	return &dto.ConversationListItem{
		ID:            conv.ID,
		LastMessage:   dto.NewMessage(last[conv.ID]),
		UnreadCount:   unread[conv.ID],
		LastMessageAt: dto.JSONTimePtr(conv.LastMessageAt),
		IsDevChat:     conv.IsDevChat,
		IsPetChat:     conv.IsPetChat,
		PetName:       petName,
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
		Conversation: *dto.NewConversation(conv, nil),
		OtherUser:    dto.NewDirectoryUser(other),
	}, nil
}

// getOrCreateSolo — dev/pet-чат владельца (get_or_create_*_chat_for_user).
func (s *Service) getOrCreateSolo(ctx context.Context, userID, companyID int64, pet bool) (*domain.Conversation, error) {
	conv, err := s.repo.GetSolo(ctx, userID, pet)
	if err != nil || conv != nil {
		return conv, err
	}
	return s.repo.CreateSolo(ctx, userID, companyID, pet)
}

// OpenDevChat — личный чат с техподдержкой (у Администратора системы его нет).
func (s *Service) OpenDevChat(ctx context.Context, userID int64) (*dto.Conversation, error) {
	me, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if me == nil {
		return nil, domain.NewError("USER_NOT_FOUND", "Пользователь не найден", 404)
	}
	if me.CompanyID == nil {
		return nil, domain.NewError("ADMIN_HAS_NO_DEVCHAT",
			"У Администратора системы нет своего чата с техподдержкой", 400)
	}
	conv, err := s.getOrCreateSolo(ctx, userID, *me.CompanyID, false)
	if err != nil {
		return nil, err
	}
	return dto.NewConversation(conv, nil), nil
}

// OpenPetChat — чат пользователя со своим Грувиком.
func (s *Service) OpenPetChat(ctx context.Context, userID int64) (*dto.Conversation, error) {
	me, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if me == nil {
		return nil, domain.NewError("USER_NOT_FOUND", "Пользователь не найден", 404)
	}
	if me.CompanyID == nil {
		return nil, domain.NewError("ADMIN_HAS_NO_PET", "У Администратора системы нет Грувика", 400)
	}
	conv, err := s.getOrCreateSolo(ctx, userID, *me.CompanyID, true)
	if err != nil {
		return nil, err
	}
	petName, err := s.repo.PetName(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.NewConversation(conv, petName), nil
}

// SupportInbox — все личные dev-чаты пользователей (вкладка «Техподдержка»
// Администратора системы).
func (s *Service) SupportInbox(ctx context.Context, userID int64) ([]*dto.ConversationListItem, error) {
	me, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if me == nil || me.CompanyID != nil {
		return nil, domain.NewError("FORBIDDEN", "Только Администратор системы", 403)
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
	// Соло-чаты удалять нельзя — они живут сколько живёт владелец.
	if conv.IsDevChat {
		return false, domain.NewError("DEV_CHAT_UNDELETABLE", "Чат техподдержки удалить нельзя", 400)
	}
	if conv.IsPetChat {
		return false, domain.NewError("PET_CHAT_UNDELETABLE", "Чат с Грувиком удалить нельзя — он обидится", 400)
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
	pinned := conv.PinnedAtFor(userID) == nil
	if err := s.repo.SetConversationPin(ctx, convID, conv.Side(userID), pinned); err != nil {
		return false, err
	}
	// Эхо в другие вкладки этого же пользователя.
	s.pub.Publish(ctx, "conversation:pin", rooms(userID),
		dto.ConversationPinEvent{ConversationID: convID, IsPinned: pinned})
	return pinned, nil
}

// TotalUnread — общее число непрочитанных по всем не скрытым диалогам.
func (s *Service) TotalUnread(ctx context.Context, userID int64) (int, error) {
	return s.repo.TotalUnread(ctx, userID)
}
