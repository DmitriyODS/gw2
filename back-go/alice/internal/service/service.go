// Package service — диалоговая логика навыка Алисы: авторизация по
// access-токену из связки аккаунтов, интент-парсер и сценарии над
// tasksvc/diarysvc/notesvc (gRPC). Мультиход — через session_state Диалогов
// (сервер stateless).
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

const (
	protocolVersion = "1.0"
	// maxSpeech — потолок текста реплики (лимит Диалогов — 1024 символа).
	maxSpeech = 950
)

type Deps struct {
	Tasks    domain.TasksClient
	Diary    domain.DiaryClient
	Notes    domain.NotesClient
	AI       domain.IntentParser // nil — только классический разбор
	Verifier *pasetoauth.Verifier
	Log      *slog.Logger
}

type Service struct {
	tasks    domain.TasksClient
	diary    domain.DiaryClient
	notes    domain.NotesClient
	ai       domain.IntentParser
	verifier *pasetoauth.Verifier
	log      *slog.Logger
}

func New(d Deps) *Service {
	return &Service{tasks: d.Tasks, diary: d.Diary, notes: d.Notes, ai: d.AI, verifier: d.Verifier, log: d.Log}
}

// validKinds — интенты, которые вправе вернуть ИИ-разбор.
var validKinds = map[string]bool{
	"help": true, "task_create": true, "task_close": true, "task_list": true,
	"unit_start": true, "unit_stop": true, "unit_status": true,
	"diary_create": true, "diary_add": true, "diary_list": true, "diary_done": true,
	"diary_move": true, "diary_delete": true, "note_create": true, "note_append": true,
	"note_read": true, "note_delete": true, "folder_create": true, "unknown": true,
}

// parseIntent — ИИ-разбор фразы ключом активной компании (точность и
// гибкость), фолбэк — классический регэксп-парсер: нет ИИ на аккаунте
// (AI_DISABLED), нет компании, таймаут или мусорный ответ.
func (s *Service) parseIntent(ctx context.Context, sess *session, command, original string) Intent {
	classic := Parse(command, sess.now)
	if s.ai == nil || sess.companyID == 0 {
		return classic
	}
	utterance := strings.TrimSpace(original)
	if utterance == "" {
		utterance = command
	}
	it, err := s.ai.ParseIntent(ctx, sess.companyID, utterance, sess.now)
	if err != nil || it == nil || !validKinds[it.Kind] {
		if err != nil {
			s.log.Warn("alice.ai_parse_fallback", "error", err)
		}
		return classic
	}
	// «Не понял» от ИИ не должен прятать команду, которую знает классика.
	if it.Kind == "unknown" && classic.Kind != "unknown" {
		return classic
	}
	return *it
}

// session — контекст одного запроса вебхука.
type session struct {
	userID    int64
	companyID int64 // 0 — активной компании нет
	now       time.Time
	state     domain.DialogState
}

const greeting = "Привет! Я — Groove Work. Могу добавлять и закрывать задачи, запускать юниты, " +
	"вести ежедневник и заметки. Например: «добавь задачу подготовить отчёт», " +
	"«запиши на завтра позвонить клиенту» или «создай заметку идеи». Скажите «помощь» — расскажу подробнее."

const helpText = "Вот что я умею.\n" +
	"Задачи: «добавь задачу …», «закрой задачу …», «мои задачи», «начни работу над …», «останови работу».\n" +
	"Ежедневник: «запиши на завтра …», «что у меня на сегодня», «отметь … выполненным», «перенеси … на пятницу», «удали запись …», «создай ежедневник …».\n" +
	"Заметки: «создай заметку … с текстом …», «допиши в заметку … текст …», «прочитай заметку …», «удали заметку …», «создай папку …»."

const linkGreeting = "Привет! Я — Groove Work: задачи, ежедневник и заметки голосом. " +
	"Чтобы я работала с вашими данными, нужно один раз связать аккаунт — " +
	"просто скажите любую команду, например «мои задачи», и я пришлю ссылку для входа."

