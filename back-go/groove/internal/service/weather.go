package service

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/dto"
)

// Погодный слой Грувика: фоновый цикл следит за погодой по локациям
// пользователей (Open-Meteo), значимые перемены превращает в реплики
// в pet-чате, а текущее состояние подмешивается в AI-промпты
// (pet-чат, утренний брифинг). Всё fail-open: ни API погоды, ни Redis
// не роняют основной флоу.

const (
	weatherTickInterval = 30 * time.Minute

	weatherCurKeyPrefix      = "gw2:groove:weather:cur:"      // текущая погода (JSON) для промптов и UI
	weatherStateKeyPrefix    = "gw2:groove:weather:state:"    // последнее состояние для детекта переходов
	weatherNotifiedKeyPrefix = "gw2:groove:weather:notified:" // кулдаун реплик в pet-чате

	weatherCurTTL   = 2 * time.Hour
	weatherStateTTL = 24 * time.Hour
	weatherCooldown = 4 * time.Hour

	// Окно реплик о погоде (МСК): ночью Грувик спит.
	weatherQuietFromH = 8
	weatherQuietToH   = 22

	geoSearchLimit = 6
)

// weatherSnapshot — снимок погоды в Redis-кэше.
type weatherSnapshot struct {
	Code     int     `json:"code"`
	TempC    float64 `json:"temp_c"`
	Category string  `json:"category"`
	Desc     string  `json:"description"`
	Emoji    string  `json:"emoji"`
	IsDay    bool    `json:"is_day"`
}

// classifyWeather — WMO weather interpretation code → категория/описание/эмодзи.
func classifyWeather(code int) (category, desc, emoji string) {
	switch {
	case code == 0:
		return "clear", "ясно", "☀️"
	case code == 1 || code == 2:
		return "clouds", "переменная облачность", "🌤️"
	case code == 3:
		return "overcast", "пасмурно", "☁️"
	case code == 45 || code == 48:
		return "fog", "туман", "🌫️"
	case code >= 51 && code <= 57:
		return "drizzle", "морось", "🌦️"
	case (code >= 61 && code <= 67) || (code >= 80 && code <= 82):
		return "rain", "дождь", "🌧️"
	case (code >= 71 && code <= 77) || code == 85 || code == 86:
		return "snow", "снег", "❄️"
	case code >= 95:
		return "storm", "гроза", "⛈️"
	default:
		return "clouds", "облачно", "🌥️"
	}
}

func newWeatherSnapshot(w *domain.Weather) weatherSnapshot {
	cat, desc, emoji := classifyWeather(w.Code)
	return weatherSnapshot{
		Code: w.Code, TempC: w.TempC,
		Category: cat, Desc: desc, Emoji: emoji, IsDay: w.IsDay,
	}
}

// weatherGroup — категории, близкие по сути, считаем одним состоянием:
// морось → дождь не считается «началом дождя» заново.
func weatherGroup(category string) string {
	switch category {
	case "drizzle", "rain":
		return "rain"
	case "snow", "storm", "fog":
		return category
	default:
		return "dry"
	}
}

// weatherTransition — значимый переход погоды между наблюдениями;
// "" — перемена не заслуживает реплики.
func weatherTransition(prev, cur weatherSnapshot) string {
	pg, cg := weatherGroup(prev.Category), weatherGroup(cur.Category)
	if cg != pg {
		switch cg {
		case "rain", "snow", "storm", "fog":
			return cg
		case "dry":
			if pg != "fog" && cur.Category == "clear" {
				return "cleared"
			}
		}
	}
	if cur.TempC >= 30 && prev.TempC < 30 {
		return "heat"
	}
	if cur.TempC <= -15 && prev.TempC > -15 {
		return "frost"
	}
	return ""
}

func formatTempC(t float64) string {
	n := int(math.Round(t))
	if n > 0 {
		return "+" + strconvInt(n) + "°C"
	}
	return strconvInt(n) + "°C"
}

// Статичные реплики, если AI выключен.
var weatherRemarks = map[string][]string{
	"rain": {
		"Кажется, за окном начался дождик 🌧️ Уютно стучит — хорошо, что грувы не промокают.",
		"Дождь пошёл! Если соберёшься на улицу — возьми зонт, а я тут лапки погрею.",
	},
	"snow": {
		"Смотри, снег пошёл! ❄️ Я бы слепил снеговика, но у меня лапки.",
		"За окном снежинки! Самое время для тёплого чая и пары спокойных юнитов.",
	},
	"storm": {
		"Ого, гроза собирается! ⛈️ Я спрячусь под стол, а ты далеко не уходи.",
		"Гремит! Если что, я не боюсь. Почти. Подержи меня за лапку.",
	},
	"fog": {
		"Туман за окном — ничего не видно 🌫️ Хорошо, что дорогу к задачам я помню наизусть.",
		"Всё в тумане! Как мои планы на вечер. Но тебе виднее.",
	},
	"cleared": {
		"Распогодилось! ☀️ Может, выйдешь проветриться на перерыве?",
		"Солнышко вышло! Самое время на короткую прогулку — я покараулю задачи.",
	},
	"heat": {
		"Ну и жара сегодня 🥵 Пей побольше воды, а я полежу в тени монитора.",
		"За окном пекло! Береги себя — перегреваться разрешено только процессору.",
	},
	"frost": {
		"Брр, ну и мороз! 🥶 Одевайся теплее, а я укутаюсь в свой стрик.",
		"Холодина за окном! Лучший план: тёплый чай, плед и мы с тобой.",
	},
}

