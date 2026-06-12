package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/yougile"
)

// Регистрация и приём webhook'ов YouGile (порт webhook_service.py).
//
// Исходящая регистрация: при включении интеграции дёргаем POST /webhooks
// один раз; id храним в companies.yg_webhook_id, сгенерированный secret —
// в yg_webhook_secret (он в URL, по которому YG нам стучится).
// Ингресс: POST /api/yougile/webhook/<company_id>/<secret> — проверка
// секрета + применение событий (yougile_apply.go).

// webhookEventPattern — одна подписка `task-*` покрывает created/updated/
// moved/deleted/restored/renamed/completed. `chat_message-*` НЕ подписываем —
// чат мы намеренно не зеркалим.
const webhookEventPattern = "task-.*"

func (y *Yougile) ingressURL(company *domain.YougileCompany) (string, error) {
	if y.publicBase == "" {
		return "", domain.NewError("PUBLIC_BASE_MISSING",
			"Не задан YOUGILE_WEBHOOK_PUBLIC_BASE — невозможно зарегистрировать webhook", 400)
	}
	return y.publicBase + "/api/yougile/webhook/" +
		strconv.FormatInt(company.ID, 10) + "/" + strOrEmpty(company.YgWebhookSecret), nil
}

// ensureWebhookRegistered — зарегистрировать webhook на стороне YouGile
// (или обновить, если уже есть). Идемпотентно: при наличии yg_webhook_id
// делаем PUT на тот же id — это покрывает «сменили доску, надо
// переподписаться». Мутирует company и сохраняет yg_webhook_id/secret.
func (y *Yougile) ensureWebhookRegistered(ctx context.Context, actorID int64, company *domain.YougileCompany) error {
	if company.YgBoardID == nil || company.YgCompanyID == nil {
		return domain.NewError("NOT_CONFIGURED", "Сначала выберите проект и доску", 400)
	}
	client, err := y.buildClientForUser(ctx, actorID)
	if err != nil {
		return err
	}
	if client == nil {
		return domain.NewError("NOT_CONNECTED", "Подключите свой YouGile-аккаунт", 400)
	}

	fields := map[string]any{}
	if company.YgWebhookSecret == nil || *company.YgWebhookSecret == "" {
		secret := y.genSecret()
		company.YgWebhookSecret = &secret
		fields["yg_webhook_secret"] = secret
	}

	hookURL, err := y.ingressURL(company)
	if err != nil {
		return err
	}
	// Фильтр по location — id доски: YG пришлёт события только по нашим
	// карточкам, чужие доски игнор.
	filters := []map[string]any{{"name": "location", "value": []string{*company.YgBoardID}}}

	if company.YgWebhookID != nil && *company.YgWebhookID != "" {
		err = client.UpdateWebhook(*company.YgWebhookID, map[string]any{
			"url": hookURL, "event": webhookEventPattern, "filters": filters,
		})
	} else {
		var data map[string]any
		data, err = client.CreateWebhook(hookURL, webhookEventPattern, filters)
		if err == nil {
			wid, _ := data["id"].(string)
			if wid == "" {
				return domain.NewError("BAD_RESPONSE", "YouGile не вернул id webhook'а", 400)
			}
			company.YgWebhookID = &wid
			fields["yg_webhook_id"] = wid
		}
	}
	if err != nil {
		if yougile.IsAuth(err) {
			return domain.NewError("BAD_KEY",
				"Ключ YouGile недействителен, переподключите аккаунт", 400)
		}
		return domain.NewError("YOUGILE_ERROR", "YouGile: "+err.Error(), 400)
	}

	if err := y.repo.UpdateYougileCompanyFields(ctx, company.ID, fields); err != nil {
		return err
	}
	y.log.Info("yougile.webhook_registered",
		"company_id", company.ID, "webhook_id", strOrEmpty(company.YgWebhookID))
	return nil
}

