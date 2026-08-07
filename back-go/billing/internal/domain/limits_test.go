package domain

import (
	"testing"
	"time"
)

// Пока подписки скрыты, токены ИИ не зависят от тарифа: всем поровну и на сутки.
func TestAIQuotaIgnoresPlanWhileSubscriptionsHidden(t *testing.T) {
	if !SubscriptionsHidden {
		t.Skip("подписки снова видны — квота считается по тарифу")
	}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	for _, plan := range PlanCodes {
		tokens, _ := AIQuota(LimitsFor(plan), now)
		if tokens != DailyAITokens {
			t.Fatalf("тариф %s: квота %d, ждали %d", plan, tokens, DailyAITokens)
		}
	}
}

// Период кончается в московскую полночь — «1000 в день» считается по календарю
// МСК, как и остальная статистика платформы.
func TestAIQuotaPeriodEndsAtMoscowMidnight(t *testing.T) {
	if !SubscriptionsHidden {
		t.Skip("подписки снова видны — период месячный")
	}
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			// 12:00 UTC = 15:00 МСК → полночь МСК той же ночи (21:00 UTC).
			name: "день",
			now:  time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			want: time.Date(2026, 8, 4, 21, 0, 0, 0, time.UTC),
		},
		{
			// 22:00 UTC = 01:00 МСК СЛЕДУЮЩИХ суток → полночь через сутки.
			name: "после московской полуночи",
			now:  time.Date(2026, 8, 4, 22, 0, 0, 0, time.UTC),
			want: time.Date(2026, 8, 5, 21, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, end := AIQuota(LimitsFor(PlanJunior), c.now)
			if !end.Equal(c.want) {
				t.Fatalf("конец периода %s, ждали %s", end.UTC(), c.want)
			}
			if !end.After(c.now) {
				t.Fatalf("период кончается в прошлом: %s <= %s", end.UTC(), c.now)
			}
		})
	}
}

/* Пока подписки скрыты, тариф не должен ограничивать: заплатить нельзя, а
   бесплатный «Джун» разрешает одну доску и трёх человек — упёршемуся некуда
   идти. Живых ограничений два: токены ИИ (AIQuota) и место в хранилище. */
func TestEffectiveLimitsLiftPlanCapsWhileSubscriptionsHidden(t *testing.T) {
	if !SubscriptionsHidden {
		t.Skip("подписки снова видны — действуют лимиты тарифа")
	}
	got := EffectiveLimits(LimitsFor(PlanJunior))

	counters := map[string]int64{
		"Tasks":       got.Tasks,
		"Companies":   int64(got.Companies),
		"Members":     int64(got.Members),
		"Calendars":   int64(got.Calendars),
		"Diaries":     int64(got.Diaries),
		"Boards":      int64(got.Boards),
		"Registries":  int64(got.Registries),
		"ChatFolders": int64(got.ChatFolders),
	}
	for name, v := range counters {
		if v != Unlimited {
			t.Errorf("%s = %d, ждали безлимит", name, v)
		}
	}
	if !got.Portal || !got.AdvancedStats || !got.DataTransfer || !got.UserStatuses {
		t.Errorf("платные возможности должны быть открыты: %+v", got)
	}

	// Сама линейка тарифов не тронута — она вернётся вместе с оплатой.
	if LimitsFor(PlanJunior).Boards != 1 {
		t.Error("PlanLimits изменены: тарифная линейка должна остаться как есть")
	}
}

// Место — единственный счётный лимит, который остаётся при скрытых подписках:
// оно упирается в диск, а не в тариф, и одинаково у всех.
func TestEffectiveLimitsKeepStorageCapWhileSubscriptionsHidden(t *testing.T) {
	if !SubscriptionsHidden {
		t.Skip("подписки снова видны — действуют лимиты тарифа")
	}
	for _, plan := range PlanCodes {
		if got := EffectiveLimits(LimitsFor(plan)).StorageBytes; got != FreeStorageBytes {
			t.Errorf("%s: место = %d, ждали %d", plan, got, FreeStorageBytes)
		}
	}
}