// Подсказки AI: что именно случилось за окном.
var weatherRemarkHint = map[string]string{
	"rain":    "у хозяина за окном начался дождь",
	"snow":    "у хозяина за окном пошёл снег",
	"storm":   "у хозяина за окном началась гроза",
	"fog":     "у хозяина за окном опустился туман",
	"cleared": "у хозяина за окном распогодилось и вышло солнце",
	"heat":    "у хозяина за окном сильная жара",
	"frost":   "у хозяина за окном крепкий мороз",
}

// ───────────────────────── фоновый цикл ────────────────────────────

// RunWeatherLoop — раз в полчаса опрашивает Open-Meteo по локациям
// пользователей и оживляет Грувика погодным контекстом.
func (s *Service) RunWeatherLoop(ctx context.Context) {
	s.log.Info("groove.weather.loop_start", "interval", weatherTickInterval.String())
	ticker := time.NewTicker(weatherTickInterval)
	defer ticker.Stop()
	for {
		s.weatherTick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) weatherTick(ctx context.Context) {
	locs, err := s.locs.ListLocations(ctx)
	if err != nil {
		s.log.Warn("groove.weather.tick_failed", "error", err)
		return
	}
	for _, loc := range locs {
		if ctx.Err() != nil {
			return
		}
		s.observeWeather(ctx, loc)
	}
}

// observeWeather — снять погоду по локации, обновить кэши и, если случилась
// значимая перемена, сказать об этом в pet-чате.
func (s *Service) observeWeather(ctx context.Context, loc *domain.UserLocation) {
	w, err := s.weather.Current(ctx, loc.Lat, loc.Lon)
	if err != nil {
		s.log.Warn("groove.weather.fetch_failed", "user_id", loc.UserID, "error", err)
		return
	}
	cur := newWeatherSnapshot(w)
	raw, _ := json.Marshal(cur)
	s.daily.SetCache(ctx, weatherCurKeyPrefix+strconvI64(loc.UserID), string(raw), weatherCurTTL)

	prevRaw := s.daily.GetCache(ctx, weatherStateKeyPrefix+strconvI64(loc.UserID))
	s.daily.SetCache(ctx, weatherStateKeyPrefix+strconvI64(loc.UserID), string(raw), weatherStateTTL)
	if prevRaw == "" {
		return // первое наблюдение — не с чем сравнивать
	}
	var prev weatherSnapshot
	if json.Unmarshal([]byte(prevRaw), &prev) != nil {
		return
	}
	if kind := weatherTransition(prev, cur); kind != "" {
		s.maybeWeatherRemark(ctx, loc, cur, kind)
	}
}

func (s *Service) maybeWeatherRemark(ctx context.Context, loc *domain.UserLocation,
	cur weatherSnapshot, kind string) {

	now := time.Now().In(domain.MSK)
	if now.Hour() < weatherQuietFromH || now.Hour() >= weatherQuietToH {
		return
	}
	cooldownKey := weatherNotifiedKeyPrefix + strconvI64(loc.UserID)
	if s.daily.Exists(ctx, cooldownKey) {
		return
	}
	conv, err := s.convs.GetPetConversationByOwner(ctx, loc.UserID)
	if err != nil || conv == nil {
		return // pet-чата ещё нет — некуда писать
	}
	pet, err := s.pets.GetOrCreate(ctx, loc.UserID, conv.CompanyID)
	if err != nil {
		return
	}
	text := s.weatherRemarkText(ctx, conv.CompanyID, pet, loc.UserID, cur, kind)
	if text == "" {
		return
	}
	// msgsvc сам эмитит message:new — здесь ничего не публикуем.
	if err := s.msgr.PostBotMessage(ctx, conv.ID, text); err != nil {
		s.log.Warn("groove.weather.post_failed", "user_id", loc.UserID, "error", err)
		return
	}
	s.daily.SetCache(ctx, cooldownKey, "1", weatherCooldown)
	s.log.Info("groove.weather.remark", "user_id", loc.UserID, "kind", kind)
}

func (s *Service) weatherRemarkText(ctx context.Context, companyID int64,
	pet *domain.Pet, userID int64, cur weatherSnapshot, kind string) string {

	if s.ai.Enabled(ctx, companyID) {
		ownerName := "хозяин"
		if owner, err := s.users.GetUser(ctx, userID); err == nil && owner != nil {
			ownerName = firstName(owner.FIO)
		}
		prompt := "Ты — " + pet.Name + ", виртуальный питомец-Грувик сотрудника " +
			ownerName + ". Сейчас " + weatherRemarkHint[kind] + " (" + cur.Desc + ", " +
			formatTempC(cur.TempC) + "). Напиши ОДНУ короткую реплику хозяину в чат " +
			"об этой перемене погоды (до 160 символов): тепло, по-дружески, с юмором, " +
			"можно эмодзи. Не зови работать и не упоминай задачи."
		text, err := s.ai.Chat(ctx, companyID, []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		}, 120, 0.95, 20*time.Second)
		if err == nil {
			if t := trimAIReply(text); t != "" {
				return t
			}
		} else {
			s.log.Warn("groove.weather.ai_failed", "user_id", userID, "error", err)
		}
	}
	pool := weatherRemarks[kind]
	if len(pool) == 0 {
		return ""
	}
	return pool[randIntn(len(pool))]
}

