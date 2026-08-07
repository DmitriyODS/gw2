// Package billingclient — общий клиент лимитов тарифа для микросервисов.
//
// Лимиты живут в billingsvc, но проверяет их тот сервис, который создаёт
// сущность: он один знает своё текущее количество. Клиент забирает лимиты по
// gRPC и держит их в коротком кэше — на горячем пути (создание задачи, заливка
// файла) не должно быть сетевого запроса на каждый вызов.
//
// Политика при недоступности биллинга — FAIL-OPEN: без ответа считаем, что
// ограничений нет. Упавший биллинг не должен останавливать работу платформы;
// в логе остаётся предупреждение.
package billingclient

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/billingpb"
)

// Unlimited — «без ограничения» (совпадает с domain.Unlimited биллинга).
const Unlimited = -1

// cacheTTL — насколько живёт снимок лимитов. Секунды задержки после покупки
// тарифа приемлемы, лишний gRPC на каждое действие — нет.
const cacheTTL = 30 * time.Second

// callTimeout — биллинг рядом, ждать его долго незачем.
const callTimeout = 2 * time.Second

// Entitlements — лимиты и расход владельца квоты.
type Entitlements struct {
	Plan        string
	PlanName    string
	Limits      *billingpb.Limits
	StorageUsed int64
	TokensLeft  int64
	OwnerID     int64
	// Fallback — ответ не получен, лимиты считаются безграничными.
	Fallback bool
}

// Client — клиент биллинга. Нулевой (nil) клиент разрешает всё: так сервис
// работает, когда BILLING_GRPC_ADDR не задан (локальная разработка без
// биллинга и тесты).
type Client struct {
	api  billingpb.BillingServiceClient
	conn *grpc.ClientConn
	log  *slog.Logger

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	ent *Entitlements
	at  time.Time
}

// New — клиент по адресу gRPC биллинга. Пустой адрес — выключено (nil).
func New(addr string, log *slog.Logger) (*Client, error) {
	if addr == "" {
		return nil, nil
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		api:   billingpb.NewBillingServiceClient(conn),
		conn:  conn,
		log:   log,
		cache: map[string]cacheEntry{},
	}, nil
}

func (c *Client) Close() {
	if c != nil && c.conn != nil {
		_ = c.conn.Close()
	}
}

// unlimited — разрешающий ответ: им отвечает выключенный или недоступный
// биллинг.
func unlimited() *Entitlements {
	return &Entitlements{
		Plan: "", Limits: &billingpb.Limits{
			Tasks: Unlimited, Companies: Unlimited, Members: Unlimited,
			StorageBytes: Unlimited, AiTokens: Unlimited, Calendars: Unlimited,
			Diaries: Unlimited, Boards: Unlimited, Registries: Unlimited,
			ChatFolders: Unlimited, CallParticipants: Unlimited,
			DataTransfer: true, AdvancedStats: true, Portal: true, UserStatuses: true,
			PremiumThemes: true, PremiumPetSkins: true, PremiumPetHouse: true, PremiumPetGoods: true,
		},
		Fallback: true,
	}
}

// Entitlements — лимиты пользователя (companyID > 0 — лимиты его компании,
// то есть тариф её создателя).
func (c *Client) Entitlements(ctx context.Context, userID, companyID int64) *Entitlements {
	if c == nil {
		return unlimited()
	}
	key := strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(companyID, 10)
	c.mu.Lock()
	if e, ok := c.cache[key]; ok && time.Since(e.at) < cacheTTL {
		c.mu.Unlock()
		return e.ent
	}
	c.mu.Unlock()

	rctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.api.GetEntitlements(rctx, &billingpb.GetEntitlementsRequest{
		UserId: userID, CompanyId: companyID,
	})
	if err != nil || res.GetLimits() == nil {
		c.log.Warn("billing.entitlements_failed", "user_id", userID, "company_id", companyID, "error", err)
		return unlimited()
	}
	ent := &Entitlements{
		Plan: res.GetPlan(), PlanName: res.GetPlanName(), Limits: res.GetLimits(),
		StorageUsed: res.GetStorageUsed(), TokensLeft: res.GetTokensLeft(), OwnerID: res.GetOwnerId(),
	}
	c.mu.Lock()
	c.cache[key] = cacheEntry{ent: ent, at: time.Now()}
	c.mu.Unlock()
	return ent
}

// Invalidate — сбросить кэш пользователя (после покупки лимиты должны
// действовать сразу, а не через полминуты).
func (c *Client) Invalidate(userID int64) {
	if c == nil {
		return
	}
	prefix := strconv.FormatInt(userID, 10) + ":"
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.cache {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.cache, k)
		}
	}
}

// StoredFile — файл, попавший в хранилище: строка журнала раздела
// «Настройки → Хранилище». Ref* описывают, ГДЕ файл лежит (сущность-носитель);
// их можно не заполнять — файл нередко грузится раньше своей сущности.
type StoredFile struct {
	Key   string
	Name  string
	Size  int64
	Kind  string // вид сущности: message, note, board, record, post, avatar
	ID    string // её идентификатор
	Title string // человекочитаемое «где именно»
}

// StorageChange — что случилось с файлами владельца. Занятое место биллинг
// пересчитает сам: размеры добавленных он получит здесь, размеры удалённых
// возьмёт из журнала — мерить объекты в хранилище перед удалением не нужно.
type StorageChange struct {
	Service string
	Added   []StoredFile
	Removed []string // ключи удалённых
}

