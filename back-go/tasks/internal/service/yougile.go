package service

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/yougile"
)

// Yougile — бизнес-логика интеграции с YouGile внутри tasksvc. Портировано
// из back/app/integrations/yougile/{account,company,task,webhook}_service.py,
// task_push.py и task_apply.py без изменения правил.
//
// Антицикл двусторонней синхры: после каждой исходящей правки/создания пишем
// yougile_sync_hash от полей, которые сами отправили (title/deadline/
// completed); входящие вебхуки сравнивают и игнорят своё же эхо.
type Yougile struct {
	svc        *Service
	repo       domain.YougileRepository
	cipher     domain.YougileCipher
	newClient  func(key string) domain.YougileAPI // key "" — анонимный (auth-флоу)
	publicBase string                             // YOUGILE_WEBHOOK_PUBLIC_BASE
	genSecret  func() string
	log        *slog.Logger
}

type YougileDeps struct {
	Service    *Service
	Repo       domain.YougileRepository
	Cipher     domain.YougileCipher
	NewClient  func(key string) domain.YougileAPI
	PublicBase string
	GenSecret  func() string // nil — crypto/rand (token_urlsafe(24))
	Log        *slog.Logger
}

func NewYougile(d YougileDeps) *Yougile {
	gen := d.GenSecret
	if gen == nil {
		gen = randomSecret
	}
	y := &Yougile{
		svc: d.Service, repo: d.Repo, cipher: d.Cipher, newClient: d.NewClient,
		publicBase: strings.TrimRight(strings.TrimSpace(d.PublicBase), "/"),
		genSecret:  gen, log: d.Log,
	}
	d.Service.yg = y
	return y
}

// randomSecret — как secrets.token_urlsafe(24).
func randomSecret() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// ── общие ошибки ──────────────────────────────────────────────────────────

// errEncKeyMisconfigured — 500: на сервере не задан YOUGILE_ENC_KEY. Видно
// админу — он знает, что нужно поправить env.
var errEncKeyMisconfigured = domain.NewError(
	"ENC_KEY_MISCONFIGURED", "На сервере не задан YOUGILE_ENC_KEY", 500)

// wrapMisconfig — ErrMisconfigured пакета yougile → доменная 500-ошибка.
func wrapMisconfig(err error) error {
	if errors.Is(err, yougile.ErrMisconfigured) {
		return errEncKeyMisconfigured
	}
	return err
}

// ── клиент по запросу ─────────────────────────────────────────────────────

// buildClientForUser — готовый клиент с расшифрованным ключом или nil.
//
// nil означает «пользователь не подключён или ключ не расшифровался».
// Вызывающая сторона решает: отдать 412 «подключите YG» или молча пропустить.
func (y *Yougile) buildClientForUser(ctx context.Context, userID int64) (domain.YougileAPI, error) {
	acc, err := y.repo.GetYougileAccount(ctx, userID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, nil
	}
	key, err := y.cipher.DecryptKey(acc.KeyCiphertext)
	if err != nil {
		return nil, wrapMisconfig(err)
	}
	if key == "" {
		// ENC_KEY сменили — UI попросит переподключение.
		return nil, nil
	}
	return y.newClient(key), nil
}

// ── URL-хелперы ───────────────────────────────────────────────────────────

// shortTeamID — `773aa0c0-…-ed7037760782` → `ed7037760782`: YouGile в
// адресной строке использует последние 12 hex-символов UUID компании.
func shortTeamID(ygCompanyID string) string {
	clean := strings.ReplaceAll(ygCompanyID, "-", "")
	if len(clean) >= 12 {
		return strings.ToLower(clean[len(clean)-12:])
	}
	return ""
}

// ygTaskURL — канонический URL карточки в YouGile. Если есть idTaskProject
// (`OIP1-2454`) — короткая ссылка, как видит её пользователь в адресной
// строке; иначе fallback на формат с UUID.
func ygTaskURL(ygCompanyID, ygTaskID, idShort string) string {
	short := shortTeamID(ygCompanyID)
	if idShort != "" && short != "" {
		return "https://ru.yougile.com/team/" + short + "/#" + idShort
	}
	return "https://ru.yougile.com/team/" + ygCompanyID + "/#tasks?task=" + ygTaskID
}

// gwTaskURL — canonical-ссылка на задачу в GW. origin приходит из запроса
// (как request.url_root во Flask), чтобы сервис не зависел от транспорта.
func gwTaskURL(taskID int64, origin string) string {
	base := strings.TrimRight(origin, "/")
	return base + "/tasks/" + strconv.FormatInt(taskID, 10)
}

// ── время и антицикл ──────────────────────────────────────────────────────

func msToTime(ms int64) *time.Time {
	if ms == 0 {
		return nil
	}
	t := time.UnixMilli(ms).UTC()
	return &t
}

func timeToMs(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}

// jsonNumber — числовое значение из payload вебхука/ответа YG
// (encoding/json даёт float64).
func jsonNumber(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}

// syncHash — хеш «состояния, которое мы только что отправили в YG».
//
// Если вебхук вернёт payload с тем же хешем — игнор. Поля выбраны строго те,
// которыми реально обмениваемся (title/deadline/completed): описание и
// чек-листы мы НЕ синхронизируем и сознательно НЕ включаем в хеш — иначе
// push (где описания нет) и приём вебхука (где оно есть) давали бы разные
// хеши, и антицикл не ловил бы собственное эхо.
func syncHash(title string, deadlineMs int64, completed bool) string {
	ms := ""
	if deadlineMs != 0 {
		ms = strconv.FormatInt(deadlineMs, 10)
	}
	done := "0"
	if completed {
		done = "1"
	}
	parts := strings.TrimSpace(title) + "|" + ms + "|" + done
	sum := sha1.Sum([]byte(parts))
	return hex.EncodeToString(sum[:])
}

// ── системные сообщения ───────────────────────────────────────────────────

// postSystemComment — системное сообщение в чат GW-задачи. Пишем от лица
// инициатора (отдельного «system»-юзера нет). Префикс 🔗 помогает фронту
// понять «это служебное». Best-effort: ошибка только в лог.
func (y *Yougile) postSystemComment(ctx context.Context, taskID, authorID int64, text string) {
	comment := &domain.Comment{TaskID: taskID, AuthorID: authorID, Text: text}
	if err := y.svc.comments.CreateComment(ctx, comment); err != nil {
		y.log.Warn("yougile.system_comment_failed", "task_id", taskID, "error", err)
	}
}

// postYGLinkBack — системное сообщение в чат YG-карточки: «Карточка в GW:
// <url>». Чат задачи в YG: id чата == id задачи. text+textHtml+label
// обязательны (CreateChatMessageDto).
func (y *Yougile) postYGLinkBack(client domain.YougileAPI, ygTaskID, gwURL string) {
	err := client.PostChatMessage(ygTaskID, map[string]any{
		"text": "🔗 Карточка в Groove Work: " + gwURL,
		"textHtml": fmt.Sprintf(
			"<p>🔗 Карточка в Groove Work: <a href=%q>%s</a></p>", gwURL, gwURL),
		"label": "Groove Work",
	})
	if err != nil {
		y.log.Warn("yougile.post_link_back_failed", "yg_task_id", ygTaskID, "error", err)
	}
}

// mapStr — строковое значение ключа из ответа YG (fmt.Sprint для
// не-строк, как str() в Python).
func mapStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprint(v)
	}
	return ""
}

// strOrEmpty / normalizeOpt — работа с nullable yg-полями компании.
func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// normalizeOpt — пустая строка → NULL (как `payload[...] or None` во Flask).
func normalizeOpt(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}
