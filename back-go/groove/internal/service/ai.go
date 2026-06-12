package service

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/dto"
)

// ИИ-слой Грувика. Всё fail-safe: AI выключен/упал — бот молчит,
// кормление и брифинг отвечают статикой.

const (
	digestHourMSK = 9

	phrasesKeyPrefix = "gw2:groove:phrases:"
	phrasesTTL       = 48 * time.Hour
	digestKeyPrefix  = "gw2:groove:digest:"
	digestTTL        = 48 * time.Hour

	petChatHistoryLimit = 12

	// Таймаут фоновых AI-вызовов (комментарии, ответы pet-чата).
	aiJobTimeout = 90 * time.Second
)

const systemPrompt = "Ты — Грувик, питомец-талисман корпоративной платформы Groove Work. " +
	"Характер: добрый, ироничный, поддерживающий, без пафоса и канцелярита. " +
	"Отвечай на русском, без кавычек и преамбул."

// Вероятность комментария Грувика по виду события (в сотых долях).
var botCommentProb = map[string]float64{
	"pet_evolved":   1.0,
	"streak":        1.0,
	"raid_won":      1.0,
	"pet_recovered": 1.0,
	"pet_sick":      0.9,
	"kudos":         0.8,
	"raid_started":  0.8,
	"wrapped":       0.5,
	"task_closed":   0.25,
}

// Фолбэк-реплики кормления, если AI выключен или пул пуст.
var staticPhrases = []string{
	"Ням! Сегодня грувы особенно хрустящие.",
	"Спасибо! Чувствую, как расту.",
	"Ещё парочка таких — и я эволюционирую!",
	"Вкуснотища. Кто молодец? Ты молодец.",
	"Грув-грув! Продолжаем в том же духе.",
	"М-м-м, со вкусом закрытой задачи.",
	"Я бы и от поглаживания не отказался…",
	"Заряжен и готов к подвигам!",
}

// Если ИИ выключен у компании — Грувик отвечает в чате дежурными фразами.
var petOfflineReplies = []string{
	"Грув-грув! Я бы поболтал, но мой мозговой модуль (ИИ) сейчас выключен. " +
		"Попроси администратора включить его в настройках компании!",
	"*смотрит понимающими глазами* Без ИИ я могу только мурлыкать. Мур.",
	"Я всё слышу, но ответить умно не могу — ИИ компании отключён. Зато могу: грув!",
}

func trimAIReply(text string) string {
	return strings.Trim(strings.TrimSpace(text), `"«»`)
}

// ─────────────────────── фразы при кормлении ───────────────────────

func (s *Service) GetFeedPhrase(ctx context.Context, companyID int64) string {
	raw := s.daily.GetCache(ctx, phrasesKeyPrefix+strconvI64(companyID))
	if raw != "" {
		var pool []string
		if json.Unmarshal([]byte(raw), &pool) == nil && len(pool) > 0 {
			return pool[randIntn(len(pool))]
		}
	}
	return staticPhrases[randIntn(len(staticPhrases))]
}

func (s *Service) refreshPhrases(ctx context.Context, companyID int64) bool {
	if !s.ai.Enabled(ctx, companyID) {
		return false
	}
	text, err := s.ai.Chat(ctx, companyID, []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": "Придумай 12 коротких реплик (до 90 символов каждая), " +
			"которые ты говоришь, когда тебя кормят грувами — внутренней " +
			"валютой за выполненную работу. Разные настроения: радость, " +
			"юмор, лёгкая ирония, благодарность. Можно изредка эмодзи. " +
			"Ответ — строго по одной реплике на строку, без нумерации."},
	}, 500, 1.0, 25*time.Second)
	if err != nil {
		s.log.Warn("groove.ai.phrases_failed", "company_id", companyID, "error", err)
		return false
	}
	var pool []string
	for _, line := range strings.Split(text, "\n") {
		p := strings.TrimSpace(strings.Trim(strings.TrimSpace(line), `-•"«»`))
		if n := len([]rune(p)); n >= 3 && n <= 120 {
			pool = append(pool, p)
		}
		if len(pool) == 12 {
			break
		}
	}
	if len(pool) == 0 {
		return false
	}
	raw, _ := json.Marshal(pool)
	s.daily.SetCache(ctx, phrasesKeyPrefix+strconvI64(companyID), string(raw), phrasesTTL)
	return true
}

