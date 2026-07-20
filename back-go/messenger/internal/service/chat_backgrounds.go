package service

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// maxChatBgRecipeBytes — потолок размера рецепта. Рецепт — компактная композиция
// (пресет/пятна/узор), а не картинка; лимит отсекает злоупотребления JSONB.
const maxChatBgRecipeBytes = 4 * 1024

// GetChatBackgrounds — весь набор оформления чатов пользователя.
func (s *Service) GetChatBackgrounds(ctx context.Context, userID int64) (*dto.ChatBackgroundsResponse, error) {
	list, err := s.repo.ListChatBackgrounds(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := &dto.ChatBackgroundsResponse{Overrides: map[string]json.RawMessage{}}
	for _, bg := range list {
		if bg.ConversationID == nil {
			resp.Default = bg.Recipe
			continue
		}
		resp.Overrides[strconv.FormatInt(*bg.ConversationID, 10)] = bg.Recipe
	}
	return resp, nil
}

// SetChatBackground — сохранить рецепт (convID nil — общий дефолт) и разослать
// эхо в другие устройства этого же пользователя.
func (s *Service) SetChatBackground(ctx context.Context, userID int64, convID *int64, recipe json.RawMessage) error {
	if len(recipe) == 0 || !json.Valid(recipe) {
		return domain.NewError("BAD_RECIPE", "Некорректный рецепт оформления", 400)
	}
	if len(recipe) > maxChatBgRecipeBytes {
		return domain.NewError("RECIPE_TOO_LARGE", "Слишком большой рецепт оформления", 413)
	}
	// Переопределение возможно только для доступного пользователю чата.
	if convID != nil {
		if _, err := s.conversationForUser(ctx, *convID, userID); err != nil {
			return err
		}
	}
	if err := s.repo.UpsertChatBackground(ctx, userID, convID, recipe); err != nil {
		return err
	}
	s.pub.Publish(ctx, "chat_bg:updated", rooms(userID),
		dto.ChatBackgroundEvent{ConversationID: convID, Recipe: recipe})
	return nil
}

// DeleteChatBackground — снять рецепт (convID nil — дефолт) и разослать эхо.
func (s *Service) DeleteChatBackground(ctx context.Context, userID int64, convID *int64) error {
	if err := s.repo.DeleteChatBackground(ctx, userID, convID); err != nil {
		return err
	}
	s.pub.Publish(ctx, "chat_bg:updated", rooms(userID),
		dto.ChatBackgroundEvent{ConversationID: convID, Recipe: nil})
	return nil
}
