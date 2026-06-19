package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/dto"
)

// recordEvent — единственная точка создания событий ленты: пишет, вещает
// feed:new и (опционально) асинхронно зовёт комментарий Грувика.
func (s *Service) recordEvent(ctx context.Context, companyID int64, userID *int64,
	kind string, payload map[string]any, botComment bool) (*domain.FeedEvent, error) {

	event, err := s.feed.CreateEvent(ctx, companyID, userID, kind, payload)
	if err != nil {
		return nil, err
	}
	data := dto.NewFeedEvent(event)
	data.MyReactions = []string{}
	s.pub.Publish(ctx, "feed:new", []string{"all"}, data)
	if botComment {
		s.scheduleBotComment(event.ID)
	}
	return event, nil
}

// ───────────────────────────── лента ───────────────────────────────

func (s *Service) GetFeedPage(ctx context.Context, companyID, userID,
	beforeID int64, limit int) (*dto.FeedPageDTO, error) {

	if limit <= 0 {
		limit = domain.FeedPageLimit
	}
	limit = max(1, min(limit, 100))
	events, err := s.feed.ListEvents(ctx, companyID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	counts, err := s.feed.ReactionCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	mine, err := s.feed.MyReactions(ctx, ids, userID)
	if err != nil {
		return nil, err
	}
	comments, err := s.feed.CommentCounts(ctx, ids)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.FeedEventDTO, 0, len(events))
	for _, e := range events {
		data := dto.NewFeedEvent(e)
		if c := counts[e.ID]; c != nil {
			data.Reactions = c
		}
		data.MyReactions = mine[e.ID]
		if data.MyReactions == nil {
			data.MyReactions = []string{}
		}
		data.CommentsCount = comments[e.ID]
		items = append(items, data)
	}
	return &dto.FeedPageDTO{
		Items:            items,
		HasMore:          len(events) == limit,
		AllowedReactions: domain.FeedReactions,
	}, nil
}

// ─────────────────────────── реакции ───────────────────────────────

func isAllowedReaction(emoji string) bool {
	for _, e := range domain.FeedReactions {
		if e == emoji {
			return true
		}
	}
	return false
}

func (s *Service) ToggleReaction(ctx context.Context, eventID, userID,
	companyID int64, emoji string) (map[string]any, error) {

	if !isAllowedReaction(emoji) {
		return nil, domain.NewError("BAD_EMOJI", "Недопустимая реакция", 422)
	}
	event, err := s.feed.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event == nil || event.CompanyID != companyID {
		return nil, domain.NewError("NOT_FOUND", "Событие не найдено", 404)
	}
	added, err := s.feed.ToggleReaction(ctx, eventID, userID, emoji)
	if err != nil {
		return nil, err
	}
	count, err := s.feed.ReactionCountFor(ctx, eventID, emoji)
	if err != nil {
		return nil, err
	}
	if added && event.UserID != nil && *event.UserID != userID {
		s.AwardBeans(ctx, *event.UserID, companyID, "reaction", 1)
	}
	s.pub.Publish(ctx, "feed:reaction", []string{"all"}, map[string]any{
		"event_id": eventID, "emoji": emoji, "count": count,
		"user_id": userID, "added": added, "company_id": companyID,
	})
	return map[string]any{"added": added, "count": count}, nil
}

// ─────────────────────────── комментарии ───────────────────────────

func (s *Service) ListComments(ctx context.Context, eventID, companyID int64) ([]*dto.FeedCommentDTO, error) {
	event, err := s.feed.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event == nil || event.CompanyID != companyID {
		return nil, domain.NewError("NOT_FOUND", "Событие не найдено", 404)
	}
	comments, err := s.feed.ListComments(ctx, eventID)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.FeedCommentDTO, 0, len(comments))
	for _, c := range comments {
		out = append(out, dto.NewFeedComment(c))
	}
	return out, nil
}