// Handle — обработка одного запроса вебхука; ошибки сервисов превращаются в
// голосовые реплики, транспорту всегда отдаётся валидный ответ.
func (s *Service) Handle(ctx context.Context, req *domain.WebhookRequest) *domain.WebhookResponse {
	token := req.Session.User.AccessToken
	claims := s.verifier.ParseAccess(token)
	if token == "" || claims.UserID == 0 {
		// Новая сессия без связанного аккаунта — обычный текстовый ответ:
		// его же ждёт валидатор Диалогов при сохранении Webhook URL
		// (директива start_account_linking до настройки связки — «ошибка
		// сервера» и завал модерации). Директива уходит на первую команду.
		if req.Session.New {
			return reply(linkGreeting)
		}
		return &domain.WebhookResponse{StartAccountLinking: &struct{}{}, Version: protocolVersion}
	}

	loc, err := time.LoadLocation(req.Meta.Timezone)
	if err != nil || req.Meta.Timezone == "" {
		loc = time.FixedZone("MSK", 3*3600)
	}
	sess := &session{userID: claims.UserID, now: time.Now().In(loc)}
	if claims.CompanyID != nil {
		sess.companyID = *claims.CompanyID
	}
	if len(req.State.Session) > 0 {
		_ = json.Unmarshal(req.State.Session, &sess.state)
	}

	cmd := normalize(req.Request.Command)
	if req.Session.New && cmd == "" {
		return reply(greeting)
	}

	// Уточняющие ответы (да/нет/номер варианта) разбираются без ИИ.
	if sess.state.Pending != "" {
		return s.handlePending(ctx, sess, cmd)
	}

	intent := s.parseIntent(ctx, sess, cmd, req.Request.OriginalUtterance)
	return s.dispatch(ctx, sess, intent)
}

func (s *Service) dispatch(ctx context.Context, sess *session, it Intent) *domain.WebhookResponse {
	switch it.Kind {
	case "greet":
		return reply(greeting)
	case "help":
		return reply(helpText)
	case "yes", "no":
		return reply("Сейчас подтверждать нечего. Скажите «помощь», если нужна подсказка.")

	case "task_create":
		return s.taskCreate(ctx, sess, it.Title, 0)
	case "task_close":
		return s.taskClose(ctx, sess, it.Title)
	case "task_list":
		return s.taskList(ctx, sess)
	case "unit_start":
		return s.unitStart(ctx, sess, it.Title)
	case "unit_stop":
		return s.unitStop(ctx, sess)
	case "unit_status":
		return s.unitStatus(ctx, sess)

	case "diary_create":
		return s.diaryCreate(ctx, sess, it.Title)
	case "diary_add":
		return s.diaryAdd(ctx, sess, it.Title, it.Date)
	case "diary_list":
		return s.diaryList(ctx, sess, it.Date)
	case "diary_done":
		return s.diaryDone(ctx, sess, it.Title)
	case "diary_move":
		return s.diaryMove(ctx, sess, it.Title, it.Date)
	case "diary_delete":
		return s.diaryDelete(ctx, sess, it.Title)

	case "note_create":
		return s.noteCreate(ctx, sess, it.Title, it.Text)
	case "note_append":
		return s.noteAppend(ctx, sess, it.Title, it.Text)
	case "note_read":
		return s.noteRead(ctx, sess, it.Title)
	case "note_delete":
		return s.noteDelete(ctx, sess, it.Title)
	case "folder_create":
		return s.folderCreate(ctx, sess, it.Title)
	}
	return reply("Я не поняла команду. Скажите «помощь» — перечислю, что умею.")
}

