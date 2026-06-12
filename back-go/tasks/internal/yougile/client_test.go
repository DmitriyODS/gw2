package yougile

// Порт back/tests/test_yougile_client.py: парсер ссылок, Fernet-крипто и
// тонкий HTTP-клиент (без сети — фейковый RoundTripper).

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testEncKey = "CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc="

// ── parser ───────────────────────────────────────────────────────────────

func TestParseURLHashQuery(t *testing.T) {
	r := ParseTaskURL("https://ru.yougile.com/team/9347006b-dc75-4550-97d5-3008ba00d4a0/" +
		"#tasks?task=4f6f0391-0f94-4d30-9b0e-99430a36d4fb")
	if r == nil {
		t.Fatal("ссылка не разобрана")
	}
	if r.TaskID != "4f6f0391-0f94-4d30-9b0e-99430a36d4fb" {
		t.Fatalf("task_id = %q", r.TaskID)
	}
	if r.CompanyID != "9347006b-dc75-4550-97d5-3008ba00d4a0" {
		t.Fatalf("company_id = %q", r.CompanyID)
	}
}

func TestParseURLTaskHash(t *testing.T) {
	r := ParseTaskURL("https://yougile.com/board/abc/#task-4F6F0391-0F94-4D30-9B0E-99430A36D4FB")
	if r == nil || r.TaskID != "4f6f0391-0f94-4d30-9b0e-99430a36d4fb" {
		t.Fatalf("r = %+v", r)
	}
}

func TestParseURLShortFormat(t *testing.T) {
	r := ParseTaskURL("https://ru.yougile.com/team/ed7037760782/#OIP1-2454")
	if r == nil {
		t.Fatal("короткая ссылка не разобрана")
	}
	if r.TaskID != "" || r.ShortTaskID != "OIP1-2454" || r.ShortTeamID != "ed7037760782" {
		t.Fatalf("r = %+v", r)
	}
}

func TestParseURLRejectsNonYougile(t *testing.T) {
	for _, raw := range []string{
		"https://example.com/4f6f0391-0f94-4d30-9b0e-99430a36d4fb",
		"not a url",
		"",
	} {
		if r := ParseTaskURL(raw); r != nil {
			t.Fatalf("%q: ожидался nil, получено %+v", raw, r)
		}
	}
}

// ── crypto ───────────────────────────────────────────────────────────────

func TestCryptoRoundtrip(t *testing.T) {
	c := NewCipher(testEncKey)
	const key = "H6HngIA816fpIhY7tBvWx/it3YbVvEt/33Sk8afA39MCR9a"
	enc, err := c.EncryptKey(key)
	if err != nil || len(enc) == 0 {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := c.DecryptKey(enc)
	if err != nil || got != key {
		t.Fatalf("decrypt: %q, %v", got, err)
	}
	if got, err := c.DecryptKey(nil); err != nil || got != "" {
		t.Fatalf("decrypt(nil): %q, %v", got, err)
	}
}

func TestCryptoMisconfigured(t *testing.T) {
	c := NewCipher("")
	if _, err := c.EncryptKey("x"); err != ErrMisconfigured {
		t.Fatalf("encrypt без ключа: %v", err)
	}
	if _, err := c.DecryptKey([]byte("junk")); err != ErrMisconfigured {
		t.Fatalf("decrypt без ключа: %v", err)
	}
}

func TestFingerprintLast4(t *testing.T) {
	if MakeFingerprint("abcdef0123") != "0123" {
		t.Fatal("fingerprint != last4")
	}
	if MakeFingerprint("") != "" {
		t.Fatal("fingerprint пустой строки")
	}
}

// ── фейковый транспорт ───────────────────────────────────────────────────

type fakeCall struct {
	Method string
	URL    string
	Header http.Header
	Body   map[string]any
	Query  map[string]string
}

type fakeTransport struct {
	queue []*http.Response
	calls []fakeCall
}

func resp(status int, payload any) *http.Response {
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     http.Header{},
	}
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	call := fakeCall{Method: req.Method, URL: req.URL.String(), Header: req.Header.Clone(),
		Query: map[string]string{}}
	for k, v := range req.URL.Query() {
		call.Query[k] = v[0]
	}
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		if len(data) > 0 {
			_ = json.Unmarshal(data, &call.Body)
		}
	}
	f.calls = append(f.calls, call)
	if len(f.queue) == 0 {
		panic("закончились запасы фейковых ответов")
	}
	out := f.queue[0]
	f.queue = f.queue[1:]
	return out, nil
}