// deregisterWebhook — отписаться. YouGile не даёт DELETE на /webhooks/{id},
// только PUT (отключение): шлём url-заглушку и event=disabled-*. Параллельно
// чистим локальные yg_webhook_id/secret, чтобы при следующем enable создать
// webhook с чистого листа.
func (y *Yougile) deregisterWebhook(ctx context.Context, actorID int64, company *domain.YougileCompany) error {
	if company.YgWebhookID == nil || *company.YgWebhookID == "" {
		return nil
	}
	client, err := y.buildClientForUser(ctx, actorID)
	if err != nil {
		return err
	}
	if client != nil {
		err := client.UpdateWebhook(*company.YgWebhookID, map[string]any{
			"url": "https://example.invalid/disabled", "event": "disabled-event",
			"filters": []map[string]any{},
		})
		if err != nil {
			y.log.Warn("yougile.webhook_deregister_failed",
				"company_id", company.ID, "error", err)
		}
	}
	company.YgWebhookID = nil
	company.YgWebhookSecret = nil
	if err := y.repo.UpdateYougileCompanyFields(ctx, company.ID, map[string]any{
		"yg_webhook_id": nil, "yg_webhook_secret": nil,
	}); err != nil {
		return err
	}
	y.log.Info("yougile.webhook_deregistered", "company_id", company.ID)
	return nil
}

// RegisterWebhook — POST /webhook/register (ручная регистрация на случай
// «сбросилось»/«поменяли URL»). Возвращает признак webhook_registered.
func (y *Yougile) RegisterWebhook(ctx context.Context, actor *domain.User, companyID int64) (bool, error) {
	company, err := y.repo.GetYougileCompany(ctx, companyID)
	if err != nil {
		return false, err
	}
	if company == nil {
		return false, domain.NewError("NO_COMPANY", "", 400)
	}
	if err := y.ensureWebhookRegistered(ctx, actor.ID, company); err != nil {
		return false, wrapMisconfig(err)
	}
	return company.YgWebhookID != nil && *company.YgWebhookID != "", nil
}

// verifySecret — constant-time сравнение, чтобы не утечь secret по таймингу.
func verifySecret(company *domain.YougileCompany, secret string) bool {
	expected := strOrEmpty(company.YgWebhookSecret)
	if expected == "" || secret == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(secret)) == 1
}

// HandleWebhook — приём событий YouGile (без токена — авторизация через
// secret в URL). found=false → транспорт отвечает 404, не светя деталями.
// Возвращаем 2xx и results, чтобы YG не ставил retry-задержку: один сбойный
// event не должен ронять весь batch — иначе YG переотправит его целиком.
func (y *Yougile) HandleWebhook(ctx context.Context, companyID int64, secret string,
	body []byte) (results []map[string]any, found bool, err error) {

	company, err := y.repo.GetYougileCompany(ctx, companyID)
	if err != nil {
		return nil, false, err
	}
	if company == nil || !verifySecret(company, secret) {
		return nil, false, nil
	}

	// YouGile может слать одиночное событие или массив — приводим к списку.
	var payload any
	_ = json.Unmarshal(body, &payload)
	var events []map[string]any
	switch v := payload.(type) {
	case []any:
		for _, item := range v {
			ev, _ := item.(map[string]any)
			if ev == nil {
				ev = map[string]any{}
			}
			events = append(events, ev)
		}
	case map[string]any:
		events = []map[string]any{v}
	default:
		events = []map[string]any{{}}
	}

	results = make([]map[string]any, 0, len(events))
	for _, ev := range events {
		res, err := y.applyEvent(ctx, company, ev)
		if err != nil {
			event, _ := ev["event"].(string)
			y.log.Error("yougile.webhook_apply_failed",
				"company_id", companyID, "event", event, "error", err)
			results = append(results, map[string]any{
				"status": "error", "message": err.Error(),
			})
			continue
		}
		results = append(results, res)
	}
	return results, true, nil
}