func (s *Service) AddComment(ctx context.Context, eventID, authorID, companyID int64,
	text string, replyToID *int64) (*dto.FeedCommentDTO, error) {

	event, err := s.feed.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event == nil || event.CompanyID != companyID {
		return nil, domain.NewError("NOT_FOUND", "Событие не найдено", 404)
	}
	if replyToID != nil {
		parent, err := s.feed.GetComment(ctx, *replyToID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.EventID != eventID {
			return nil, domain.NewError("REPLY_NOT_FOUND", "Комментарий не найден", 404)
		}
	}
	comment, err := s.feed.CreateComment(ctx, eventID, &authorID,
		strings.TrimSpace(text), replyToID, false)
	if err != nil {
		return nil, err
	}
	data := dto.NewFeedComment(comment)
	s.pub.Publish(ctx, "feed:comment", []string{"all"}, map[string]any{
		"event_id": eventID, "comment": data, "company_id": companyID,
	})
	return data, nil
}

func (s *Service) DeleteComment(ctx context.Context, commentID, userID int64, userLevel int) error {
	comment, err := s.feed.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return domain.NewError("NOT_FOUND", "Комментарий не найден", 404)
	}
	if (comment.AuthorID == nil || *comment.AuthorID != userID) && userLevel < domain.LevelAdmin {
		return domain.NewError("FORBIDDEN", "Недостаточно прав", 403)
	}
	event, err := s.feed.GetEvent(ctx, comment.EventID)
	if err != nil {
		return err
	}
	if err := s.feed.DeleteComment(ctx, commentID); err != nil {
		return err
	}
	if event != nil {
		s.pub.Publish(ctx, "feed:comment_deleted", []string{"all"}, map[string]any{
			"event_id": event.ID, "comment_id": commentID,
			"company_id": event.CompanyID,
		})
	}
	return nil
}

// ───────────────────────────── кудосы ──────────────────────────────

func (s *Service) SendKudos(ctx context.Context, companyID, fromUserID,
	toUserID int64, text string) error {

	if fromUserID == toUserID {
		return domain.NewError("SELF_KUDOS", "Нельзя благодарить самого себя", 422)
	}
	target, err := s.users.GetUser(ctx, toUserID)
	if err != nil {
		return err
	}
	if target == nil || !target.IsActive {
		return domain.NewError("USER_NOT_FOUND", "Сотрудник не найден", 404)
	}
	_, err = s.recordEvent(ctx, companyID, &fromUserID, "kudos", map[string]any{
		"to_user_id":     target.ID,
		"to_fio":         target.FIO,
		"to_avatar_path": target.AvatarPath,
		"text":           strings.TrimSpace(text),
	}, true)
	if err != nil {
		return err
	}
	s.AwardBeans(ctx, toUserID, companyID, "kudos", 2)
	return nil
}

// ─────────────────────── live и заряды энергии ─────────────────────

func (s *Service) GetLive(ctx context.Context, companyID, viewerID int64) (*dto.LiveDTO, error) {
	units, err := s.feed.ListActiveUnits(ctx, companyID)
	if err != nil {
		return nil, err
	}
	unitIDs := make([]int64, len(units))
	for i, u := range units {
		unitIDs[i] = u.ID
	}
	zaps := s.daily.GetZaps(ctx, unitIDs)
	items := make([]*dto.LiveItemDTO, 0, len(units))
	for _, u := range units {
		items = append(items, dto.NewLiveItem(u, zaps[u.ID]))
	}
	return &dto.LiveDTO{
		Items:    items,
		ZapsLeft: s.daily.Left(ctx, viewerID, "zap_sent", domain.ZapSentDailyMax),
		ZapsMax:  domain.ZapSentDailyMax,
	}, nil
}