func newTestClient(key string, responses ...*http.Response) (*Client, *fakeTransport) {
	tr := &fakeTransport{queue: responses}
	c := NewClient(key,
		WithHTTPClient(&http.Client{Transport: tr}),
		WithSleep(func(time.Duration) {})) // отключаем backoff в тестах
	return c, tr
}

// ── client ───────────────────────────────────────────────────────────────

func TestClientAnonymousEndpointsDontSendBearer(t *testing.T) {
	c, tr := newTestClient("", resp(200, map[string]any{
		"content": []any{map[string]any{"id": "c1", "name": "X"}},
	}))
	items, err := c.ListCompanies("a@b.c", "pw")
	if err != nil || len(items) != 1 || items[0]["id"] != "c1" {
		t.Fatalf("items = %v, err = %v", items, err)
	}
	if tr.calls[0].Header.Get("Authorization") != "" {
		t.Fatal("анонимный эндпоинт ушёл с Bearer")
	}
}

func TestClientBearerForAuthenticatedEndpoint(t *testing.T) {
	c, tr := newTestClient("K", resp(200, map[string]any{"id": "u1"}))
	if _, err := c.Me(); err != nil {
		t.Fatalf("me: %v", err)
	}
	if tr.calls[0].Header.Get("Authorization") != "Bearer K" {
		t.Fatalf("Authorization = %q", tr.calls[0].Header.Get("Authorization"))
	}
}

func TestClientAuth401Raises(t *testing.T) {
	c, _ := newTestClient("K", resp(401, nil))
	_, err := c.Me()
	if !IsAuth(err) {
		t.Fatalf("ожидалась auth-ошибка, получено %v", err)
	}
}

func TestClient429RetriesThenRaises(t *testing.T) {
	c, tr := newTestClient("K", resp(429, nil), resp(429, nil), resp(429, nil))
	_, err := c.Me()
	var e *Error
	if !asError(err, &e) || !e.RateLimited {
		t.Fatalf("ожидался rate-limit, получено %v", err)
	}
	if len(tr.calls) != 3 {
		t.Fatalf("попыток = %d", len(tr.calls))
	}
}

func TestClientPaginationCollectsPages(t *testing.T) {
	page1 := make([]any, 1000)
	for i := range page1 {
		page1[i] = map[string]any{"id": "p", "title": "t"}
	}
	page2 := make([]any, 500)
	for i := range page2 {
		page2[i] = map[string]any{"id": "p", "title": "t"}
	}
	c, tr := newTestClient("K",
		resp(200, map[string]any{"content": page1}),
		resp(200, map[string]any{"content": page2}))
	items, err := c.ListProjects(2000)
	if err != nil || len(items) != 1500 {
		t.Fatalf("items = %d, err = %v", len(items), err)
	}
	if tr.calls[1].Query["offset"] != "1000" {
		t.Fatalf("второй запрос offset = %q", tr.calls[1].Query["offset"])
	}
}

func TestCreateKeyReturnsString(t *testing.T) {
	c, _ := newTestClient("", resp(201, map[string]any{"key": "KKK"}))
	key, err := c.CreateKey("a@b.c", "pw", "comp")
	if err != nil || key != "KKK" {
		t.Fatalf("key = %q, err = %v", key, err)
	}
}