// TrackStorage — сообщить биллингу о появившихся и удалённых файлах.
// companyID > 0 — файлы компании: владельца квоты (её создателя) находит сам
// биллинг. Ошибки только логируются: учёт места не должен ронять загрузку.
func (c *Client) TrackStorage(ctx context.Context, userID, companyID int64, ch StorageChange) {
	if c == nil || (userID <= 0 && companyID <= 0) {
		return
	}
	if len(ch.Added) == 0 && len(ch.Removed) == 0 {
		return
	}
	added := make([]*billingpb.StoredFile, 0, len(ch.Added))
	for _, f := range ch.Added {
		added = append(added, &billingpb.StoredFile{
			Key: f.Key, Name: f.Name, Size: f.Size,
			RefKind: f.Kind, RefId: f.ID, Title: f.Title,
		})
	}
	rctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	if _, err := c.api.TrackStorage(rctx, &billingpb.TrackStorageRequest{
		UserId: userID, CompanyId: companyID, Service: ch.Service,
		Added: added, RemovedKeys: ch.Removed,
	}); err != nil {
		c.log.Warn("billing.track_storage_failed", "user_id", userID, "error", err)
		return
	}
	c.Invalidate(userID)
}

// EnsureStorage — влезает ли файл размера bytes в квоту владельца.
func (c *Client) EnsureStorage(ctx context.Context, userID, companyID, bytes int64) error {
	if c == nil {
		return nil
	}
	ent := c.Entitlements(ctx, userID, companyID)
	limit := ent.Limits.GetStorageBytes()
	if limit == Unlimited {
		return nil
	}
	if ent.StorageUsed+bytes > limit {
		return LimitError("storage", limit, ent.StorageUsed, ent.PlanName)
	}
	return nil
}

// AI — остаток токенов и плательщик (для компанийных функций это создатель
// компании). ok=false — токенов нет.
func (c *Client) AI(ctx context.Context, userID, companyID int64) (payerID, left int64, ok bool) {
	if c == nil {
		return userID, Unlimited, true
	}
	rctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.api.CheckAI(rctx, &billingpb.CheckAIRequest{UserId: userID, CompanyId: companyID})
	if err != nil {
		c.log.Warn("billing.check_ai_failed", "user_id", userID, "error", err)
		return userID, Unlimited, true // fail-open
	}
	return res.GetPayerId(), res.GetTokensLeft(), res.GetAllowed()
}

// ConsumeAI — списать токены после обращения к модели. ok=false — не хватило.
func (c *Client) ConsumeAI(ctx context.Context, in *billingpb.ConsumeAIRequest) (ok bool, left int64) {
	if c == nil {
		return true, Unlimited
	}
	rctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := c.api.ConsumeAI(rctx, in)
	if err != nil {
		c.log.Warn("billing.consume_ai_failed", "payer_id", in.GetPayerId(), "error", err)
		return true, Unlimited
	}
	c.Invalidate(in.GetPayerId())
	return res.GetOk(), res.GetTokensLeft()
}

// LogAction — запись в журнал «Аудита платформы» (действия супер-админа из
// других сервисов). Fire-and-forget.
func (c *Client) LogAction(ctx context.Context, actorID int64, action, targetKind, targetID, summary string) {
	if c == nil {
		return
	}
	rctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	if _, err := c.api.LogAction(rctx, &billingpb.LogActionRequest{
		ActorId: actorID, Action: action, TargetKind: targetKind,
		TargetId: targetID, Summary: summary,
	}); err != nil {
		c.log.Warn("billing.log_action_failed", "action", action, "error", err)
	}
}

// ---------- Проверки лимитов ----------

// limitLabels — как лимит называется в сообщении пользователю.
var limitLabels = map[string]string{
	"tasks":        "задачи",
	"companies":    "компании",
	"members":      "участники компании",
	"calendars":    "календари",
	"diaries":      "ежедневники",
	"boards":       "доски",
	"registries":   "реестры",
	"chat_folders": "папки чатов",
	"call":         "участники звонка",
	"storage":      "хранилище",
	"tokens":       "токены ИИ",
}

// LimitError — превышение лимита тарифа. Код LIMIT_REACHED и HTTP 402: фронт
// по ним показывает предложение перейти в магазин.
func LimitError(kind string, limit, current int64, planName string) *apierror.Error {
	label := limitLabels[kind]
	if label == "" {
		label = kind
	}
	msg := "Достигнут предел тарифа: " + label + ". Оформите подписку в магазине, чтобы продолжить."
	if planName != "" {
		msg = "Тариф «" + planName + "» исчерпан по разделу «" + label + "». Оформите подписку в магазине, чтобы продолжить."
	}
	// Место — не про тариф: докупить его нельзя, пока подписки скрыты, зато
	// всегда можно освободить. Ведём туда, где видно, чем оно занято.
	if kind == "storage" {
		msg = "Место в хранилище закончилось. Освободите его в разделе «Настройки → Хранилище»."
	}
	return apierror.NewExtra("LIMIT_REACHED", msg, http.StatusPaymentRequired, map[string]any{
		"limit_kind": kind, "limit": limit, "current": current, "plan": planName,
	})
}

// FeatureError — возможность недоступна на текущем тарифе.
func FeatureError(feature, planName string) *apierror.Error {
	return apierror.NewExtra("PLAN_FEATURE_REQUIRED",
		"Эта возможность доступна на платном тарифе. Оформите подписку в магазине.",
		http.StatusPaymentRequired, map[string]any{"feature": feature, "plan": planName})
}

// EnsureCount — общая проверка «влезает ли ещё одна сущность». limit берётся
// из Limits вызывающим (он же знает current).
func EnsureCount(kind string, limit int, current int, planName string) error {
	if limit == Unlimited || current < limit {
		return nil
	}
	return LimitError(kind, int64(limit), int64(current), planName)
}