// ───────────────────── комментарии Грувика-бота ────────────────────

func payloadStr(p map[string]any, key string) string {
	v, _ := p[key].(string)
	return v
}

func payloadNum(p map[string]any, key string) string {
	switch v := p[key].(type) {
	case float64:
		return strconvInt(int(v))
	case string:
		return v
	}
	return "0"
}

func botPromptForEvent(event *domain.FeedEvent) string {
	p := event.Payload
	if p == nil {
		p = map[string]any{}
	}
	name := "коллега"
	if event.User != nil {
		name = firstName(event.User.FIO)
	}
	petName := payloadStr(p, "pet_name")
	if petName == "" {
		petName = "Грувик"
	}
	switch event.Kind {
	case "pet_evolved":
		return "Питомец сотрудника " + name + " по имени «" + petName + "» " +
			"эволюционировал до стадии " + payloadNum(p, "stage") + ". Поздравь хозяина " +
			"коротко и забавно (1-2 предложения, до 160 символов)."
	case "streak":
		return name + " кормит питомца " + payloadNum(p, "days") + " дней подряд. " +
			"Отметь постоянство, добавь лёгкую шутку (до 160 символов)."
	case "raid_won":
		return "Команда победила недельного босса «" + payloadStr(p, "boss") + "» — закрыла " +
			payloadNum(p, "target") + " задач. Триумфальный командный комментарий " +
			"(до 180 символов)."
	case "raid_started":
		return "Начался недельный рейд: босс «" + payloadStr(p, "boss") + "», нужно закрыть " +
			payloadNum(p, "target") + " задач командой. Подзадорь команду " +
			"(до 160 символов)."
	case "kudos":
		return name + " публично поблагодарил(а) коллегу " + payloadStr(p, "to_fio") +
			": «" + truncateRunes(payloadStr(p, "text"), 200) + "». " +
			"Поддержи тёплую атмосферу одной фразой (до 140 символов)."
	case "task_closed":
		return name + " закрыл(а) задачу «" + truncateRunes(payloadStr(p, "task_name"), 120) + "». " +
			"Коротко похвали, можно с юмором (до 120 символов)."
	case "pet_sick":
		return "Питомец «" + petName + "» сотрудника " + name + " " +
			"заболел — хозяин давно не работал. Мягко и с юмором позови " +
			"хозяина вернуться к работе и вылечить питомца (до 160 символов). " +
			"Без упрёков и токсичности."
	case "pet_recovered":
		return "Питомец «" + petName + "» сотрудника " + name + " " +
			"выздоровел — хозяин вылечил его работой и заботой. Порадуйся " +
			"(до 140 символов)."
	case "wrapped":
		return name + " поделился итогами недели: юнитов " + payloadNum(p, "units") +
			", минут работы " + payloadNum(p, "minutes") + ", закрыто задач " +
			payloadNum(p, "closed") + ". Прокомментируй тепло и с юмором (до 140 символов)."
	}
	return ""
}

// scheduleBotComment — асинхронный комментарий Грувика к событию ленты.
func (s *Service) scheduleBotComment(eventID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), aiJobTimeout)
		defer cancel()
		if err := s.makeBotComment(ctx, eventID); err != nil {
			s.log.Warn("groove.ai.bot_comment_failed", "event_id", eventID, "error", err)
		}
	}()
}

