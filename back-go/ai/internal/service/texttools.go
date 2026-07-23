// Package service — ИИ-инструменты текста для заметок (POST /api/ai/text-tools):
// одна операция над выделенным фрагментом (переписать, исправить, сократить,
// перевести и т. п.) одним ходом Chat, без tools-цикла и без сохранения
// диалога. Ключ и модель — активной компании пользователя (как у ассистента).
package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

const (
	// textToolMaxChars — потолок фрагмента: заметки правят кусками, а не
	// томами; больше — и медленно, и дорого, и бьёт в лимит контекста.
	textToolMaxChars = 12000
	// textToolMaxTokens — с запасом под «развернуть» длинного фрагмента.
	textToolMaxTokens = 1500
	// textToolTemperature — ниже дефолтной 0.7: редактор текста должен
	// сохранять смысл, а не сочинять.
	textToolTemperature = 0.3
	textToolTimeout     = 45 * time.Second
)

// textToolSystemPrompt — общий инвариант всех операций: в ответе ТОЛЬКО
// результат, без пояснений и Markdown (результат вставляется в заметку как
// обычный текст).
const textToolSystemPrompt = "Ты — редактор текста в личных заметках пользователя. " +
	"Выполни ровно одну операцию над присланным текстом. В ответ верни ТОЛЬКО результат " +
	"операции: без пояснений, вступлений, кавычек и Markdown-разметки. Сохраняй язык " +
	"исходного текста (если операция — не перевод) и переносы строк, где они есть."

// textToolTones — стили для action=tone (style — ключ).
var textToolTones = map[string]string{
	"formal":    "деловом, официальном тоне",
	"friendly":  "дружелюбном, тёплом тоне",
	"confident": "уверенном, убедительном тоне",
	"casual":    "непринуждённом, разговорном тоне",
}

// textToolLangs — целевые языки для action=translate (style — ключ).
var textToolLangs = map[string]string{
	"en": "английский",
	"ru": "русский",
}

func errTextToolValidation(msg string) *domain.Error {
	return domain.NewError("VALIDATION", msg, 400)
}

// textToolInstruction — формулировка операции для user-сообщения; style нужен
// только tone/translate, остальные его игнорируют.
func textToolInstruction(action, style string) (string, *domain.Error) {
	switch action {
	case "improve":
		return "Улучши текст: сделай его яснее и грамотнее, убери шероховатости и повторы. Смысл и структуру сохрани.", nil
	case "fix":
		return "Исправь орфографические, грамматические и пунктуационные ошибки. Формулировки, стиль и структуру не меняй.", nil
	case "rephrase":
		return "Переформулируй текст другими словами, сохранив смысл и примерно тот же объём.", nil
	case "shorten":
		return "Сократи текст примерно вдвое, сохранив главный смысл.", nil
	case "expand":
		return "Разверни текст: добавь уместные детали и связки, сохранив смысл и тон.", nil
	case "simplify":
		return "Упрости текст: короткие предложения, простые слова, без канцелярита. Смысл сохрани.", nil
	case "summarize":
		return "Составь краткое резюме текста: 1–3 предложения с главными мыслями.", nil
	case "bullets":
		return "Преобразуй текст в список тезисов: каждый тезис с новой строки, начиная с «— », без вступления.", nil
	case "continue":
		return "Продолжи текст: напиши 1–3 предложения логичного продолжения в том же стиле и языке. Верни ТОЛЬКО продолжение, без исходного текста.", nil
	case "tone":
		label, ok := textToolTones[style]
		if !ok {
			return "", errTextToolValidation("Неизвестный тон")
		}
		return "Перепиши текст в " + label + ", сохранив смысл.", nil
	case "translate":
		lang, ok := textToolLangs[style]
		if !ok {
			return "", errTextToolValidation("Неизвестный язык перевода")
		}
		return "Переведи текст на " + lang + " язык, сохранив смысл, тон и переносы строк.", nil
	}
	return "", errTextToolValidation("Неизвестное действие")
}

