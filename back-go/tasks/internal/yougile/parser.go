package yougile

import (
	"net/url"
	"regexp"
	"strings"
)

// Извлечение YouGile task_id из URL карточки.
//
// YouGile показывает три практических формата ссылок:
//
//  1. Длинный (старый):
//     `https://ru.yougile.com/team/<companyUUID>/#tasks?task=<taskUUID>`
//  2. Длинный с board (редко):
//     `https://yougile.com/team/<companyUUID>/?board=<boardUUID>#task-<taskUUID>`
//  3. Короткий (сейчас дефолт в адресной строке и кнопке «Скопировать»):
//     `https://yougile.com/team/<shortTeamId>/#<idTaskProject>`
//     Где `shortTeamId` — последние 12 hex-символов UUID компании, а
//     `idTaskProject` — человекочитаемый id карточки `OIP1-2454`.
//
// Для (1)/(2) парсер сразу отдаёт TaskID (UUID). Для (3) UUID в URL нет —
// отдаём ShortTaskID и ShortTeamID, а вызывающий код резолвит UUID через
// YouGile API (см. Client.FindTaskByShortID).

var (
	uuidRE      = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	teamUUIDRE  = regexp.MustCompile(`(?i)/team/([0-9a-f-]{36})/?`)
	teamShortRE = regexp.MustCompile(`(?i)/team/([0-9a-f]{12})/?`)
	// `OIP1-2454`, `ABC-1`, `Z9-100500` — буквы (минимум одна) + опц. цифры,
	// дефис, цифры. Строго ALPHA-NUM до дефиса (без _), чтобы не словить
	// UUID по ошибке.
	shortTaskRE = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]*-\d+)$`)
)

type ParsedURL struct {
	TaskID      string // UUID карточки (если разобрали сразу)
	CompanyID   string // UUID компании (длинный формат)
	ShortTeamID string // 12-hex (короткий формат)
	ShortTaskID string // `OIP1-2454`
}

// ParseTaskURL — разобрать ссылку на YG-карточку.
//
// Возвращает nil, если в строке ничего YG-подобного. На уровне вызова
// решают: есть UUID → импортируем напрямую; есть только ShortTaskID →
// идём в API искать UUID.
func ParseTaskURL(raw string) *ParsedURL {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(parsed.Host), "yougile") {
		return nil
	}

	// 1. UUID карточки явным образом (?task=UUID или #...task=UUID, #task-UUID).
	var candidates []string
	if parsed.RawQuery != "" {
		if q, err := url.ParseQuery(parsed.RawQuery); err == nil {
			candidates = append(candidates, q["task"]...)
		}
	}
	frag := parsed.Fragment
	if frag != "" {
		if i := strings.Index(frag, "?"); i >= 0 {
			if q, err := url.ParseQuery(frag[i+1:]); err == nil {
				candidates = append(candidates, q["task"]...)
			}
		}
		if i := strings.Index(frag, "task-"); i >= 0 {
			candidates = append(candidates, frag[i+len("task-"):])
		}
	}

	taskID := ""
	for _, c := range candidates {
		if m := uuidRE.FindString(c); m != "" {
			taskID = m
			break
		}
	}
	if taskID == "" {
		// Fallback: первый UUID где-нибудь в URL.
		taskID = uuidRE.FindString(raw)
	}

	// 2. companyId из path: либо полный UUID, либо 12-hex короткий.
	companyID, shortTeamID := "", ""
	if m := teamUUIDRE.FindStringSubmatch(parsed.Path); m != nil {
		companyID = m[1]
	} else if m := teamShortRE.FindStringSubmatch(parsed.Path); m != nil {
		shortTeamID = strings.ToLower(m[1])
	}

	// 3. Короткий taskId в hash'е (`#OIP1-2454`). Не пытаемся, если уже
	// есть UUID.
	shortTaskID := ""
	if taskID == "" && frag != "" {
		head := frag
		if i := strings.Index(head, "?"); i >= 0 {
			head = head[:i]
		}
		if m := shortTaskRE.FindStringSubmatch(head); m != nil {
			shortTaskID = strings.ToUpper(m[1])
		}
	}

	if taskID == "" && shortTaskID == "" {
		return nil
	}
	return &ParsedURL{
		TaskID:      strings.ToLower(taskID),
		CompanyID:   companyID,
		ShortTeamID: shortTeamID,
		ShortTaskID: shortTaskID,
	}
}