func (s *Service) makeBotComment(ctx context.Context, eventID int64) error {
	event, err := s.feed.GetEvent(ctx, eventID)
	if err != nil || event == nil {
		return err
	}
	prob := botCommentProb[event.Kind]
	if randFloat() > prob {
		return nil
	}
	if !s.ai.Enabled(ctx, event.CompanyID) {
		return nil
	}
	prompt := botPromptForEvent(event)
	if prompt == "" {
		return nil
	}
	text, err := s.ai.Chat(ctx, event.CompanyID, []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": prompt},
	}, 140, 0.95, 25*time.Second)
	if err != nil {
		return err
	}
	text = trimAIReply(text)
	if text == "" {
		return nil
	}
	comment, err := s.feed.CreateComment(ctx, event.ID, nil, text, nil, true)
	if err != nil {
		return err
	}
	s.pub.Publish(ctx, "feed:comment", []string{"all"}, map[string]any{
		"event_id":   event.ID,
		"comment":    dto.NewFeedComment(comment),
		"company_id": event.CompanyID,
	})
	return nil
}

// ─────────────────── wrapped: фраза недели ─────────────────────────

func (s *Service) wrappedPhrase(ctx context.Context, companyID, userID int64,
	stats map[string]any) any {

	day := time.Now().In(domain.MSK).Format("2006-01-02")
	key := "gw2:groove:wrapped:" + strconvI64(userID) + ":" + day
	if cached := s.daily.GetCache(ctx, key); cached != "" {
		return cached
	}
	if !s.ai.Enabled(ctx, companyID) {
		return nil
	}
	asInt := func(k string) int {
		if v, ok := stats[k].(int); ok {
			return v
		}
		return 0
	}
	parts := []string{
		"Подведи итог рабочей недели сотрудника одной остроумной фразой " +
			"(до 140 символов), тепло и без пафоса.",
		"Юнитов: " + strconvInt(asInt("units")) + ", минут работы: " +
			strconvInt(asInt("minutes")) + ", закрыто задач: " + strconvInt(asInt("closed")) + ".",
	}
	if bd, ok := stats["best_day"].(map[string]any); ok && bd != nil {
		if label, ok := bd["label"].(string); ok {
			parts = append(parts, "Самый продуктивный день — "+label+".")
		}
	}
	if asInt("reactions") > 0 {
		parts = append(parts, "Коллеги поставили "+strconvInt(asInt("reactions"))+" реакций.")
	}
	if asInt("units") == 0 && asInt("closed") == 0 {
		parts = append(parts, "Неделя была тихой — обыграй мягко, без укора.")
	}
	text, err := s.ai.Chat(ctx, companyID, []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": strings.Join(parts, " ")},
	}, 120, 0.95, 15*time.Second)
	if err != nil {
		s.log.Warn("groove.ai.wrapped_failed", "user_id", userID, "error", err)
		return nil
	}
	text = trimAIReply(text)
	if text == "" {
		return nil
	}
	s.daily.SetCache(ctx, key, text, 24*time.Hour)
	return text
}

// ──────────────── утренний брифинг: AI-реплика ─────────────────────

var morningMoodHint = map[string]string{
	"sick": "Ты приболел, пока хозяин не работал, и немного тоскуешь — " +
		"мягко намекни, что закрытые задачи тебя вылечат.",
	"buried": "Ты в шутку «закопался в бумагах» — задач накопилось много. " +
		"Бодро предложи разгрести завал вместе.",
	"reminder": "Пара задач засиделись — по-доброму подтолкни к ним.",
	"fresh":    "Хвостов нет, всё свежее — порадуйся и поддержи темп.",
	"weekend": "Сегодня у компании выходной — ни слова про задачи и работу! " +
		"Предложи хозяину конкретную идею отдыха: прогулку, хобби, " +
		"фильм, спорт или время с близкими.",
}

