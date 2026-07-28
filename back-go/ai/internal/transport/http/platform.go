package http

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
)

// Платформенный ИИ (супер-админ): глобальный ключ proxy-api, адрес API,
// модели по умолчанию и каталог с ценами. Цена модели задаёт стоимость
// обращения в токенах доступа, поэтому правка каталога сразу меняет
// тарификацию для всех.

func (h *handlers) getPlatformSettings(c *fiber.Ctx) error {
	res, err := h.eps.Service.GetPlatformSettings(c.Context())
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(res)
}

func (h *handlers) updatePlatformSettings(c *fiber.Ctx) error {
	var upd dto.PlatformAiUpdate
	if err := json.Unmarshal(c.Body(), &upd); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "message": "Некорректное тело запроса",
		})
	}
	res, err := h.eps.Service.UpdatePlatformSettings(c.Context(), upd)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(res)
}

func (h *handlers) testPlatformSettings(c *fiber.Ctx) error {
	res, err := h.eps.Service.TestPlatformSettings(c.Context())
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(res)
}
