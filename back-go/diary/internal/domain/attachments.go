package domain

import (
	"strconv"
	"strings"
)

/* Вложения записи — то, что человек положил «во время» из других разделов:
   заметку к созвону, файл к встрече, опрос к пятнице. Хранится ссылка и
   снимок названия, а не копия: вещь живёт в своём разделе, а запись лишь
   возвращает её в нужный момент. Доступ к самой вещи проверяет её раздел при
   открытии — снимок названия ничего сверх того не раскрывает. */

// Attachment — ссылка на вещь другого раздела.
type Attachment struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
	// ParentID — контейнер вещи там, где без него её не открыть: реестр у
	// записи реестра, календарь у события.
	ParentID *int64 `json:"parent_id,omitempty"`
	Title    string `json:"title"`
}

// AttachmentKinds — что можно приложить. Ключ совпадает с разделом на клиенте.
var AttachmentKinds = map[string]bool{
	"note": true, "board": true, "drive_file": true, "drive_folder": true,
	"form": true, "registry": true, "registry_record": true,
	"calendar": true, "calendar_entry": true, "schedule": true, "task": true,
}

const (
	MaxAttachments     = 20
	maxAttachmentTitle = 200
)

var ErrAttachmentInvalid = NewError("VALIDATION", "Вложение не распознано", 400)

// NormalizeAttachments — проверить вложения и подрезать названия. Повтор
// одной и той же вещи схлопывается.
func NormalizeAttachments(in []Attachment) ([]Attachment, error) {
	if len(in) > MaxAttachments {
		return nil, NewError("VALIDATION", "Слишком много вложений", 400)
	}
	out := make([]Attachment, 0, len(in))
	seen := map[string]bool{}
	for _, a := range in {
		if !AttachmentKinds[a.Kind] || a.ID <= 0 {
			return nil, ErrAttachmentInvalid
		}
		if (a.Kind == "registry_record" || a.Kind == "calendar_entry") && (a.ParentID == nil || *a.ParentID <= 0) {
			return nil, ErrAttachmentInvalid
		}
		key := a.Kind + ":" + strconv.FormatInt(a.ID, 10)
		if seen[key] {
			continue
		}
		seen[key] = true
		a.Title = strings.TrimSpace(a.Title)
		if r := []rune(a.Title); len(r) > maxAttachmentTitle {
			a.Title = string(r[:maxAttachmentTitle])
		}
		out = append(out, a)
	}
	return out, nil
}