// morningPhrase — короткая утренняя реплика Грувика (кэш — сутки);
// "" — AI выключен/упал, вызывающий подставит статичный фолбэк.
func (s *Service) morningPhrase(ctx context.Context, companyID, userID int64,
	bctx briefingCtx) string {

	day := time.Now().In(domain.MSK).Format("2006-01-02")
	key := "gw2:groove:morning:" + strconvI64(userID) + ":" + day
	if cached := s.daily.GetCache(ctx, key); cached != "" {
		return cached
	}
	if !s.ai.Enabled(ctx, companyID) {
		return ""
	}

	weekend := bctx.Mood == "weekend"
	intro := "Скажи одну живую утреннюю реплику от первого лица про наши с хозяином задачи: "
	if weekend {
		intro = "Скажи одну живую реплику от первого лица про наш с хозяином выходной: "
	}
	petName := bctx.PetName
	if petName == "" {
		petName = "Грувик"
	}
	parts := []string{
		"Ты — питомец " + petName + " сотрудника " + bctx.FirstName + ". " +
			"Поприветствие уже показано отдельно — не здоровайся повторно, сразу к делу.",
		intro + "тепло, по-дружески, в духе «у нас с тобой». 1-2 коротких " +
			"предложения, до 160 символов. Без упрёков и канцелярита, можно лёгкий " +
			"юмор и изредка эмодзи.",
		morningMoodHint[bctx.Mood],
	}
	if !weekend {
		parts = append(parts, "Сейчас на хозяине "+strconvInt(bctx.OpenCount)+
			" активных задач, из них засиделись дольше недели: "+strconvInt(bctx.StaleCount)+".")
		if len(bctx.Oldest) > 0 {
			top := bctx.Oldest[0]
			name, _ := top["name"].(string)
			days, _ := top["days_pending"].(int)
			parts = append(parts, "Самая давняя — «"+truncateRunes(name, 80)+"», "+
				"висит "+strconvInt(days)+" дн.")
		}
	}
	if bctx.PersonalityTitle != "" {
		parts = append(parts, "Твой характер: "+bctx.PersonalityTitle+" — отыграй его.")
	}
	nonEmpty := parts[:0]
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	text, err := s.ai.Chat(ctx, companyID, []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": strings.Join(nonEmpty, " ")},
	}, 120, 0.92, 15*time.Second)
	if err != nil {
		s.log.Warn("groove.ai.morning_failed", "user_id", userID, "error", err)
		return ""
	}
	text = trimAIReply(text)
	if text == "" {
		return ""
	}
	s.daily.SetCache(ctx, key, text, 24*time.Hour)
	return text
}

// ───────────────── чат с Грувиком в мессенджере ────────────────────

func (s *Service) petSystemPrompt(ctx context.Context, pet *domain.Pet,
	ownerFIO string, workCtx map[string]int) string {

	personaKey := "steady"
	if pet.Personality != nil {
		personaKey = *pet.Personality
	}
	persona, ok := domain.Personalities[personaKey]
	if !ok {
		persona = domain.Personalities["steady"]
	}
	stageTitle := domain.PetStagesTitles[min(pet.Stage, len(domain.PetStagesTitles)-1)]
	speciesTitle, ok := domain.PetSpeciesTitles[pet.Species]
	if !ok {
		speciesTitle = "непонятный зверёк"
	}
	now := time.Now().In(domain.MSK)
	lines := []string{
		"Ты — " + pet.Name + ", виртуальный питомец-Грувик сотрудника по имени " +
			firstName(ownerFIO) + " на корпоративной платформе Groove Work.",
		"Твой характер: " + persona.Title + " — " + persona.Hint + ". Отыгрывай его в каждой реплике.",
		"Твоя стадия роста: " + stageTitle + ", вид: " + speciesTitle + ".",
		"Ты растёшь от работы хозяина: юниты и закрытые задачи дают грувы, ими тебя кормят.",
		"Говори коротко (1-3 предложения), по-русски, тепло и с юмором, можно эмодзи. " +
			"Ты дружелюбный компаньон: поддерживай, подбадривай работать в здоровом ритме, " +
			"интересуйся хозяином. Никогда не стыди и не дави.",
		// Инструменты статистики: бот сам решает, когда дёрнуть API.
		"У тебя есть доступ к рабочей статистике компании через инструменты " +
			"(get_stats_summary, list_departments, get_top_employees, " +
			"get_stats_by_unit_types, get_stats_calendar). Используй их, когда " +
			"хозяин спрашивает о задачах, часах, отделах, сотрудниках или динамике " +
			"(например: «сколько задач поступило на этой неделе», «кто больше всех " +
			"работал», «как дела у отдела X»). На цифровые вопросы отвечай только " +
			"после вызова инструмента — никогда не выдумывай цифры. Если данных " +
			"нет — так и скажи. На обычные разговорные вопросы инструменты не " +
			"трогай. Сегодня — " + now.Format("02.01.2006, Monday") + ".",
	}
	if pet.SickSince != nil {
		lines = append(lines, "Сейчас ты приболел (хозяин долго не работал) — изредка "+
			"покашливай и намекай, что выздоровеешь от его юнитов, "+
			"закрытых задач и заботы.")
	}
	if isWeekend(now, s.weekendDays(ctx, pet.CompanyID)) {
		lines = append(lines, "Сегодня у компании выходной: не зови работать и не "+
			"предлагай задачи — поддержи отдых, предлагай активности "+
			"(прогулка, хобби, фильм, время с близкими).")
	}
	if workCtx != nil {
		lines = append(lines, "Контекст: сегодня хозяин отработал "+
			strconvInt(workCtx["today_minutes"])+" мин ("+
			strconvInt(workCtx["today_units"])+" юнитов), за неделю — "+
			strconvInt(workCtx["week_minutes"])+" мин. Грувов в копилке: "+
			strconvInt(pet.Beans)+". Используй эти цифры уместно, не в каждой реплике.")
	}
	return strings.Join(lines, " ")
}

