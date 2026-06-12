// Package dto — формы REST-ответов, байт-в-байт совместимые с прежними
// marshmallow-схемами Flask (schemas/groove.py + dump_pet/get_feed_page).
package dto

import (
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

// ISO-форматы как у marshmallow: datetime с офсетом, date — YYYY-MM-DD.
func isoTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000+00:00") }
func isoDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

type FeedEventDTO struct {
	ID            int64           `json:"id"`
	CompanyID     int64           `json:"company_id"`
	Kind          string          `json:"kind"`
	Payload       map[string]any  `json:"payload"`
	CreatedAt     string          `json:"created_at"`
	User          *domain.UserRef `json:"user"`
	Reactions     map[string]int  `json:"reactions"`
	MyReactions   []string        `json:"my_reactions,omitempty"`
	CommentsCount int             `json:"comments_count"`
}

func NewFeedEvent(e *domain.FeedEvent) *FeedEventDTO {
	payload := e.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	return &FeedEventDTO{
		ID:        e.ID,
		CompanyID: e.CompanyID,
		Kind:      e.Kind,
		Payload:   payload,
		CreatedAt: isoTime(e.CreatedAt),
		User:      e.User,
		Reactions: map[string]int{},
	}
}

type FeedCommentDTO struct {
	ID        int64           `json:"id"`
	EventID   int64           `json:"event_id"`
	Text      string          `json:"text"`
	IsBot     bool            `json:"is_bot"`
	ReplyToID *int64          `json:"reply_to_id"`
	CreatedAt string          `json:"created_at"`
	Author    *domain.UserRef `json:"author"`
}

func NewFeedComment(c *domain.FeedComment) *FeedCommentDTO {
	return &FeedCommentDTO{
		ID:        c.ID,
		EventID:   c.EventID,
		Text:      c.Text,
		IsBot:     c.IsBot,
		ReplyToID: c.ReplyToID,
		CreatedAt: isoTime(c.CreatedAt),
		Author:    c.Author,
	}
}

type FeedPageDTO struct {
	Items            []*FeedEventDTO `json:"items"`
	HasMore          bool            `json:"has_more"`
	AllowedReactions []string        `json:"allowed_reactions"`
}

type QuestDTO struct {
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Hint     string `json:"hint"`
	Unit     string `json:"unit"`
	Target   int    `json:"target"`
	Progress int    `json:"progress"`
	Done     bool   `json:"done"`
	Claimed  bool   `json:"claimed"`
	Reward   int    `json:"reward"`
}

// PetDTO — PetSchema + расширения dump_pet. Контекстные поля (feeds_left,
// phrase, evolved, strokes_today…) добавляются по месту использования.
type PetDTO struct {
	UserID           int64           `json:"user_id"`
	Name             string          `json:"name"`
	Species          string          `json:"species"`
	Stage            int             `json:"stage"`
	XP               int             `json:"xp"`
	Beans            int             `json:"beans"`
	Hat              *string         `json:"hat"`
	Accessories      []string        `json:"accessories"`
	FeedStreak       int             `json:"feed_streak"`
	LastFedDate      *string         `json:"last_fed_date"`
	User             *domain.UserRef `json:"user,omitempty"`
	NextStageXP      *int            `json:"next_stage_xp"`
	Sick             bool            `json:"sick"`
	Recovery         int             `json:"recovery"`
	RecoveryTarget   int             `json:"recovery_target"`
	Personality      *string         `json:"personality"`
	PersonalityTitle *string         `json:"personality_title"`
	UnlockedSpecies  []string        `json:"unlocked_species"`
	Quest            *QuestDTO       `json:"quest"`

	// Контекстные поля.
	FeedsLeft    *int    `json:"feeds_left,omitempty"`
	FeedsMax     *int    `json:"feeds_max,omitempty"`
	Phrase       *string `json:"phrase,omitempty"`
	Evolved      *bool   `json:"evolved,omitempty"`
	Recovered    *bool   `json:"recovered,omitempty"`
	StrokesToday *int    `json:"strokes_today,omitempty"`
	StrokedByMe  *bool   `json:"stroked_by_me,omitempty"`
}

// NewPet — порт dump_pet из pet_service.py.
func NewPet(p *domain.Pet) *PetDTO {
	dto := &PetDTO{
		UserID:         p.UserID,
		Name:           p.Name,
		Species:        p.Species,
		Stage:          p.Stage,
		XP:             p.XP,
		Beans:          p.Beans,
		Hat:            p.Hat,
		Accessories:    orEmpty(p.Accessories),
		FeedStreak:     p.FeedStreak,
		LastFedDate:    isoDate(p.LastFedDate),
		User:           p.User,
		Sick:           p.SickSince != nil,
		Recovery:       p.Recovery,
		RecoveryTarget: domain.RecoveryTarget,
		Personality:    p.Personality,
	}
	if p.Stage < domain.MaxStage {
		next := domain.StageXP[p.Stage+1]
		dto.NextStageXP = &next
	}
	if p.Personality != nil {
		if pers, ok := domain.Personalities[*p.Personality]; ok {
			title := pers.Title
			dto.PersonalityTitle = &title
		}
	}
	// Доступные облики: всё разблокированное + текущий вид (старые питомцы
	// до миграции могли не иметь его в unlocked).
	unlocked := append([]string{}, p.UnlockedSpecies...)
	if p.Species != "" && p.Species != "egg" && !contains(unlocked, p.Species) {
		unlocked = append(unlocked, p.Species)
	}
	dto.UnlockedSpecies = orEmpty(unlocked)
	dto.Quest = questSnapshot(p)
	return dto
}

func questSnapshot(p *domain.Pet) *QuestDTO {
	if p.QuestKind == nil || p.QuestTarget == nil || *p.QuestTarget == 0 {
		return nil
	}
	var tpl *domain.QuestTemplate
	for i := range domain.QuestTemplates {
		if domain.QuestTemplates[i].Kind == *p.QuestKind {
			tpl = &domain.QuestTemplates[i]
			break
		}
	}
	q := &QuestDTO{
		Kind:    *p.QuestKind,
		Title:   "Дневной квест",
		Target:  *p.QuestTarget,
		Claimed: p.QuestClaimed,
		Reward:  domain.QuestRewardBeans,
	}
	if tpl != nil {
		q.Title, q.Hint, q.Unit = tpl.Title, tpl.Hint, tpl.Unit
	}
	q.Progress = min(p.QuestProgress, q.Target)
	q.Done = q.Progress >= q.Target
	return q
}

type LiveItemDTO struct {
	UnitID    int64           `json:"unit_id"`
	UnitName  string          `json:"unit_name"`
	TaskID    int64           `json:"task_id"`
	TaskName  *string         `json:"task_name"`
	StartedAt string          `json:"started_at"`
	User      *domain.UserRef `json:"user"`
	Zaps      int             `json:"zaps"`
}

type LiveDTO struct {
	Items    []*LiveItemDTO `json:"items"`
	ZapsLeft int            `json:"zaps_left"`
	ZapsMax  int            `json:"zaps_max"`
}

type RaidDTO struct {
	ID        int64  `json:"id"`
	Boss      string `json:"boss"`
	Target    int    `json:"target"`
	Progress  int    `json:"progress"`
	Reward    string `json:"reward"`
	Defeated  bool   `json:"defeated"`
	WeekStart string `json:"week_start"`
	DaysLeft  int    `json:"days_left"`
}

func NewLiveItem(u *domain.ActiveUnit, zaps int) *LiveItemDTO {
	return &LiveItemDTO{
		UnitID:    u.ID,
		UnitName:  u.Name,
		TaskID:    u.TaskID,
		TaskName:  u.TaskName,
		StartedAt: isoTime(u.StartedAt),
		User:      u.User,
		Zaps:      zaps,
	}
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
