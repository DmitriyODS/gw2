// Package apitest — интеграционные API-тесты микросервисов Groove Work.
//
// Харнес: требует dev-инфраструктуру Docker (Redis :6379 и mailpit SMTP :1025 —
// `cd deploy && docker compose up -d`); Postgres поднимается СВОИМ контейнером
// gw2-apitest-db на :15432 (общий порт :5432 делят другие проекты машины, и
// DROP DATABASE рядом с dev-данными ни к чему). Нет инфраструктуры — тесты
// пропускаются, не падают. На каждый прогон создаётся ЧИСТАЯ тестовая БД
// gw2_apitest (drop+create), в неё накатываются goose-миграции
// (`go run ./cmd/migrate`), затем стартуют реальные сервисы (go run) на портах
// +10000 к dev-портам: authsvc :18091, diarysvc :18101, tasksvc :18095,
// registrysvc :18099, calendarsvc :18100, msgsvc :18092/:19092,
// groovesvc :18094/:19094, gatewaysvc :18096, pushsvc :18097 (FCM off),
// mailsvc gRPC :19098. aisvc и callsvc НЕ поднимаются: AI-пути обязаны быть
// fail-open (поиск → LIKE, Грувик → статичные реплики), а команды call:*
// шлюза — отвечать CALLS_UNAVAILABLE (это тоже проверка).
// Тесты ходят по HTTP как настоящий клиент; коды подтверждения email и токены
// сброса пароля читаются напрямую из тестовой БД (надёжнее парсинга писем).
package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// Тестам выделен СВОЙ Postgres-контейнер на :15432 (тот же образ, что в
	// deploy). Дев-БД на :5432 не трогаем: порт делят другие проекты машины, а
	// DROP DATABASE рядом с рабочими данными ни к чему. Redis и mailpit —
	// общие из dev-инфраструктуры (cd deploy && docker compose up -d).
	pgImage     = "pgvector/pgvector:pg16"
	pgContainer = "gw2-apitest-db"
	pgPort      = "15432"
	pgAdminURL  = "postgresql://grovework:grovework_local@localhost:" + pgPort + "/grovework?sslmode=disable"
	testDBName  = "gw2_apitest"
	testDBURL   = "postgresql://grovework:grovework_local@localhost:" + pgPort + "/" + testDBName + "?sslmode=disable"

	authBase      = "http://localhost:18091"
	diaryBase     = "http://localhost:18101"
	tasksBase     = "http://localhost:18095"
	registryBase  = "http://localhost:18099"
	calendarBase  = "http://localhost:18100"
	messengerBase = "http://localhost:18092"
	grooveBase    = "http://localhost:18094"
	gatewayBase   = "http://localhost:18096"
	pushBase      = "http://localhost:18097"
	gatewayWSURL  = "ws://localhost:18096/ws"

	// Тестам выделена СВОЯ база Redis того же dev-инстанса: ключи presence
	// (gw2:presence:*) и дневных капов Groove (gw2:groove:daily:*) не должны
	// пересекаться с dev-данными db0 — id пользователей тестовой БД начинаются
	// с 1 на каждый прогон и совпали бы с dev-остатками. FLUSHDB на старте
	// прогона даёт детерминированные капы/бюджеты. Pub/sub-каналы в Redis
	// глобальны для инстанса (номер БД на них не влияет) — мосту это не мешает.
	testRedisURL = "redis://localhost:6379/9"

	// Dev-ключи PASETO (синхронно с dev.sh).
	pasetoPrivateKey = "68eb779b2f672beb8fcd58d72a81ce1565a1417aed3788d1362bf4faaa3f62ac15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1"
	pasetoPublicKey  = "15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1"
	pasetoRefreshKey = "d525374c4ec7b5e1c5b140fb9c1f4cffd9c3dbf052bb18f2f32bf9f92d9fa05c"
)

var (
	db    *pgxpool.Pool // пул к тестовой БД gw2_apitest
	runID string        // уникальный суффикс прогона для логинов/email
	seq   atomic.Int64  // счётчик уникальности внутри прогона

	repoRoot   string // корень репозитория (…/gw2)
	uploadsDir string // временный UPLOAD_FOLDER прогона (файлы registry/calendar)
)