func (s *Service) SendZap(ctx context.Context, companyID, fromUserID,
	toUserID int64) (map[string]any, error) {

	if fromUserID == toUserID {
		return nil, domain.NewError("SELF_ZAP", "Себя зарядить нельзя", 422)
	}
	unitID, unitCompanyID, err := s.work.ActiveUnitForUser(ctx, toUserID)
	if err != nil {
		return nil, err
	}
	if unitID == 0 || unitCompanyID != companyID {
		return nil, domain.NewError("NOT_LIVE", "Коллега сейчас не в эфире", 422)
	}
	if s.daily.TakeBudget(ctx, fromUserID, "zap_sent", 1, domain.ZapSentDailyMax) <= 0 {
		return nil, domain.NewError("ZAP_LIMIT", "Заряды на сегодня закончились", 429)
	}

	zaps := s.daily.IncrZap(ctx, unitID)
	s.AwardBeans(ctx, toUserID, companyID, "zap", 1)

	fromFIO := "Коллега"
	if sender, err := s.users.GetUser(ctx, fromUserID); err == nil && sender != nil {
		fromFIO = sender.FIO
	}
	s.pub.Publish(ctx, "groove:zap", []string{userRoom(toUserID)}, map[string]any{
		"from_user_id": fromUserID,
		"from_fio":     fromFIO,
		"to_user_id":   toUserID,
		"unit_id":      unitID,
		"zaps":         zaps,
	})
	s.pub.Publish(ctx, "groove:zap-count", []string{"all"}, map[string]any{
		"unit_id": unitID, "zaps": zaps, "company_id": companyID,
	})
	return map[string]any{
		"zaps":      zaps,
		"zaps_left": s.daily.Left(ctx, fromUserID, "zap_sent", domain.ZapSentDailyMax),
		"zaps_max":  domain.ZapSentDailyMax,
	}, nil
}

// ───────────────────── wrapped «Моя неделя» ────────────────────────

var weekdaysRU = []string{"понедельник", "вторник", "среда", "четверг",
	"пятница", "суббота", "воскресенье"}

func (s *Service) GetWrapped(ctx context.Context, companyID, userID int64) (map[string]any, error) {
	since := time.Now().UTC().AddDate(0, 0, -7)
	units, err := s.pets.FinishedUnitsForUser(ctx, userID, since, 300)
	if err != nil {
		return nil, err
	}

	totalMinutes := 0
	var longestName string
	longestMinutes := -1
	byDay := map[int]int{}
	var startHours []int
	for _, u := range units {
		minutes := max(0, int(u.End.Sub(u.Start).Minutes()))
		totalMinutes += minutes
		if minutes > longestMinutes {
			longestName, longestMinutes = u.Name, minutes
		}
		local := u.Start.In(domain.MSK)
		byDay[pyWeekday(local)] += minutes
		startHours = append(startHours, local.Hour())
	}

	var bestDay map[string]any
	if len(byDay) > 0 {
		bestIdx, bestMinutes := 0, -1
		for idx, minutes := range byDay {
			if minutes > bestMinutes || (minutes == bestMinutes && idx < bestIdx) {
				bestIdx, bestMinutes = idx, minutes
			}
		}
		bestDay = map[string]any{"label": weekdaysRU[bestIdx], "minutes": bestMinutes}
	}

	var peakHour *int
	if len(startHours) > 0 {
		sort.Ints(startHours)
		h := startHours[len(startHours)/2]
		peakHour = &h
	}

	closed, err := s.feed.CountUserEvents(ctx, companyID, userID, "task_closed", since)
	if err != nil {
		return nil, err
	}
	reactions, err := s.feed.ReactionsReceived(ctx, userID, since)
	if err != nil {
		return nil, err
	}
	kudos, err := s.feed.KudosReceived(ctx, companyID, userID, since)
	if err != nil {
		return nil, err
	}

	var soulmate map[string]any
	mate, mateUnits, err := s.pets.SoulmateForUser(ctx, userID, since)
	if err != nil {
		return nil, err
	}
	if mate != nil {
		soulmate = map[string]any{"user": mate, "units": mateUnits}
	}

	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}

	var longest map[string]any
	if longestMinutes >= 0 {
		longest = map[string]any{"name": longestName, "minutes": longestMinutes}
	}
	stats := map[string]any{
		"units":     len(units),
		"minutes":   totalMinutes,
		"closed":    closed,
		"longest":   longest,
		"best_day":  bestDay,
		"peak_hour": peakHour,
		"reactions": reactions,
		"kudos":     kudos,
		"soulmate":  soulmate,
		"pet": map[string]any{
			"name": pet.Name, "stage": pet.Stage, "species": pet.Species,
			"feed_streak": pet.FeedStreak, "sick": pet.SickSince != nil,
		},
	}
	stats["ai_phrase"] = s.wrappedPhrase(ctx, companyID, userID, stats)
	return stats, nil
}