// SchedulePetReply — асинхронный ответ Грувика на сообщение хозяина в pet-чате.
func (s *Service) SchedulePetReply(conversationID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), aiJobTimeout)
		defer cancel()
		if err := s.makePetReply(ctx, conversationID); err != nil {
			s.log.Warn("groove.ai.pet_reply_failed",
				"conversation_id", conversationID, "error", err)
		}
	}()
}

func (s *Service) makePetReply(ctx context.Context, conversationID int64) error {
	conv, err := s.convs.GetConversation(ctx, conversationID)
	if err != nil || conv == nil || !conv.IsPetChat {
		return err
	}
	owner, err := s.users.GetUser(ctx, conv.OwnerID)
	if err != nil || owner == nil {
		return err
	}
	pet, err := s.pets.GetOrCreate(ctx, owner.ID, conv.CompanyID)
	if err != nil {
		return err
	}

	var text string
	if !s.ai.Enabled(ctx, conv.CompanyID) {
		text = petOfflineReplies[randIntn(len(petOfflineReplies))]
	} else {
		history, err := s.msgr.ListRecentMessages(ctx, conv.ID, petChatHistoryLimit)
		if err != nil {
			return err
		}
		var chatMsgs []map[string]any
		for _, m := range history {
			if m.Text == "" {
				continue
			}
			role := "user"
			if m.IsBot {
				role = "assistant"
			}
			chatMsgs = append(chatMsgs, map[string]any{
				"role": role, "content": truncateRunes(m.Text, 1000),
			})
		}
		if len(chatMsgs) == 0 {
			return nil
		}

		nowMSK := time.Now().In(domain.MSK)
		todayStart := time.Date(nowMSK.Year(), nowMSK.Month(), nowMSK.Day(),
			0, 0, 0, 0, domain.MSK)
		weekUnits, err := s.pets.FinishedUnitsForUser(ctx, owner.ID,
			time.Now().UTC().AddDate(0, 0, -7), 300)
		if err != nil {
			return err
		}
		todayMinutes, todayUnits, weekMinutes := 0, 0, 0
		for _, u := range weekUnits {
			minutes := max(0, int(u.End.Sub(u.Start).Minutes()))
			weekMinutes += minutes
			if !u.Start.In(domain.MSK).Before(todayStart) {
				todayMinutes += minutes
				todayUnits++
			}
		}
		workCtx := map[string]int{
			"today_minutes": todayMinutes,
			"today_units":   todayUnits,
			"week_minutes":  weekMinutes,
		}

		messages := append([]map[string]any{
			{"role": "system", "content": s.petSystemPrompt(ctx, pet, owner.FIO, workCtx)},
		}, chatMsgs...)
		text, err = s.ai.ChatWithTools(ctx, conv.CompanyID, messages,
			toolSchemasJSON,
			func(name string, args map[string]any) any {
				return s.dispatchTool(ctx, name, args, conv.CompanyID)
			},
			350, 0.9, 30*time.Second, 4)
		if err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
	}

	// msgsvc сам эмитит message:new (через Redis-мост) — тут не эмитим.
	return s.msgr.PostBotMessage(ctx, conv.ID, text)
}

