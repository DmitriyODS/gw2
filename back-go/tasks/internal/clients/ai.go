package clients

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/aipb"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

const (
	aiStatusTimeout  = 5 * time.Second
	aiSearchTimeout  = 30 * time.Second
	aiStatusCacheTTL = time.Minute
)

// AI — клиент aisvc для поиска задач и реиндексации. Fail-open (как прежний
// get_ai_client во Flask): сервис недоступен / ИИ выключен → Enabled=false,
// поиск уходит в обычный LIKE. Status кэшируется per-company на минуту;
// негатив не кэшируется — включение ИИ подхватывается сразу.
type AI struct {
	conn *grpc.ClientConn
	stub aipb.AiServiceClient
	log  *slog.Logger

	mu    sync.Mutex
	cache map[int64]time.Time // company_id → enabled-до
}

var _ domain.AIClient = (*AI)(nil)

func NewAI(addr string, log *slog.Logger) (*AI, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &AI{
		conn:  conn,
		stub:  aipb.NewAiServiceClient(conn),
		log:   log,
		cache: map[int64]time.Time{},
	}, nil
}

func (c *AI) Close() { _ = c.conn.Close() }

func (c *AI) Enabled(ctx context.Context, companyID int64) bool {
	c.mu.Lock()
	until, ok := c.cache[companyID]
	c.mu.Unlock()
	if ok && time.Now().Before(until) {
		return true
	}

	rctx, cancel := context.WithTimeout(ctx, aiStatusTimeout)
	defer cancel()
	resp, err := c.stub.Status(rctx, &aipb.StatusRequest{CompanyId: companyID})
	if err != nil || resp.GetError() != nil || !resp.GetEnabled() {
		c.mu.Lock()
		delete(c.cache, companyID)
		c.mu.Unlock()
		return false
	}
	c.mu.Lock()
	c.cache[companyID] = time.Now().Add(aiStatusCacheTTL)
	c.mu.Unlock()
	return true
}

// SemanticSearch — id задач по убыванию релевантности. Fail-open: любая
// ошибка → nil (вызывающий трактует как пустую семантическую выдачу,
// как `except Exception: hits = []` во Flask).
func (c *AI) SemanticSearch(ctx context.Context, companyID int64, query string) []int64 {
	rctx, cancel := context.WithTimeout(ctx, aiSearchTimeout)
	defer cancel()
	resp, err := c.stub.SemanticSearch(rctx, &aipb.SemanticSearchRequest{
		CompanyId: companyID, Query: query,
	})
	if err != nil || resp.GetError() != nil {
		c.log.Warn("ai.search.failed", "company_id", companyID, "error", err)
		return nil
	}
	out := make([]int64, 0, len(resp.GetHits()))
	for _, h := range resp.GetHits() {
		out = append(out, h.GetTaskId())
	}
	return out
}

// ScheduleReindex — fire-and-forget переиндексация задачи: тихо игнорирует
// ошибки и никогда не валит вызывающий запрос (как schedule_reindex).
func (c *AI) ScheduleReindex(taskID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), aiStatusTimeout)
		defer cancel()
		resp, err := c.stub.ReindexTask(ctx, &aipb.ReindexTaskRequest{TaskId: taskID})
		if err != nil || resp.GetError() != nil {
			c.log.Warn("ai.reindex.failed", "task_id", taskID, "error", err)
		}
	}()
}
