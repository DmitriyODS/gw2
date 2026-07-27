package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Сквозной сценарий напоминаний: создание → чужой его не видит → «отложить»
// двигает срок → «готово» уводит разовое в журнал.
func TestRemindersOwnerScopeAndActions(t *testing.T) {
	owner := newVerifiedUser(t)
	other := newVerifiedUser(t)

	r := reminderAPI.doJSON(t, http.MethodPost, "/api/reminders", owner.Token, map[string]any{
		"title":     "Позвонить подрядчику",
		"note":      "уточнить сроки",
		"remind_at": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		"timezone":  "Europe/Moscow",
	})
	requireStatus(t, r, 201, "создание напоминания")
	id := int64(r.Num("id"))

	// Скоуп по владельцу: чужому — 404, не 403.
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, fmt.Sprintf("/api/reminders/%d", id)},
		{http.MethodPatch, fmt.Sprintf("/api/reminders/%d", id)},
		{http.MethodDelete, fmt.Sprintf("/api/reminders/%d", id)},
		{http.MethodPost, fmt.Sprintf("/api/reminders/%d/done", id)},
	} {
		rr := reminderAPI.doJSON(t, tc.method, tc.path, other.Token, map[string]any{"title": "x"})
		if rr.Status != 404 {
			t.Fatalf("%s %s чужим: ожидался 404, получен %d: %s", tc.method, tc.path, rr.Status, rr.Raw)
		}
	}

	// Отложить: срок уезжает вперёд, напоминание остаётся активным.
	r = reminderAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/reminders/%d/snooze", id), owner.Token,
		map[string]any{"minutes": 15})
	requireStatus(t, r, 200, "отложить")
	if !r.Bool("active") {
		t.Fatalf("после отсрочки напоминание должно остаться активным: %s", r.Raw)
	}

	// Готово: разовое уходит в журнал (active=false).
	r = reminderAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/reminders/%d/done", id), owner.Token, nil)
	requireStatus(t, r, 200, "готово")
	if r.Bool("active") {
		t.Fatalf("разовое напоминание после «готово» должно стать неактивным: %s", r.Raw)
	}

	r = reminderAPI.doJSON(t, http.MethodGet, "/api/reminders?scope=done", owner.Token, nil)
	requireStatus(t, r, 200, "журнал")
	if !strings.Contains(string(r.Raw), "Позвонить подрядчику") {
		t.Fatalf("сработавшее напоминание должно быть в журнале: %s", r.Raw)
	}
}

// Валидация: без текста и без времени напоминание не создаётся.
func TestRemindersValidation(t *testing.T) {
	owner := newVerifiedUser(t)

	r := reminderAPI.doJSON(t, http.MethodPost, "/api/reminders", owner.Token, map[string]any{
		"remind_at": time.Now().Add(time.Hour).Format(time.RFC3339),
	})
	if r.Status != 400 {
		t.Fatalf("напоминание без текста должно отклоняться: %d %s", r.Status, r.Raw)
	}

	r = reminderAPI.doJSON(t, http.MethodPost, "/api/reminders", owner.Token, map[string]any{"title": "Без времени"})
	if r.Status != 400 {
		t.Fatalf("напоминание без времени должно отклоняться: %d %s", r.Status, r.Raw)
	}
}

// Планировщик: наступивший срок разово гасит активность, а повторяющееся
// напоминание переносит вперёд. Проверяем на живом сервисе (тик — 30 с).
func TestRemindersSchedulerFires(t *testing.T) {
	if testing.Short() {
		t.Skip("ждём тик планировщика — не для -short")
	}
	owner := newVerifiedUser(t)
	past := time.Now().Add(-time.Minute).Format(time.RFC3339)

	r := reminderAPI.doJSON(t, http.MethodPost, "/api/reminders", owner.Token, map[string]any{
		"title": "Разовое", "remind_at": past, "timezone": "Europe/Moscow",
	})
	requireStatus(t, r, 201, "создание просроченного разового")
	oneShot := int64(r.Num("id"))

	r = reminderAPI.doJSON(t, http.MethodPost, "/api/reminders", owner.Token, map[string]any{
		"title": "Ежедневное", "remind_at": past, "timezone": "Europe/Moscow",
		"repeat": map[string]any{"kind": "daily", "interval": 1},
	})
	requireStatus(t, r, 201, "создание просроченного повтора")
	repeating := int64(r.Num("id"))

	deadline := time.Now().Add(45 * time.Second)
	for {
		one := reminderAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/reminders/%d", oneShot), owner.Token, nil)
		rep := reminderAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/reminders/%d", repeating), owner.Token, nil)
		fired := !one.Bool("active") && rep.Str("last_fired_at") != ""
		if fired {
			// Повтор обязан уехать в будущее, а не выстрелить пачкой пропусков.
			next, err := time.Parse(time.RFC3339, rep.Str("remind_at"))
			if err != nil {
				t.Fatalf("не разобрали следующий срок: %v (%s)", err, rep.Raw)
			}
			if !next.After(time.Now()) {
				t.Fatalf("следующий срок повтора должен быть в будущем: %s", next)
			}
			if !rep.Bool("active") {
				t.Fatalf("повторяющееся напоминание должно остаться активным: %s", rep.Raw)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("планировщик не сработал за отведённое время: разовое=%s повтор=%s", one.Raw, rep.Raw)
		}
		time.Sleep(2 * time.Second)
	}
}