func (s *Service) ShareWrapped(ctx context.Context, companyID, userID int64) error {
	key := "gw2:groove:wrapped_share:" + strconvI64(userID)
	if s.daily.Exists(ctx, key) {
		return domain.NewError("ALREADY_SHARED", "Итог недели уже опубликован сегодня", 429)
	}
	stats, err := s.GetWrapped(ctx, companyID, userID)
	if err != nil {
		return err
	}
	var bestDayLabel any
	if bd, ok := stats["best_day"].(map[string]any); ok && bd != nil {
		bestDayLabel = bd["label"]
	}
	_, err = s.recordEvent(ctx, companyID, &userID, "wrapped", map[string]any{
		"units":     stats["units"],
		"minutes":   stats["minutes"],
		"closed":    stats["closed"],
		"best_day":  bestDayLabel,
		"reactions": stats["reactions"],
		"kudos":     stats["kudos"],
	}, true)
	if err != nil {
		return err
	}
	s.daily.SetCache(ctx, key, "1", 24*time.Hour)
	return nil
}

// ──────────────── утренний брифинг от Грувика ──────────────────────

const morningStaleDays = 7

var greetings = map[string]string{
	"morning": "Доброе утро",
	"day":     "Добрый день",
	"evening": "Добрый вечер",
	"night":   "Привет",
}

// Идеи для выходного, когда AI выключен — Грувик зовёт отдыхать, не работать.
var weekendFallbacks = []string{
	"Сегодня выходной — никаких задач! Может, прогуляемся? Говорят, на улице целый мир. 🌞",
	"Выходной! План такой: вкусный завтрак, любимое хобби и ноль рабочих мыслей. 🧘",
	"Объявляю день отдыха! Фильм, плед и что-нибудь вкусное — я уже занял место рядом. 🍿",
	"Задачи подождут до будней. А нам — гулять, отдыхать и набираться сил! 🌿",
	"Сегодня по плану: отдых, отдых и ещё раз отдых. Ты заслужил — я проверял. 🎈",
}

type briefingCtx struct {
	FirstName        string
	OpenCount        int
	StaleCount       int
	Oldest           []map[string]any
	Mood             string
	PetName          string
	PersonalityTitle string
	Sick             bool
}

// morningFallback — статичная реплика, когда AI выключен/недоступен.
func (s *Service) morningFallback(ctx briefingCtx) string {
	if ctx.Mood == "weekend" {
		return weekendFallbacks[randIntn(len(weekendFallbacks))]
	}
	openWord := plural(ctx.OpenCount, "задача", "задачи", "задач")
	var oldest map[string]any
	if len(ctx.Oldest) > 0 {
		oldest = ctx.Oldest[0]
	}
	switch {
	case ctx.Mood == "sick":
		return "Я что-то расклеился, пока мы отдыхали… На нас " +
			strconvInt(ctx.OpenCount) + " " + openWord + ". " +
			"Закроем парочку — и мне сразу полегчает. 🤒"
	case ctx.Mood == "buried":
		return "Кажется, я закопался в бумагах: у нас с тобой " +
			strconvInt(ctx.OpenCount) + " " + openWord + ", и " +
			strconvInt(ctx.StaleCount) + " засиделись дольше недели. Разгребём вместе? 💪"
	case ctx.Mood == "reminder" && oldest != nil:
		days, _ := oldest["days_pending"].(int)
		name, _ := oldest["name"].(string)
		return "У нас с тобой " + strconvInt(ctx.OpenCount) + " " + openWord +
			". Вот эта засиделась (" + strconvInt(days) + " " +
			plural(days, "день", "дня", "дней") + ") — «" + truncateRunes(name, 60) + "». Глянем?"
	case ctx.Mood == "fresh":
		return strconvInt(ctx.OpenCount) + " " + openWord +
			" в работе — и все свежие, ни одна не залежалась. Красота, держим темп! ✨"
	}
	return "У нас с тобой сегодня " + strconvInt(ctx.OpenCount) + " " + openWord + ". За дело!"
}

