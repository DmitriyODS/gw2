package service

import (
	"encoding/json"
	"fmt"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// maxBackgroundRecipeBytes — потолок размера рецепта (компактная композиция, не
// картинка; сама картинка живёт в uploads, в рецепте — только ссылка).
const maxBackgroundRecipeBytes = 4 * 1024

func userRoom(userID int64) string { return fmt.Sprintf("user_%d", userID) }

// GetBackground — рецепт оформления ленты пользователя (nil — не задан).
func (s *Service) GetBackground(ctx domain.Ctx, userID int64) (json.RawMessage, error) {
	return s.repo.GetPortalBackground(ctx, userID)
}

// SetBackground — сохранить рецепт и разослать эхо на другие устройства.
func (s *Service) SetBackground(ctx domain.Ctx, userID int64, recipe json.RawMessage) error {
	if len(recipe) == 0 || !json.Valid(recipe) {
		return domain.NewError("BAD_RECIPE", "Некорректный рецепт оформления", 400)
	}
	if len(recipe) > maxBackgroundRecipeBytes {
		return domain.NewError("RECIPE_TOO_LARGE", "Слишком большой рецепт оформления", 413)
	}
	if err := s.repo.UpsertPortalBackground(ctx, userID, recipe); err != nil {
		return err
	}
	s.bus.Publish(ctx, "portal_bg:updated", []string{userRoom(userID)},
		map[string]any{"recipe": recipe})
	return nil
}

// DeleteBackground — снять рецепт и разослать эхо.
func (s *Service) DeleteBackground(ctx domain.Ctx, userID int64) error {
	if err := s.repo.DeletePortalBackground(ctx, userID); err != nil {
		return err
	}
	s.bus.Publish(ctx, "portal_bg:updated", []string{userRoom(userID)},
		map[string]any{"recipe": nil})
	return nil
}