// ─────────────────── погода в AI-промптах Грувика ───────────────────

func (s *Service) cachedWeather(ctx context.Context, userID int64) *weatherSnapshot {
	raw := s.daily.GetCache(ctx, weatherCurKeyPrefix+strconvI64(userID))
	if raw == "" {
		return nil
	}
	var snap weatherSnapshot
	if json.Unmarshal([]byte(raw), &snap) != nil {
		return nil
	}
	return &snap
}

// weatherPromptLine — строка о погоде за окном хозяина для AI-промптов;
// "" — локация не задана или кэш ещё не прогрет фоновым циклом.
func (s *Service) weatherPromptLine(ctx context.Context, userID int64) string {
	snap := s.cachedWeather(ctx, userID)
	if snap == nil {
		return ""
	}
	return "За окном у хозяина сейчас " + snap.Desc + ", " + formatTempC(snap.TempC) +
		". Можешь к месту сослаться на погоду (зонт, прогулка, тёплый чай) — " +
		"но не в каждой реплике."
}

// ───────────────────── use-case'ы локации ──────────────────────────

// GetUserLocation — локация пользователя и кэш погоды для UI.
func (s *Service) GetUserLocation(ctx context.Context, userID int64) (*dto.LocationStateDTO, error) {
	loc, err := s.locs.GetLocation(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := &dto.LocationStateDTO{}
	if loc != nil {
		out.Location = dto.NewLocation(loc)
		if snap := s.cachedWeather(ctx, userID); snap != nil {
			out.Weather = &dto.WeatherDTO{
				TempC:       snap.TempC,
				Description: snap.Desc,
				Category:    snap.Category,
				Emoji:       snap.Emoji,
			}
		}
	}
	return out, nil
}

// SetUserLocation — сохранить локацию и сразу прогреть кэш погоды,
// чтобы UI и промпты получили контекст без ожидания фонового цикла.
func (s *Service) SetUserLocation(ctx context.Context, userID int64,
	lat, lon float64, city *string) (*dto.LocationStateDTO, error) {

	if err := s.locs.SaveLocation(ctx, &domain.UserLocation{
		UserID: userID, Lat: lat, Lon: lon, City: city,
	}); err != nil {
		return nil, err
	}
	if w, err := s.weather.Current(ctx, lat, lon); err == nil {
		cur := newWeatherSnapshot(w)
		raw, _ := json.Marshal(cur)
		s.daily.SetCache(ctx, weatherCurKeyPrefix+strconvI64(userID), string(raw), weatherCurTTL)
		s.daily.SetCache(ctx, weatherStateKeyPrefix+strconvI64(userID), string(raw), weatherStateTTL)
	} else {
		s.log.Warn("groove.weather.warm_failed", "user_id", userID, "error", err)
	}
	return s.GetUserLocation(ctx, userID)
}

func (s *Service) DeleteUserLocation(ctx context.Context, userID int64) error {
	return s.locs.DeleteLocation(ctx, userID)
}

// SearchCities — прокси к геокодингу Open-Meteo (фронт ходит только на наш
// API). Fail-open: геокодинг лёг — пустой список, не ошибка.
func (s *Service) SearchCities(ctx context.Context, query string) ([]domain.GeoPlace, error) {
	places, err := s.weather.SearchCities(ctx, query, geoSearchLimit)
	if err != nil {
		s.log.Warn("groove.weather.geocode_failed", "error", err)
		return []domain.GeoPlace{}, nil
	}
	if places == nil {
		places = []domain.GeoPlace{}
	}
	return places, nil
}
