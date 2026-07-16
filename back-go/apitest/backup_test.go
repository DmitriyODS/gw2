package apitest

// API-тесты резервной копии: полный цикл экспорт → порча данных →
// восстановление, и — главное — покрытие схемы разделами.
//
// Карта разделов (domain.BackupSections в authsvc) статическая, а схему ведут
// миграции: разъезжаются они молча. Забытая в разделе таблица-спутник особенно
// опасна при ВОССТАНОВЛЕНИИ раздела — импорт делает TRUNCATE … CASCADE, каскад
// FK вычистит спутника, а данных его в архиве раздела нет: потеря без следа.
// Поэтому проверяем не константы (пакет authsvc — internal, другой модуль), а
// поведение ручек экспорта.

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"testing"
)

// backupExcluded — таблицы, которые бэкап не берёт намеренно (≡ domain.
// BackupExcluded authsvc): транзиентные коды и перегенерируемые эмбеддинги.
var backupExcluded = map[string]bool{
	"email_verifications": true,
	"password_resets":     true,
	"task_embeddings":     true,
	"goose_db_version":    true,
}

// backupSectionKeys — разделы выбора (≡ domain.BackupSections + SectionOther;
// подписи — на фронте, front/src/utils/backupSections.js).
var backupSectionKeys = []string{
	"auth", "companies", "tasks", "registry", "calendar", "diary", "notes",
	"messenger", "calls", "groove", "portal", "ai", "integration",
}

// dbTables — таблицы public-схемы тестовой БД (её ведут те же миграции, что и прод).
func dbTables(t *testing.T) []string {
	t.Helper()
	rows, err := db.Query(dbCtx(t), `SELECT tablename FROM pg_tables WHERE schemaname = 'public'`)
	if err != nil {
		t.Fatalf("список таблиц: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("скан таблицы: %v", err)
		}
		out = append(out, name)
	}
	return out
}

// exportArchive — выгрузить бэкап (sections пуст — вся база) и разобрать data.json.
func exportArchive(t *testing.T, root *actor, sections ...string) map[string]json.RawMessage {
	t.Helper()
	path := "/api/backup/export"
	if len(sections) > 0 {
		path += "?sections=" + url.QueryEscape(joinComma(sections))
	}
	r := authAPI.doJSON(t, http.MethodGet, path, root.Token, nil)
	requireStatus(t, r, 200, "экспорт "+path)

	zr, err := zip.NewReader(bytes.NewReader(r.Raw), int64(len(r.Raw)))
	if err != nil {
		t.Fatalf("архив не читается: %v", err)
	}
	for _, f := range zr.File {
		if f.Name != "data.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("data.json: %v", err)
		}
		defer rc.Close()
		var archive struct {
			Version  int                        `json:"version"`
			Sections []string                   `json:"sections"`
			Tables   map[string]json.RawMessage `json:"tables"`
		}
		if err := json.NewDecoder(rc).Decode(&archive); err != nil {
			t.Fatalf("разбор data.json: %v", err)
		}
		if archive.Version == 0 {
			t.Fatal("архив без версии")
		}
		return archive.Tables
	}
	t.Fatal("в архиве нет data.json")
	return nil
}