// TransformText — выполнить операцию action над text. Компания без
// включённого AI → AI_DISABLED 409 (как у ассистента: UI подскажет включить).
func (s *Service) TransformText(ctx context.Context, companyID int64, action, style, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errTextToolValidation("Текст не может быть пустым")
	}
	if utf8.RuneCountInString(text) > textToolMaxChars {
		return "", errTextToolValidation("Слишком длинный фрагмент — выделите меньше текста")
	}
	instruction, derr := textToolInstruction(action, style)
	if derr != nil {
		return "", derr
	}
	client, err := s.clientFor(ctx, companyID)
	if err != nil {
		return "", err
	}
	if client == nil {
		return "", errAiDisabled(409)
	}

	messages := []map[string]any{
		{"role": "system", "content": textToolSystemPrompt},
		{"role": "user", "content": instruction + "\n\nТекст:\n" + text},
	}
	raw, err := json.Marshal(messages)
	if err != nil {
		return "", err
	}
	res, err := s.Chat(ctx, ChatArgs{
		CompanyID:    companyID,
		MessagesJSON: string(raw),
		MaxTokens:    textToolMaxTokens,
		Temperature:  textToolTemperature,
		TimeoutSec:   textToolTimeout.Seconds(),
	})
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(res.Content)
	if out == "" {
		return "", domain.NewError("AI_EMPTY", "ИИ не вернул результат — попробуйте ещё раз", 502)
	}
	return out, nil
}

const (
	// proofreadMaxChars — суммарный потолок текста заметки за один прогон.
	proofreadMaxChars  = 16000
	proofreadMaxTokens = 6000
)

// proofreadSystemPrompt — корректура без переписывания: правим только ошибки,
// структуру и порядок сегментов сохраняем строго (сегменты — текстовые узлы
// TipTap-документа, форматирование восстанавливается по индексу на клиенте).
const proofreadSystemPrompt = "Ты — корректор текста в личных заметках. На вход даётся JSON-массив строк " +
	"(фрагменты одной заметки по порядку). Исправь в КАЖДОЙ строке орфографические, грамматические и " +
	"пунктуационные ошибки. НЕ меняй смысл, стиль, формулировки, регистр там где он уместен, порядок и " +
	"КОЛИЧЕСТВО строк. Не объединяй и не разбивай строки, не добавляй и не удаляй их. Сохраняй язык каждой " +
	"строки. Ответ — СТРОГО JSON-массив строк той же длины и порядка, без пояснений и Markdown."

// Proofread — корректура орфографии и пунктуации по всем текстовым сегментам
// заметки за один вызов LLM (сохраняет структуру/картинки/таблицы: клиент лишь
// подменяет текст узлов по индексу). Возвращает массив той же длины; при сбое
// формата — доменная ошибка (клиент покажет тост, ничего не подменит).
func (s *Service) Proofread(ctx context.Context, companyID int64, segments []string) ([]string, error) {
	if len(segments) == 0 {
		return nil, errTextToolValidation("Нет текста для проверки")
	}
	total := 0
	for _, seg := range segments {
		total += utf8.RuneCountInString(seg)
	}
	if total == 0 {
		return nil, errTextToolValidation("Нет текста для проверки")
	}
	if total > proofreadMaxChars {
		return nil, errTextToolValidation("Слишком большая заметка для проверки за раз")
	}
	client, err := s.clientFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errAiDisabled(409)
	}

	input, err := json.Marshal(segments)
	if err != nil {
		return nil, err
	}
	messages := []map[string]any{
		{"role": "system", "content": proofreadSystemPrompt},
		{"role": "user", "content": "Фрагменты заметки:\n" + string(input)},
	}
	raw, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	res, err := s.Chat(ctx, ChatArgs{
		CompanyID:    companyID,
		MessagesJSON: string(raw),
		MaxTokens:    proofreadMaxTokens,
		Temperature:  textToolTemperature,
		TimeoutSec:   textToolTimeout.Seconds(),
	})
	if err != nil {
		return nil, err
	}
	out, ok := parseProofread(res.Content, len(segments))
	if !ok {
		return nil, domain.NewError("AI_EMPTY", "ИИ вернул некорректный результат — попробуйте ещё раз", 502)
	}
	return out, nil
}

// parseProofread — извлечь JSON-массив строк из ответа модели (терпим к обёртке
// в ```json fences); длина обязана совпасть с числом сегментов.
func parseProofread(content string, want int) ([]string, bool) {
	content = strings.TrimSpace(content)
	if i := strings.IndexByte(content, '['); i >= 0 {
		if j := strings.LastIndexByte(content, ']'); j > i {
			content = content[i : j+1]
		}
	}
	var out []string
	if json.Unmarshal([]byte(content), &out) != nil || len(out) != want {
		return nil, false
	}
	return out, true
}
