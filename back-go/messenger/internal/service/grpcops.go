package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// petChatHistoryLimit — дефолт глубины контекста pet-чата (как last_messages).
const petChatHistoryLimit = 12

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

// PostBotMessage — бот-сообщение Грувика (sender NULL + is_bot) в pet-чат;
// message:new владельцу публикуем сами.
func (s *Service) PostBotMessage(ctx context.Context, convID int64, text string) (int64, error) {
	conv, err := s.repo.GetConversation(ctx, convID)
	if err != nil {
		return 0, err
	}
	if conv == nil {
		return 0, errConvNotFound()
	}
	if !conv.IsPetChat {
		return 0, domain.NewError("NOT_PET_CHAT", "Диалог не является pet-чатом", 400)
	}
	if text == "" {
		return 0, domain.NewError("EMPTY_MESSAGE", "Пустое сообщение", 400)
	}
	msg, err := s.repo.CreateMessage(ctx, domain.NewMessage{
		ConversationID: convID,
		Text:           &text,
		Kind:           domain.KindText,
		IsBot:          true,
	})
	if err != nil {
		return 0, err
	}
	s.pub.Publish(ctx, "message:new", rooms(conv.UserAID), dto.MessageNewEvent{
		ConversationID: convID, Message: dto.NewMessage(msg), FromUserID: nil,
	})
	return msg.ID, nil
}

// ListRecentMessages — последние limit сообщений диалога в хронологическом
// порядке (контекст AI-ответа pet-чата).
func (s *Service) ListRecentMessages(ctx context.Context, convID int64, limit int) ([]*domain.Message, error) {
	if limit <= 0 {
		limit = petChatHistoryLimit
	}
	return s.repo.ListRecent(ctx, convID, limit)
}