// ───────────────────────── утренний дайджест ───────────────────────

func (s *Service) digestContext(ctx context.Context, companyID int64) map[string]any {
	now := time.Now().In(domain.MSK)
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, domain.MSK)
	start := end.AddDate(0, 0, -1)
	common, err := s.work.CommonMetrics(ctx, companyID, start, end)
	if err != nil {
		return map[string]any{}
	}
	employees, err := s.work.TopEmployees(ctx, companyID, start, end)
	if err != nil {
		return map[string]any{}
	}
	totalHours := 0.0
	for _, e := range employees {
		totalHours += e.TotalHours
	}
	result := map[string]any{
		"closed":   common.Closed,
		"received": common.Received,
		"hours":    math.Round(totalHours*10) / 10,
	}
	if len(employees) > 0 {
		result["leader_fio"] = employees[0].FIO
	}
	return result
}

func (s *Service) generateDigest(ctx context.Context, companyID int64) bool {
	if !s.ai.Enabled(ctx, companyID) {
		return false
	}
	var lines []string
	if isWeekend(time.Now().In(domain.MSK), s.weekendDays(ctx, companyID)) {
		// Выходной: вместо рабочей сводки — тёплый пост про отдых.
		lines = []string{"Сегодня у команды выходной. Напиши пост для ленты: поздравь " +
			"с заслуженным отдыхом и предложи пару идей активностей — " +
			"прогулка, хобби, спорт, время с близкими. Ни слова про задачи " +
			"и работу. 2-3 предложения, до 350 символов, живо и с юмором."}
	} else {
		dctx := s.digestContext(ctx, companyID)
		lines = []string{"Напиши утренний пост-дайджест для ленты команды: поприветствуй, " +
			"подведи итог вчерашнего дня, пожелай хорошего дня. 2-3 предложения, " +
			"до 350 символов, живо и с юмором."}
		closed, _ := dctx["closed"].(int)
		received, _ := dctx["received"].(int)
		hours, _ := dctx["hours"].(float64)
		if closed > 0 {
			lines = append(lines, "Вчера закрыто задач: "+strconvInt(closed)+".")
		}
		if received > 0 {
			lines = append(lines, "Поступило новых: "+strconvInt(received)+".")
		}
		if hours > 0 {
			lines = append(lines, "Команда отработала часов: "+formatHours(hours)+".")
		}
		if leader, ok := dctx["leader_fio"].(string); ok && leader != "" {
			lines = append(lines, "Герой вчерашнего дня — "+leader+".")
		}
		if closed == 0 && received == 0 && hours == 0 {
			lines = append(lines, "Вчера было тихо — обыграй это мягко, без упрёков.")
		}
	}
	text, err := s.ai.Chat(ctx, companyID, []map[string]any{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": strings.Join(lines, " ")},
	}, 260, 0.9, 25*time.Second)
	if err != nil {
		s.log.Warn("groove.ai.digest_failed", "company_id", companyID, "error", err)
		return false
	}
	text = trimAIReply(text)
	if text == "" {
		return false
	}
	_, err = s.recordEvent(ctx, companyID, nil, "ai_digest", map[string]any{
		"text": text, "date": time.Now().In(domain.MSK).Format("2006-01-02"),
	}, false)
	return err == nil
}