func TestCreateKeyUnexpectedPayloadRaises(t *testing.T) {
	c, _ := newTestClient("", resp(201, map[string]any{}))
	if _, err := c.CreateKey("a@b.c", "pw", "comp"); err == nil {
		t.Fatal("ожидалась ошибка на пустой ответ /auth/keys")
	}
}

// ── FindTaskByShortID ────────────────────────────────────────────────────

func TestFindShortIDTopLevel(t *testing.T) {
	c, _ := newTestClient("K", resp(200, map[string]any{"content": []any{
		map[string]any{"id": "t1", "idTaskProject": "OIP1-1"},
		map[string]any{"id": "t2", "idTaskProject": "OIP1-2"},
	}}))
	found, err := c.FindTaskByShortID("b", "oip1-2", []string{"col1"})
	if err != nil || found == nil || found["id"] != "t2" {
		t.Fatalf("found = %v, err = %v", found, err)
	}
}

func TestFindShortIDInSubtask(t *testing.T) {
	// Колонка отдаёт родителя без совпадения, но с subtasks; подзадача
	// находится отдельным GET /tasks/{id}.
	c, tr := newTestClient("K",
		resp(200, map[string]any{"content": []any{
			map[string]any{"id": "parent", "idTaskProject": "OIP1-1",
				"subtasks": []any{"sub1", "sub2"}},
		}}),
		resp(200, map[string]any{"id": "sub1", "idTaskProject": "OIP1-7",
			"subtasks": []any{"sub3"}}))
	found, err := c.FindTaskByShortID("b", "OIP1-7", []string{"col1"})
	if err != nil || found == nil || found["id"] != "sub1" {
		t.Fatalf("found = %v, err = %v", found, err)
	}
	if !strings.HasSuffix(strings.SplitN(tr.calls[1].URL, "?", 2)[0], "/tasks/sub1") {
		t.Fatalf("второй запрос: %s", tr.calls[1].URL)
	}
}

func TestFindShortIDInNestedSubtask(t *testing.T) {
	c, _ := newTestClient("K",
		resp(200, map[string]any{"content": []any{
			map[string]any{"id": "parent", "idTaskProject": "OIP1-1",
				"subtasks": []any{"sub1"}},
		}}),
		resp(200, map[string]any{"id": "sub1", "idTaskProject": "OIP1-2",
			"subtasks": []any{"sub2"}}),
		resp(200, map[string]any{"id": "sub2", "idTaskProject": "OIP1-3"}))
	found, err := c.FindTaskByShortID("b", "OIP1-3", []string{"col1"})
	if err != nil || found == nil || found["id"] != "sub2" {
		t.Fatalf("found = %v, err = %v", found, err)
	}
}

func TestFindShortIDSubtaskFetchErrorSkipped(t *testing.T) {
	// Битая подзадача (404) не валит поиск — идём дальше по очереди.
	c, _ := newTestClient("K",
		resp(200, map[string]any{"content": []any{
			map[string]any{"id": "parent", "idTaskProject": "OIP1-1",
				"subtasks": []any{"bad", "good"}},
		}}),
		resp(404, map[string]any{"message": "nope"}),
		resp(200, map[string]any{"id": "good", "idTaskProject": "OIP1-9"}))
	found, err := c.FindTaskByShortID("b", "OIP1-9", []string{"col1"})
	if err != nil || found == nil || found["id"] != "good" {
		t.Fatalf("found = %v, err = %v", found, err)
	}
}

func TestFindShortIDNotFoundReturnsNil(t *testing.T) {
	c, _ := newTestClient("K", resp(200, map[string]any{"content": []any{
		map[string]any{"id": "t1", "idTaskProject": "OIP1-1"},
	}}))
	found, err := c.FindTaskByShortID("b", "OIP1-99", []string{"col1"})
	if err != nil || found != nil {
		t.Fatalf("found = %v, err = %v", found, err)
	}
}

func asError(err error, target **Error) bool {
	e, ok := err.(*Error)
	if ok {
		*target = e
	}
	return ok
}