func (s *Service) MorningBriefing(ctx context.Context, companyID, userID int64,
	part string) (map[string]any, error) {

	if _, ok := greetings[part]; !ok {
		part = "morning"
	}

	pet, err := s.pets.GetOrCreate(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	sick := pet.SickSince != nil
	personalityTitle := ""
	if pet.Personality != nil {
		if p, ok := domain.Personalities[*pet.Personality]; ok {
			personalityTitle = p.Title
		}
	}

	dayOff := isWeekend(time.Now().In(domain.MSK), s.weekendDays(ctx, companyID))

	now := time.Now().UTC()
	threshold := now.AddDate(0, 0, -morningStaleDays)
	openCount, err := s.work.CountUserActive(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}

	// Показываем когда есть о чём сказать: задачи, грустный питомец — или
	// выходной, в который Грувик зовёт отдыхать, а не работать.
	if openCount == 0 && !sick && !dayOff {
		return map[string]any{"show": false}, nil
	}

	var stale []map[string]any
	mood := "fresh"
	if dayOff {
		// В выходной за задачи не пилим: ни списка засидевшихся, ни упрёков.
		stale = []map[string]any{}
		mood = "weekend"
	} else {
		tasks, err := s.work.UserStale(ctx, userID, companyID, threshold, 20)
		if err != nil {
			return nil, err
		}
		for _, t := range tasks {
			var dept map[string]any
			if t.DepartmentName != nil {
				dept = map[string]any{"name": *t.DepartmentName}
			}
			stale = append(stale, map[string]any{
				"id":           t.ID,
				"name":         t.Name,
				"department":   dept,
				"days_pending": int(now.Sub(t.ReceivedAt.UTC()).Hours() / 24),
			})
		}
		switch {
		case sick:
			mood = "sick"
		case len(stale) >= 3:
			mood = "buried"
		case len(stale) >= 1:
			mood = "reminder"
		}
	}

	name := "коллега"
	if user, err := s.users.GetUser(ctx, userID); err == nil && user != nil {
		name = firstName(user.FIO)
	}

	bctx := briefingCtx{
		FirstName:        name,
		OpenCount:        openCount,
		StaleCount:       len(stale),
		Oldest:           staleHead(stale, 3),
		Mood:             mood,
		PetName:          pet.Name,
		PersonalityTitle: personalityTitle,
		Sick:             sick,
	}
	message := s.morningPhrase(ctx, companyID, userID, bctx)
	ai := message != ""
	if message == "" {
		message = s.morningFallback(bctx)
	}

	return map[string]any{
		"show":        true,
		"part":        part,
		"greeting":    greetings[part],
		"first_name":  name,
		"open_count":  openCount,
		"stale_count": len(stale),
		"stale":       staleHead(stale, 6),
		"mood":        mood,
		"message":     message,
		"ai":          ai,
		"pet": map[string]any{
			"name":    pet.Name,
			"species": pet.Species,
			"stage":   pet.Stage,
			"sick":    sick,
			"hat":     pet.Hat,
		},
	}, nil
}

func staleHead(stale []map[string]any, n int) []map[string]any {
	if stale == nil {
		return []map[string]any{}
	}
	if len(stale) > n {
		return stale[:n]
	}
	return stale
}
