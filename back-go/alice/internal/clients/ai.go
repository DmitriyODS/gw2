package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/aipb"
)

// aiParseTimeout — бюджет ИИ-разбора: вебхук Диалогов обязан ответить за
// секунды, при просрочке фолбэк на классический парсер.
const aiParseTimeout = 3 * time.Second

// AI — ИИ-разбор голосовой фразы через aisvc.Chat (ключ активной компании).
type AI struct {
	conn *grpc.ClientConn
	stub aipb.AiServiceClient
}

var _ domain.IntentParser = (*AI)(nil)

func NewAI(addr string) (*AI, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	return &AI{conn: conn, stub: aipb.NewAiServiceClient(conn)}, nil
}

func (c *AI) Close() { _ = c.conn.Close() }

const intentSystemPrompt = `Ты — разборщик голосовых команд корпоративного навыка Groove Work. Верни ТОЛЬКО JSON без пояснений:
{"kind":"...","title":"...","text":"...","date":"YYYY-MM-DD"}
kind — один из: task_create (добавить задачу), task_close (закрыть/завершить задачу), task_list (список моих задач), unit_start (начать работу/юнит над задачей), unit_stop (остановить работу/юнит), unit_status (что сейчас в работе), diary_create (создать ежедневник), diary_add (записать дело/план в ежедневник), diary_list (что запланировано на день), diary_done (отметить дело выполненным), diary_move (перенести дело на другой день), diary_delete (удалить дело из ежедневника), note_create (создать заметку), note_append (дописать в заметку), note_read (прочитать заметку), note_delete (удалить заметку), folder_create (создать папку заметок), help (помощь), unknown (не команда).
title — главный аргумент: название задачи/заметки/папки/ежедневника или текст дела, БЕЗ дат и служебных слов. text — тело заметки или текст дописки (для note_create/note_append), иначе пусто. date — только если во фразе назван день (сегодня/завтра/пятница/15 июля…), иначе пусто.
Если пользователь просто болтает — kind unknown.`

func (c *AI) ParseIntent(ctx context.Context, companyID int64, utterance string, now time.Time) (*domain.Intent, error) {
	ctx, cancel := context.WithTimeout(ctx, aiParseTimeout)
	defer cancel()

	weekdays := [...]string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}
	system := intentSystemPrompt + fmt.Sprintf("\nСегодня %s (%s).", now.Format("2006-01-02"), weekdays[now.Weekday()])
	messages, err := json.Marshal([]map[string]string{
		{"role": "system", "content": system},
		{"role": "user", "content": utterance},
	})
	if err != nil {
		return nil, err
	}
	resp, err := c.stub.Chat(ctx, &aipb.ChatRequest{
		CompanyId:    companyID,
		MessagesJson: string(messages),
		MaxTokens:    200,
		TimeoutSec:   aiParseTimeout.Seconds(),
	})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != "" {
		return nil, domain.NewError(resp.GetError().GetCode(), resp.GetError().GetMessage(), int(resp.GetError().GetHttpStatus()))
	}
	return parseIntentJSON(resp.GetContent())
}

// parseIntentJSON — JSON из ответа модели (терпит ```json-заборы и мусор
// вокруг фигурных скобок).
func parseIntentJSON(content string) (*domain.Intent, error) {
	start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("ответ модели без JSON")
	}
	var raw struct {
		Kind  string `json:"kind"`
		Title string `json:"title"`
		Text  string `json:"text"`
		Date  string `json:"date"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &raw); err != nil {
		return nil, err
	}
	if raw.Date != "" {
		if _, err := time.Parse("2006-01-02", raw.Date); err != nil {
			raw.Date = ""
		}
	}
	return &domain.Intent{
		Kind:  strings.TrimSpace(raw.Kind),
		Title: strings.TrimSpace(raw.Title),
		Text:  strings.TrimSpace(raw.Text),
		Date:  raw.Date,
	}, nil
}