// handlePending — ответ на уточняющий вопрос (выбор варианта, да/нет, текст).
func (s *Service) handlePending(ctx context.Context, sess *session, cmd string) *domain.WebhookResponse {
	st := sess.state
	if reNo.MatchString(cmd) || cmd == "стоп" {
		return reply("Хорошо, отменила.")
	}

	switch st.Pending {
	case "confirm_close_task":
		if reYes.MatchString(cmd) {
			return s.taskCloseByID(ctx, sess, st.TaskID)
		}
	case "confirm_delete_note":
		if reYes.MatchString(cmd) {
			if err := s.notes.DeleteNote(ctx, sess.userID, st.NoteID); err != nil {
				return s.errReply(err)
			}
			return reply(fmt.Sprintf("Удалила заметку «%s».", st.Title))
		}
	case "confirm_delete_entry":
		if reYes.MatchString(cmd) {
			if err := s.diary.DeleteEntry(ctx, sess.userID, st.DiaryID, st.EntryID); err != nil {
				return s.errReply(err)
			}
			return reply(fmt.Sprintf("Удалила запись «%s».", st.Title))
		}
	case "choose_department":
		if opt := matchOption(cmd, st.Options); opt != nil {
			return s.taskCreate(ctx, sess, st.Title, opt.ID)
		}
		return replyState("Не поняла. Назовите отдел из списка или его номер, либо скажите «отмена».", st)
	case "choose_unit_type":
		if opt := matchOption(cmd, st.Options); opt != nil {
			return s.unitStartOnTask(ctx, sess, st.TaskID, st.Title, opt.ID)
		}
		return replyState("Не поняла. Назовите тип юнита из списка или его номер, либо скажите «отмена».", st)
	case "choose_task":
		if opt := matchOption(cmd, st.Options); opt != nil {
			switch st.Kind {
			case "task_close":
				return s.taskCloseByID(ctx, sess, opt.ID)
			case "unit_start":
				return s.unitStartOnTask(ctx, sess, opt.ID, opt.Name, 0)
			}
		}
		return replyState("Не поняла. Назовите задачу из списка или её номер, либо скажите «отмена».", st)
	case "choose_diary":
		if opt := matchOption(cmd, st.Options); opt != nil {
			return s.diaryAddTo(ctx, sess, opt.ID, st.Title, st.Date)
		}
		return replyState("Не поняла. Назовите ежедневник из списка или его номер, либо скажите «отмена».", st)
	case "choose_note":
		if opt := matchOption(cmd, st.Options); opt != nil {
			switch st.Kind {
			case "note_append":
				return s.noteAppendTo(ctx, sess, opt.ID, opt.Name, st.Text)
			case "note_read":
				return s.noteReadByID(ctx, sess, opt.ID)
			case "note_delete":
				st2 := domain.DialogState{Pending: "confirm_delete_note", NoteID: opt.ID, Title: opt.Name}
				return replyState(fmt.Sprintf("Удалить заметку «%s»? Скажите «да» или «нет».", opt.Name), st2)
			}
		}
		return replyState("Не поняла. Назовите заметку из списка или её номер, либо скажите «отмена».", st)
	case "choose_entry":
		if eo := matchEntryOption(cmd, st.EntryOptions); eo != nil {
			switch st.Kind {
			case "diary_done":
				return s.entryDone(ctx, sess, eo.DiaryID, eo.EntryID)
			case "diary_move":
				return s.entryMove(ctx, sess, eo.DiaryID, eo.EntryID, eo.Name, st.Date)
			case "diary_delete":
				st2 := domain.DialogState{Pending: "confirm_delete_entry",
					DiaryID: eo.DiaryID, EntryID: eo.EntryID, Title: eo.Name}
				return replyState(fmt.Sprintf("Удалить запись «%s»? Скажите «да» или «нет».", eo.Name), st2)
			}
		}
		return replyState("Не поняла. Назовите запись из списка или её номер, либо скажите «отмена».", st)
	case "append_text":
		if cmd == "" {
			return replyState("Скажите текст, который дописать, либо «отмена».", st)
		}
		return s.noteAppendTo(ctx, sess, st.NoteID, st.Title, cmd)
	case "move_date":
		if date, _, ok := ExtractDate(cmd, sess.now); ok {
			return s.entryMove(ctx, sess, st.DiaryID, st.EntryID, st.Title, date)
		}
		return replyState("Не поняла день. Скажите, например, «на завтра» или «на 15 июля», либо «отмена».", st)
	}
	return reply("Хорошо, отменила.")
}

// ── Хелперы ответов ──

func reply(text string) *domain.WebhookResponse {
	return &domain.WebhookResponse{
		Response: &domain.Response{Text: truncate(text, maxSpeech)},
		Version:  protocolVersion,
	}
}

func replyState(text string, st domain.DialogState) *domain.WebhookResponse {
	r := reply(text)
	r.SessionState = &st
	return r
}

// errReply — доменная ошибка сервиса-владельца человеческой репликой.
func (s *Service) errReply(err error) *domain.WebhookResponse {
	if de := domain.AsDomainError(err); de != nil && de.Message != "" {
		return reply(de.Message)
	}
	s.log.Error("alice.upstream_failed", "error", err)
	return reply("Что-то пошло не так, попробуйте ещё раз чуть позже.")
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func matchOption(answer string, opts []domain.Option) *domain.Option {
	if n := ParseChoiceIndex(answer); n > 0 && n <= len(opts) {
		return &opts[n-1]
	}
	for i := range opts {
		if wordsMatch(answer, opts[i].Name) || wordsMatch(opts[i].Name, answer) {
			return &opts[i]
		}
	}
	return nil
}

func matchEntryOption(answer string, opts []domain.EntryOption) *domain.EntryOption {
	if n := ParseChoiceIndex(answer); n > 0 && n <= len(opts) {
		return &opts[n-1]
	}
	for i := range opts {
		if wordsMatch(answer, opts[i].Name) || wordsMatch(opts[i].Name, answer) {
			return &opts[i]
		}
	}
	return nil
}

// enumerate — нумерованный список вариантов для реплики.
func enumerate(names []string) string {
	var b strings.Builder
	for i, n := range names {
		fmt.Fprintf(&b, "\n%d. %s", i+1, n)
	}
	return b.String()
}
