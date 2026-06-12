package service

import (
	"regexp"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// Правила — порт schemas/user.py (marshmallow).

var (
	emailRe    = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
	nonDigitRe = regexp.MustCompile(`\D`)
)

func errValidation(message string) error {
	return domain.NewError("VALIDATION_ERROR", message, 400)
}

func validateFIO(fio string) error {
	if l := len([]rune(fio)); l < 1 || l > 255 {
		return errValidation("ФИО должно быть от 1 до 255 символов")
	}
	return nil
}

func validateLogin(login string) error {
	if l := len([]rune(login)); l < 3 || l > 100 {
		return errValidation("Логин должен быть от 3 до 100 символов")
	}
	return nil
}

func validatePost(post *string) error {
	if post != nil && len([]rune(*post)) > 255 {
		return errValidation("Должность не длиннее 255 символов")
	}
	return nil
}

func validatePassword(password string) error {
	if len([]rune(password)) < 8 {
		return errValidation("Пароль должен содержать минимум 8 символов")
	}
	return nil
}

// normalizePhone — приводит к +7XXXXXXXXXX; пустая строка → nil (очистка).
func normalizePhone(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	digits := nonDigitRe.ReplaceAllString(*raw, "")
	if digits == "" {
		return nil, nil
	}
	if strings.HasPrefix(digits, "8") && len(digits) == 11 {
		digits = "7" + digits[1:]
	}
	if len(digits) == 10 {
		digits = "7" + digits
	}
	if len(digits) != 11 || !strings.HasPrefix(digits, "7") {
		return nil, errValidation("Телефон должен быть российским мобильным (+7…)")
	}
	normalized := "+" + digits
	return &normalized, nil
}

// normalizeEmail — валидация формата; пустая строка → nil (очистка).
func normalizeEmail(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	if !emailRe.MatchString(trimmed) {
		return nil, errValidation("Неверный формат email")
	}
	return &trimmed, nil
}