// uniq — уникальное имя в рамках прогона: детерминированные тесты не должны
// зависеть от остатков прежних прогонов и порядка выполнения.
func uniq(prefix string) string {
	return fmt.Sprintf("%s%s%d", prefix, runID, seq.Add(1))
}

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	// 0. Без dev-инфраструктуры (Redis, mailpit, docker) тесты пропускаются,
	// а не падают.
	if !tcpAlive("localhost:6379") {
		fmt.Println("SKIP: нет dev-инфраструктуры (Redis :6379 недоступен) — cd deploy && docker compose up -d")
		return 0
	}
	if !tcpAlive("localhost:1025") {
		fmt.Println("SKIP: нет dev-инфраструктуры (mailpit SMTP :1025 недоступен)")
		return 0
	}
	if err := ensureTestPostgres(); err != nil {
		fmt.Println("SKIP: не удалось поднять тестовый Postgres:", err)
		return 0
	}

	var err error
	repoRoot, err = findRepoRoot()
	if err != nil {
		fmt.Println("apitest: не найден корень репозитория:", err)
		return 1
	}
	runID = fmt.Sprintf("t%x", time.Now().UnixNano()%0xFFFFFF)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. Чистая тестовая БД на каждый прогон + чистая тестовая база Redis
	// (капы/бюджеты Groove и presence детерминированы в рамках прогона).
	if err := recreateTestDB(ctx); err != nil {
		fmt.Println("apitest: пересоздание БД:", err)
		return 1
	}
	if err := flushTestRedis(); err != nil {
		fmt.Println("apitest: очистка тестовой базы Redis:", err)
		return 1
	}
	// 2. Миграции goose в тестовую БД.
	if out, err := runMigrations(ctx); err != nil {
		fmt.Println("apitest: миграции:", err, "\n", out)
		return 1
	}

	db, err = pgxpool.New(ctx, testDBURL)
	if err != nil {
		fmt.Println("apitest: подключение к тестовой БД:", err)
		return 1
	}
	defer db.Close()

	// 3. Сервисы волны 1: mailsvc (SMTP → mailpit), authsvc, diarysvc;
	// волны 2: tasksvc, registrysvc, calendarsvc.
	uploads, err := os.MkdirTemp("", "gw2-apitest-uploads-*")
	if err != nil {
		fmt.Println("apitest: tempdir:", err)
		return 1
	}
	defer os.RemoveAll(uploads)
	uploadsDir = uploads

	procs := &procGroup{}
	defer procs.stopAll()

	procs.start("mailsvc", filepath.Join(repoRoot, "back-go/mail"), "./cmd/mailsvc", []string{
		"SMTP_HOST=localhost", "SMTP_PORT=1025", "SMTP_TLS=none",
		"SMTP_FROM=noreply@apitest.local",
		"HTTP_ADDR=:18098", "GRPC_ADDR=:19098",
	})
	procs.start("authsvc", filepath.Join(repoRoot, "back-go/auth"), "./cmd/authsvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PRIVATE_KEY=" + pasetoPrivateKey,
		"PASETO_REFRESH_KEY=" + pasetoRefreshKey,
		"UPLOAD_FOLDER=" + uploads,
		"MAIL_GRPC_ADDR=localhost:19098",
		"APP_PUBLIC_BASE_URL=http://localhost:5173",
		"HTTP_ADDR=:18091",
	})
	procs.start("diarysvc", filepath.Join(repoRoot, "back-go/diary"), "./cmd/diarysvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PUBLIC_KEY=" + pasetoPublicKey,
		"HTTP_ADDR=:18101",
	})
	// tasksvc: groovesvc/aisvc НЕ поднимаем нарочно — хуки геймификации
	// fire-and-forget, а AI fail-open (поиск падает в LIKE); сервис обязан
	// переживать их недоступность. Ключ Fernet YouGile — dev-ключ из dev.sh.
	procs.start("tasksvc", filepath.Join(repoRoot, "back-go/tasks"), "./cmd/tasksvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PUBLIC_KEY=" + pasetoPublicKey,
		"GROOVE_GRPC_ADDR=localhost:19094",
		"AI_GRPC_ADDR=localhost:19093",
		"YOUGILE_ENC_KEY=CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc=",
		"HTTP_ADDR=:18095",
	})
	procs.start("registrysvc", filepath.Join(repoRoot, "back-go/registry"), "./cmd/registrysvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PUBLIC_KEY=" + pasetoPublicKey,
		"UPLOAD_FOLDER=" + uploads,
		"HTTP_ADDR=:18099",
	})
	procs.start("calendarsvc", filepath.Join(repoRoot, "back-go/calendar"), "./cmd/calendarsvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PUBLIC_KEY=" + pasetoPublicKey,
		"UPLOAD_FOLDER=" + uploads,
		"HTTP_ADDR=:18100",
	})
	// Волна 3: msgsvc, groovesvc, pushsvc. Порядок env-зависимостей:
	//   - msgsvc зовёт groovesvc по gRPC (:19094) для ответа Грувика в pet-чате;
	//   - groovesvc зовёт msgsvc по gRPC (:19092) для публикации ответа Грувика,
	//     а AI (:19093) НЕ поднят — s.ai.Enabled даёт false, Грувик отвечает
	//     статичной офлайн-репликой (fail-open, не падаем);
	//   - pushsvc без FIREBASE_* — отправка выключена (no-op sender), REST живёт.
	// Все gRPC-клиенты дозваниваются лениво (grpc.NewClient), поэтому порядок
	// старта внутри волны не важен.
	procs.start("msgsvc", filepath.Join(repoRoot, "back-go/messenger"), "./cmd/msgsvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PUBLIC_KEY=" + pasetoPublicKey,
		"UPLOAD_FOLDER=" + uploads,
		"GROOVE_GRPC_ADDR=localhost:19094",
		"GRPC_ADDR=:19092",
		"HTTP_ADDR=:18092",
	})
	procs.start("groovesvc", filepath.Join(repoRoot, "back-go/groove"), "./cmd/groovesvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PUBLIC_KEY=" + pasetoPublicKey,
		"AI_GRPC_ADDR=localhost:19093",
		"MESSENGER_GRPC_ADDR=localhost:19092",
		"GRPC_ADDR=:19094",
		"HTTP_ADDR=:18094",
	})
	procs.start("pushsvc", filepath.Join(repoRoot, "back-go/push"), "./cmd/pushsvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PUBLIC_KEY=" + pasetoPublicKey,
		// Явно глушим FCM, даже если ключи есть в окружении разработчика:
		// тесты не должны слать реальные пуши.
		"FIREBASE_CREDENTIALS_JSON=",
		"GOOGLE_APPLICATION_CREDENTIALS=",
		"HTTP_ADDR=:18097",
	})
	// gatewaysvc — realtime-шлюз: WS /ws, exact REST /api/messenger/presence,
	// мост Redis-каналов gw2:*:events → WS-комнаты. callsvc (:19090) НЕ поднят —
	// команды call:* обязаны отвечать call:error CALLS_UNAVAILABLE, не падать.
	procs.start("gatewaysvc", filepath.Join(repoRoot, "back-go/gateway"), "./cmd/gatewaysvc", []string{
		"DATABASE_URL=" + testDBURL,
		"REDIS_URL=" + testRedisURL,
		"PASETO_PUBLIC_KEY=" + pasetoPublicKey,
		"CALLS_GRPC_ADDR=localhost:19090",
		"HTTP_ADDR=:18096",
	})

	// 4. Ждём готовности каждого сервиса (retry до 30с).
	for _, hc := range []string{
		authBase + "/healthz", diaryBase + "/healthz", "http://localhost:18098/healthz",
		tasksBase + "/healthz", registryBase + "/healthz", calendarBase + "/healthz",
		messengerBase + "/healthz", grooveBase + "/healthz", pushBase + "/healthz",
		gatewayBase + "/healthz",
	} {
		if err := waitHealthz(hc, 30*time.Second); err != nil {
			fmt.Println("apitest:", err)
			procs.dumpLogs()
			return 1
		}
	}

	code := m.Run()
	if code != 0 {
		procs.dumpLogs()
	}
	return code
}