func joinComma(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

func countRows(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(dbCtx(t), `SELECT count(*) FROM `+table).Scan(&n); err != nil {
		t.Fatalf("count(%s): %v", table, err)
	}
	return n
}

// Полный экспорт обязан нести КАЖДУЮ таблицу схемы, кроме денлиста: бэкап
// схемо-независим, и новая таблица не должна выпадать из него молча.
func TestBackupExportCoversWholeSchema(t *testing.T) {
	root := newSuperAdmin(t)
	tables := exportArchive(t, root)

	var missing []string
	for _, tbl := range dbTables(t) {
		if backupExcluded[tbl] {
			if _, ok := tables[tbl]; ok {
				t.Errorf("таблица %q в денлисте, но попала в архив", tbl)
			}
			continue
		}
		if _, ok := tables[tbl]; !ok {
			missing = append(missing, tbl)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("таблиц нет в полном бэкапе: %v", missing)
	}
}

// Явные разделы обязаны покрывать всю схему: «Прочее» — страховка от потери
// данных, а не место для новых таблиц. Таблица, не попавшая в свой раздел,
// не восстановится вместе с ним (а каскад TRUNCATE её вычистит).
func TestBackupSectionsCoverEveryTable(t *testing.T) {
	root := newSuperAdmin(t)

	covered := map[string]bool{}
	for _, key := range backupSectionKeys {
		for tbl := range exportArchive(t, root, key) {
			covered[tbl] = true
		}
	}

	var orphans []string
	for _, tbl := range dbTables(t) {
		if !backupExcluded[tbl] && !covered[tbl] {
			orphans = append(orphans, tbl)
		}
	}
	if len(orphans) > 0 {
		sort.Strings(orphans)
		t.Fatalf("таблицы не покрыты ни одним явным разделом (уедут в «Прочее» и не "+
			"восстановятся вместе со своим разделом): %v", orphans)
	}

	// Зеркало той же проверки со стороны «Прочего»: раз явные разделы
	// покрывают всё, псевдо-раздел other обязан быть пуст.
	if extra := exportArchive(t, root, "other"); len(extra) > 0 {
		var names []string
		for tbl := range extra {
			names = append(names, tbl)
		}
		sort.Strings(names)
		t.Fatalf("раздел «Прочее» не пуст: %v", names)
	}
}

// Раздел питомцев обязан выгружаться целиком: экономика грувика — это не
// только pets, но и банк с магазином, историей и счётчиками признания.
func TestBackupGrooveSectionCarriesPetEconomy(t *testing.T) {
	root := newSuperAdmin(t)
	tables := exportArchive(t, root, "groove")

	for _, tbl := range []string{"pets", "pet_strokes", "pet_activity_log", "pet_kudos_ledger",
		"pet_kudos_weekly", "pet_shop_purchases", "pet_bank_goals"} {
		if _, ok := tables[tbl]; !ok {
			t.Errorf("раздел «Питомцы» не несёт таблицу %q", tbl)
		}
	}
	// Портал — свой раздел, в питомцев он попадать не должен.
	if _, ok := tables["portal_posts"]; ok {
		t.Error("раздел «Питомцы» тянет за собой портал")
	}
}

// Сквозной цикл: наполняем компанию данными новых механик (грувик с
// потребностями и болезнью, выписка банка, ветка комментариев с лайком),
// выгружаем архив, ломаем данные и восстанавливаем — состояние возвращается.
func TestBackupExportImportRoundTrip(t *testing.T) {
	root := newSuperAdmin(t)
	admin, member, _ := petsCompany(t)

	petsAPI.doJSON(t, http.MethodGet, "/api/pets/pet", member.Token, nil)
	grantKudos(t, member.ID, 50)
	setNeed(t, member.ID, "need_hygiene", 0)
	makeSick(t, member.ID, "grime", 1)
	r := petsAPI.doJSON(t, http.MethodPost, "/api/pets/bath", member.Token, nil)
	requireStatus(t, r, 200, "купание (даёт запись выписки)")

	postID := createPost(t, admin, "Пост для бэкапа")
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID),
		member.Token, map[string]any{"text": "корневой"})
	requireStatus(t, r, 201, "комментарий")
	rootComment := int64(r.Num("id"))
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID),
		admin.Token, map[string]any{"text": "ответ", "reply_to_id": rootComment})
	requireStatus(t, r, 201, "ответ на комментарий")
	r = portalAPI.doJSON(t, http.MethodPost,
		fmt.Sprintf("/api/portal/comments/%d/like", rootComment), admin.Token, nil)
	requireStatus(t, r, 200, "лайк комментария")

	counted := []string{"users", "pets", "pet_kudos_ledger", "portal_posts",
		"portal_comments", "portal_comment_likes"}
	before := map[string]int{}
	for _, tbl := range counted {
		before[tbl] = countRows(t, tbl)
	}
	var hygieneBefore int
	if err := db.QueryRow(dbCtx(t),
		`SELECT need_hygiene FROM pets WHERE user_id=$1`, member.ID).Scan(&hygieneBefore); err != nil {
		t.Fatalf("шкала до бэкапа: %v", err)
	}

	// Экспорт всей базы.
	exp := authAPI.doJSON(t, http.MethodGet, "/api/backup/export", root.Token, nil)
	requireStatus(t, exp, 200, "экспорт бэкапа")
	raw := exp.Raw

	// Ломаем: сносим обсуждение, выписку и шкалы питомца.
	for _, q := range []string{
		`DELETE FROM portal_comments`,
		`DELETE FROM pet_kudos_ledger`,
		`UPDATE pets SET need_hygiene = 0, need_satiety = 0`,
	} {
		if _, err := db.Exec(dbCtx(t), q); err != nil {
			t.Fatalf("порча данных (%s): %v", q, err)
		}
	}
	if countRows(t, "portal_comments") != 0 {
		t.Fatal("данные не удалились — тест бессмыслен")
	}

	// Восстановление.
	imp := authAPI.doMultipart(t, "/api/backup/import", root.Token, "backup.zip", raw)
	requireStatus(t, imp, 200, "импорт бэкапа")

	for _, tbl := range counted {
		if got := countRows(t, tbl); got != before[tbl] {
			t.Errorf("после восстановления в %s строк %d, ожидалось %d", tbl, got, before[tbl])
		}
	}

	// Питомец вернул шкалы и болезнь, обсуждение — дерево и лайки.
	var hygiene int
	var ailment *string
	if err := db.QueryRow(dbCtx(t),
		`SELECT need_hygiene, ailment FROM pets WHERE user_id=$1`, member.ID).
		Scan(&hygiene, &ailment); err != nil {
		t.Fatalf("питомец после восстановления: %v", err)
	}
	if hygiene != hygieneBefore {
		t.Errorf("шкала чистоты = %d, ожидалась %d", hygiene, hygieneBefore)
	}
	if ailment == nil || *ailment != "grime" {
		t.Errorf("болезнь не восстановилась: %v", ailment)
	}
	var replies, likes int
	if err := db.QueryRow(dbCtx(t),
		`SELECT count(*) FROM portal_comments WHERE reply_to_id IS NOT NULL`).Scan(&replies); err != nil {
		t.Fatalf("ответы: %v", err)
	}
	if err := db.QueryRow(dbCtx(t),
		`SELECT count(*) FROM portal_comment_likes WHERE comment_id=$1`, rootComment).Scan(&likes); err != nil {
		t.Fatalf("лайки: %v", err)
	}
	if replies == 0 || likes == 0 {
		t.Errorf("ветка ответов/лайки не восстановились: ответов %d, лайков %d", replies, likes)
	}

	// Восстановленной сессией можно работать: токен супер-админа переживает
	// импорт (пользователи вернулись теми же id).
	r = authAPI.doJSON(t, http.MethodGet, "/api/users/me", root.Token, nil)
	requireStatus(t, r, 200, "профиль после восстановления")
}
