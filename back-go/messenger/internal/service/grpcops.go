package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// EnsureDialog — найти/создать парный диалог (зовётся ДО StartCall).
func (s *Service) EnsureDialog(ctx context.Context, userAID, userBID int64) (int64, error) {
	conv, err := s.ensureConversation(ctx, userAID, userBID)
	if err != nil {
		return 0, err
	}
	return conv.ID, nil
}

// CreateCallMessage — системная плашка звонка kind='call' в парном диалоге.
// Возвращает готовый снапшот сообщения и адресатов message:new.
func (s *Service) CreateCallMessage(ctx context.Context, convID, senderID, callID int64) (*dto.Message, []int64, error) {
	conv, err := s.repo.GetConversation(ctx, convID)
	if err != nil {
		return nil, nil, err
	}
	if conv == nil {
		return nil, nil, errConvNotFound()
	}
	msg, err := s.repo.CreateMessage(ctx, domain.NewMessage{
		ConversationID: convID,
		SenderID:       &senderID,
		Kind:           domain.KindCall,
		CallID:         &callID,
	})
	if err != nil {
		return nil, nil, err
	}
	return dto.NewMessage(msg), pairNotifyIDs(conv), nil
}

// GetCallMessage — актуальный снапшот плашки звонка (статус из таблицы calls
// читается заново) для message:updated.
func (s *Service) GetCallMessage(ctx context.Context, callID int64) (int64, *dto.Message, []int64, error) {
	notFound := domain.NewError("CALL_MESSAGE_NOT_FOUND", "Плашка звонка не найдена", 404)

	call, err := s.repo.GetCall(ctx, callID)
	if err != nil {
		return 0, nil, nil, err
	}
	// Для group-звонков плашки нет (conversation_id пуст).
	if call == nil || call.ConversationID == nil {
		return 0, nil, nil, notFound
	}
	msg, err := s.repo.FindCallMessage(ctx, callID, *call.ConversationID)
	if err != nil {
		return 0, nil, nil, err
	}
	if msg == nil {
		return 0, nil, nil, notFound
	}
	conv, err := s.repo.GetConversation(ctx, *call.ConversationID)
	if err != nil {
		return 0, nil, nil, err
	}
	if conv == nil {
		return 0, nil, nil, notFound
	}
	return conv.ID, dto.NewMessage(msg), pairNotifyIDs(conv), nil
}

// CreatePostMessage — системная плашка пересланного поста kind='post' в
// диалоге (зовёт portalsvc). Превью — замороженный снапшот, переданный
// вызывающим (не JOIN на portal_posts — мессенджер не завязывается на схему
// portalsvc). Отправитель обязан быть участником диалога (как в
// ForwardMessage), в соло-чат (техподдержку) пересылка не имеет смысла.
// message:new публикует сам msgsvc (gw2:messenger:events) — тем же путём
// идут и пуши pushsvc. Возвращает снапшот сообщения и адресатов события.
func (s *Service) CreatePostMessage(ctx context.Context, convID, senderID, postID int64, title, excerpt, coverURL string) (*dto.Message, []int64, error) {
	conv, err := s.conversationForUser(ctx, convID, senderID)
	if err != nil {
		return nil, nil, err
	}
	if conv.IsSolo() {
		return nil, nil, domain.NewError("BAD_CONVERSATION",
			"В этот чат нельзя переслать пост", 400)
	}
	msg, err := s.repo.CreateMessage(ctx, domain.NewMessage{
		ConversationID: conv.ID,
		SenderID:       &senderID,
		Kind:           domain.KindPost,
		PostID:         &postID,
		PostTitle:      &title,
		PostExcerpt:    &excerpt,
		PostCoverURL:   nonEmpty(coverURL),
	})
	if err != nil {
		return nil, nil, err
	}
	payload := dto.NewMessage(msg)
	notify := pairNotifyIDs(conv)
	s.pub.Publish(ctx, "message:new", rooms(notify...), dto.MessageNewEvent{
		ConversationID: conv.ID, Message: payload, FromUserID: &senderID,
	})
	return payload, notify, nil
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