func tcpAlive(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for d := dir; d != "/"; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "back-go", "go.work")); err == nil {
			return d, nil
		}
	}
	return "", fmt.Errorf("go.work не найден вверх от %s", dir)
}

// ensureTestPostgres — выделенный Postgres-контейнер тестов: создаёт при
// отсутствии, стартует при остановке, ждёт готовности. Контейнер переживает
// прогоны (быстрее), БД gw2_apitest пересоздаётся на каждый прогон.
func ensureTestPostgres() error {
	if !tcpAlive("localhost:" + pgPort) {
		// `docker start` для существующего, иначе — создать.
		if err := exec.Command("docker", "start", pgContainer).Run(); err != nil {
			out, err := exec.Command("docker", "run", "-d", "--name", pgContainer,
				"-e", "POSTGRES_USER=grovework",
				"-e", "POSTGRES_PASSWORD=grovework_local",
				"-e", "POSTGRES_DB=grovework",
				"-p", "127.0.0.1:"+pgPort+":5432",
				pgImage).CombinedOutput()
			if err != nil {
				return fmt.Errorf("docker run: %v: %s", err, out)
			}
		}
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if exec.Command("docker", "exec", pgContainer, "pg_isready", "-U", "grovework").Run() == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("postgres в %s не готов за 60с", pgContainer)
}

func recreateTestDB(ctx context.Context) error {
	admin, err := pgxpool.New(ctx, pgAdminURL)
	if err != nil {
		return err
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS `+testDBName+` WITH (FORCE)`); err != nil {
		return err
	}
	_, err = admin.Exec(ctx, `CREATE DATABASE `+testDBName)
	return err
}

// flushTestRedis — FLUSHDB выделенной тестовой базы Redis (номер из
// testRedisURL) сырыми inline-командами RESP: тянуть redis-клиент в тесты
// ради двух команд не стоит.
func flushTestRedis() error {
	u, err := url.Parse(testRedisURL)
	if err != nil {
		return err
	}
	dbNum := strings.TrimPrefix(u.Path, "/")
	conn, err := net.DialTimeout("tcp", u.Host, 3*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := fmt.Fprintf(conn, "SELECT %s\r\nFLUSHDB\r\n", dbNum); err != nil {
		return err
	}
	// Два ответа "+OK\r\n"; любой -ERR — проблема конфигурации.
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	if resp := string(buf[:n]); strings.HasPrefix(resp, "-") {
		return fmt.Errorf("redis: %s", strings.TrimSpace(resp))
	}
	return nil
}

func runMigrations(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/migrate")
	cmd.Dir = filepath.Join(repoRoot, "back-go", "migrate")
	cmd.Env = append(os.Environ(), "DATABASE_URL="+testDBURL)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ── Управление процессами сервисов ───────────────────────────────

type proc struct {
	name string
	cmd  *exec.Cmd
	logs *bytes.Buffer
}

type procGroup struct{ procs []*proc }

// start — запускает сервис `go run` в собственной process group, чтобы при
// остановке убить и потомков (go run порождает дочерний бинарь).
func (g *procGroup) start(name, dir, pkg string, env []string) {
	logs := &bytes.Buffer{}
	cmd := exec.Command("go", "run", pkg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = logs
	cmd.Stderr = logs
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		fmt.Printf("apitest: старт %s: %v\n", name, err)
		return
	}
	g.procs = append(g.procs, &proc{name: name, cmd: cmd, logs: logs})
}

func (g *procGroup) stopAll() {
	for _, p := range g.procs {
		if p.cmd.Process != nil {
			// Убиваем всю process group (как cleanup в dev.sh).
			_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)
		}
	}
	deadline := time.After(3 * time.Second)
	done := make(chan struct{})
	go func() {
		for _, p := range g.procs {
			_ = p.cmd.Wait()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-deadline:
		for _, p := range g.procs {
			if p.cmd.Process != nil {
				_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
			}
		}
	}
}

func (g *procGroup) dumpLogs() {
	for _, p := range g.procs {
		if p.logs.Len() > 0 {
			fmt.Printf("── логи %s ──\n%s\n", p.name, tail(p.logs.String(), 8000))
		}
	}
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

func waitHealthz(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("сервис не поднялся за %s: %s", timeout, url)
}

// ── HTTP-хелперы ─────────────────────────────────────────────────

// апи-ответ: статус + разобранный JSON (объект) + сырое тело + заголовки/куки.
type apiResp struct {
	Status  int
	JSON    map[string]any
	Raw     []byte
	Header  http.Header
	Cookies []*http.Cookie
}

// Str — строковое поле верхнего уровня JSON-ответа.
func (r apiResp) Str(key string) string {
	v, _ := r.JSON[key].(string)
	return v
}

// Num — числовое поле верхнего уровня (JSON number → float64).
func (r apiResp) Num(key string) float64 {
	v, _ := r.JSON[key].(float64)
	return v
}

// Bool — булево поле верхнего уровня.
func (r apiResp) Bool(key string) bool {
	v, _ := r.JSON[key].(bool)
	return v
}

// List — поле-массив верхнего уровня.
func (r apiResp) List(key string) []any {
	v, _ := r.JSON[key].([]any)
	return v
}

// Cookie — значение куки из Set-Cookie ответа ("" — не установлена).
func (r apiResp) Cookie(name string) string {
	for _, c := range r.Cookies {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

type svcClient struct {
	base string
}

var authAPI = &svcClient{base: authBase}
var diaryAPI = &svcClient{base: diaryBase}
var tasksAPI = &svcClient{base: tasksBase}
var registryAPI = &svcClient{base: registryBase}
var calendarAPI = &svcClient{base: calendarBase}
var messengerAPI = &svcClient{base: messengerBase}
var grooveAPI = &svcClient{base: grooveBase}
var pushAPI = &svcClient{base: pushBase}

type reqOpt func(*http.Request)

func withCookie(name, value string) reqOpt {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: name, Value: value}) }
}

func withRawBody(contentType string) reqOpt {
	return func(r *http.Request) { r.Header.Set("Content-Type", contentType) }
}

// doJSON — запрос с JSON-телом (body может быть nil, готовым []byte с сырыми
// байтами или любым сериализуемым значением) и Bearer-токеном (пустой — без
// авторизации). Ответ разбирается как JSON-объект, если это возможно.
func (c *svcClient) doJSON(t *testing.T, method, path, token string, body any, opts ...reqOpt) apiResp {
	t.Helper()
	var reader io.Reader
	if body != nil {
		switch b := body.(type) {
		case []byte:
			reader = bytes.NewReader(b)
		case string:
			reader = strings.NewReader(b)
		default:
			buf, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			reader = bytes.NewReader(buf)
		}
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, opt := range opts {
		opt(req)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body %s %s: %v", method, path, err)
	}
	out := apiResp{Status: resp.StatusCode, Raw: raw, Header: resp.Header, Cookies: resp.Cookies()}
	var parsed map[string]any
	if json.Unmarshal(raw, &parsed) == nil {
		out.JSON = parsed
	}
	return out
}

// doMultipart — загрузка файла полем "file" (multipart/form-data), как это
// делает фронт для картинок/файлов реестров и календарей.
func (c *svcClient) doMultipart(t *testing.T, path, token, fileName string, content []byte) apiResp {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	fw, err := w.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("multipart: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("multipart write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("multipart close: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.base+path, buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body %s: %v", path, err)
	}
	out := apiResp{Status: resp.StatusCode, Raw: raw, Header: resp.Header, Cookies: resp.Cookies()}
	var parsed map[string]any
	if json.Unmarshal(raw, &parsed) == nil {
		out.JSON = parsed
	}
	return out
}

// jsonUnmarshal — обёртка (чтобы тестам не импортировать encoding/json ради
// массивов верхнего уровня).
func jsonUnmarshal(raw []byte, out any) error { return json.Unmarshal(raw, out) }

// urlQuery — значение query-параметра с экранированием.
func urlQuery(s string) string { return url.QueryEscape(s) }

// requireStatus — фейл теста с телом ответа при неожиданном статусе.
func requireStatus(t *testing.T, r apiResp, want int, what string) {
	t.Helper()
	if r.Status != want {
		t.Fatalf("%s: статус %d, ожидался %d; тело: %s", what, r.Status, want, r.Raw)
	}
}

// requireError — статус + код ошибки в поле "error".
func requireError(t *testing.T, r apiResp, wantStatus int, wantCode, what string) {
	t.Helper()
	requireStatus(t, r, wantStatus, what)
	if got := r.Str("error"); got != wantCode {
		t.Fatalf("%s: код ошибки %q, ожидался %q; тело: %s", what, got, wantCode, r.Raw)
	}
}

// ── Доступ к тестовой БД ─────────────────────────────────────────

// dbCtx — контекст для прямых запросов в тестовую БД.
func dbCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// verificationFor — код и токен подтверждения email пользователя (из БД —
// проще и надёжнее парсинга писем mailpit).
func verificationFor(t *testing.T, email string) (code, token string) {
	t.Helper()
	err := db.QueryRow(dbCtx(t), `
		SELECT v.code, v.token FROM email_verifications v
		JOIN users u ON u.id = v.user_id WHERE lower(u.email) = lower($1)`, email).
		Scan(&code, &token)
	if err != nil {
		t.Fatalf("verification для %s: %v", email, err)
	}
	return code, token
}

// resetTokenFor — токен сброса пароля пользователя из БД.
func resetTokenFor(t *testing.T, email string) string {
	t.Helper()
	var token string
	err := db.QueryRow(dbCtx(t), `
		SELECT r.token FROM password_resets r
		JOIN users u ON u.id = r.user_id WHERE lower(u.email) = lower($1)`, email).
		Scan(&token)
	if err != nil {
		t.Fatalf("reset-токен для %s: %v", email, err)
	}
	return token
}

// inviteTokenFor — токен email-приглашения в компанию из БД.
func inviteTokenFor(t *testing.T, companyID int64, email string) string {
	t.Helper()
	var token string
	err := db.QueryRow(dbCtx(t),
		`SELECT token FROM company_invites WHERE company_id = $1 AND lower(email) = lower($2)`,
		companyID, email).Scan(&token)
	if err != nil {
		t.Fatalf("invite-токен для %s: %v", email, err)
	}
	return token
}
