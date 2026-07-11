package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// Разрешённые MIME-категории вложений (как в messenger_service).
var allowedMimePrefixes = []string{"image/", "audio/", "video/", "application/", "text/"}

// ListMessages — сообщения диалога без скрытых на стороне пользователя.
func (s *Service) ListMessages(ctx context.Context, convID, userID int64,
	beforeID, afterID *int64, limit int) ([]*dto.Message, error) {

	conv, err := s.conversationForUser(ctx, convID, userID)
	if err != nil {
		return nil, err
	}
	msgs, err := s.repo.ListMessages(ctx, conv.ID, conv.Side(userID), beforeID, afterID, limit)
	if err != nil {
		return nil, err
	}
	return dto.NewMessages(msgs), nil
}

// SendMessage — отправка сообщения + все сокет-события (включая автоответ
// техподдержки).
func (s *Service) SendMessage(ctx context.Context, convID, senderID int64,
	req dto.MessageCreate) (*dto.Message, error) {

	conv, err := s.conversationForUser(ctx, convID, senderID)
	if err != nil {
		return nil, err
	}

	var text *string
	if req.Text != nil {
		if t := strings.TrimSpace(*req.Text); t != "" {
			text = &t
		}
	}
	attachmentIDs := req.AttachmentIDs
	if text == nil && len(attachmentIDs) == 0 && req.TaskID == nil {
		return nil, domain.NewError("EMPTY_MESSAGE", "Пустое сообщение", 400)
	}

	// Все вложения должны принадлежать отправителю и быть свободными.
	for _, attID := range attachmentIDs {
		att, err := s.repo.GetAttachment(ctx, attID)
		if err != nil {
			return nil, err
		}
		if att == nil || att.UploaderID != senderID || att.MessageID != nil {
			return nil, domain.NewError("BAD_ATTACHMENT", "Недопустимое вложение", 400)
		}
	}

	// Ответ должен указывать на сообщение этого же диалога.
	if req.ReplyToID != nil {
		target, err := s.repo.GetMessage(ctx, *req.ReplyToID)
		if err != nil {
			return nil, err
		}
		if target == nil || target.ConversationID != conv.ID {
			return nil, domain.NewError("BAD_REPLY", "Недопустимый ответ", 400)
		}
	}

	// Прикреплённая задача: из той же компании, что и диалог.
	kind := domain.KindText
	if req.TaskID != nil {
		task, err := s.repo.GetTask(ctx, *req.TaskID)
		if err != nil {
			return nil, err
		}
		if task == nil {
			return nil, domain.NewError("TASK_NOT_FOUND", "Задача не найдена", 404)
		}
		if conv.CompanyID != nil && task.CompanyID != *conv.CompanyID {
			return nil, domain.NewError("TASK_WRONG_COMPANY", "Задача из другой компании", 400)
		}
		kind = domain.KindTask
	}

	// Ответ супер-админа в dev-чате — спец-kind: фронт рисует badge
	// «Разработчики». Для kind='task' плашка задачи приоритетнее.
	if conv.IsDevChat && kind == domain.KindText {
		sender, err := s.users.GetUser(ctx, senderID)
		if err != nil {
			return nil, err
		}
		if sender != nil && sender.IsSuperAdmin {
			kind = domain.KindDevReply
		}
	}

	msg, err := s.repo.CreateMessage(ctx, domain.NewMessage{
		ConversationID: conv.ID,
		SenderID:       &senderID,
		Text:           text,
		AttachmentIDs:  attachmentIDs,
		ReplyToID:      req.ReplyToID,
		Kind:           kind,
		TaskID:         req.TaskID,
	})
	if err != nil {
		return nil, err
	}
	s.log.Info("message.send", "conversation_id", conv.ID,
		"sender_id", senderID, "message_id", msg.ID)

	payload := dto.NewMessage(msg)
	event := dto.MessageNewEvent{ConversationID: conv.ID, Message: payload, FromUserID: &senderID}

	switch {
	case conv.IsDevChat:
		// Спец-чат: уведомляем владельца и всех Администраторов системы.
		devIDs, err := s.users.DevChatUserIDs(ctx, conv.UserAID)
		if err != nil {
			return nil, err
		}
		s.pub.Publish(ctx, "message:new", rooms(devIDs...), event)
		if s.ai != nil && msg.Text != nil {
			// ИИ-поддержка: LLM-вызов занимает секунды — отвечаем фоном,
			// HTTP-ответ отправителю не ждёт. Сообщение без текста (только
			// вложения) ИИ не понять — им занимается канированная ветка.
			s.scheduleSupportAIReply(conv, msg, devIDs)
		} else {
			auto, err := s.maybeSupportAutoReply(ctx, conv, msg)
			if err != nil {
				return nil, err
			}
			if auto != nil {
				s.pub.Publish(ctx, "message:new", rooms(devIDs...), dto.MessageNewEvent{
					ConversationID: conv.ID, Message: dto.NewMessage(auto), FromUserID: nil,
				})
			}
		}
	default:
		recipientID := conv.OtherUserID(senderID)
		// Получателю + эхо отправителю (другие его вкладки/устройства).
		s.pub.Publish(ctx, "message:new", rooms(*recipientID, senderID), event)
	}
	return payload, nil
}

// maybeSupportAutoReply — автоответ техподдержки: если владелец dev-чата не
// общался с поддержкой последние сутки (любые сообщения людей), бот отвечает,
// что обращение передано разработчикам.
func (s *Service) maybeSupportAutoReply(ctx context.Context, conv *domain.Conversation,
	msg *domain.Message) (*domain.Message, error) {

	if !conv.IsDevChat || msg.SenderID == nil || *msg.SenderID != conv.UserAID {
		return nil, nil
	}
	since := time.Now().UTC().Add(-SupportAutoReplyAfter)
	has, err := s.repo.HasHumanMessageSince(ctx, conv.ID, since, msg.ID)
	if err != nil || has {
		return nil, err
	}
	text := SupportAutoReplyText
	reply, err := s.repo.CreateMessage(ctx, domain.NewMessage{
		ConversationID: conv.ID,
		Text:           &text,
		Kind:           domain.KindDevReply,
		IsBot:          true,
	})
	if err != nil {
		return nil, err
	}
	s.log.Info("message.support_auto_reply", "conversation_id", conv.ID, "message_id", reply.ID)
	return reply, nil
}

// scheduleSupportAIReply — фоновый ответ ИИ техподдержки на сообщение
// владельца dev-чата. Ошибки ИИ не фатальны: откат на канированный автоответ
// (те же «сутки тишины»).
func (s *Service) scheduleSupportAIReply(conv *domain.Conversation, msg *domain.Message, devIDs []int64) {
	if msg.SenderID == nil || *msg.SenderID != conv.UserAID {
		return // реплики поддержки бот не комментирует
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), supportReplyTimeout)
		defer cancel()
		reply, err := s.supportAIReply(ctx, conv, msg)
		if err != nil {
			s.log.Warn("support.ai_reply_failed", "conversation_id", conv.ID, "error", err)
			reply, err = s.maybeSupportAutoReply(ctx, conv, msg)
			if err != nil {
				s.log.Warn("support.auto_reply_failed", "conversation_id", conv.ID, "error", err)
				return
			}
		}
		if reply == nil {
			return
		}
		s.pub.Publish(ctx, "message:new", rooms(devIDs...), dto.MessageNewEvent{
			ConversationID: conv.ID, Message: dto.NewMessage(reply), FromUserID: nil,
		})
	}()
}

// supportAIReply — синхронное ядро ИИ-ответа поддержки: молчание (nil, nil),
// если человек-поддержка недавно отвечал (не влезаем в живой диалог); иначе
// история диалога → aisvc → бот-сообщение kind='system_dev_reply'.
func (s *Service) supportAIReply(ctx context.Context, conv *domain.Conversation,
	msg *domain.Message) (*domain.Message, error) {

	busy, err := s.repo.HasSupportHumanReplySince(ctx, conv.ID,
		time.Now().UTC().Add(-SupportHumanLullPeriod))
	if err != nil {
		return nil, err
	}
	if busy {
		return nil, nil
	}
	history, err := s.repo.ListMessages(ctx, conv.ID, conv.Side(conv.UserAID),
		nil, nil, supportHistoryLimit)
	if err != nil {
		return nil, err
	}
	messagesJSON, err := supportHistoryJSON(conv, history, msg)
	if err != nil {
		return nil, err
	}
	content, err := s.ai.SupportReply(ctx, messagesJSON)
	if err != nil {
		return nil, err
	}
	reply, err := s.repo.CreateMessage(ctx, domain.NewMessage{
		ConversationID: conv.ID,
		Text:           &content,
		Kind:           domain.KindDevReply,
		IsBot:          true,
	})
	if err != nil {
		return nil, err
	}
	s.log.Info("message.support_ai_reply", "conversation_id", conv.ID, "message_id", reply.ID)
	return reply, nil
}

// supportHistoryJSON — история dev-чата в формате OpenAI: владелец — user,
// поддержка (бот и люди) — assistant; только текстовые реплики, системный
// промпт добавляет aisvc. Свежее msg гарантированно попадает в конец (история
// из БД в теории может его не содержать из-за гонки).
func supportHistoryJSON(conv *domain.Conversation, history []*domain.Message,
	msg *domain.Message) (string, error) {

	type turn struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	turns := make([]turn, 0, len(history)+1)
	seen := false
	for _, m := range history {
		if m.Text == nil || *m.Text == "" {
			continue
		}
		if m.Kind != domain.KindText && m.Kind != domain.KindDevReply {
			continue
		}
		role := "assistant"
		if m.SenderID != nil && *m.SenderID == conv.UserAID {
			role = "user"
		}
		if m.ID == msg.ID {
			seen = true
		}
		turns = append(turns, turn{Role: role, Content: *m.Text})
	}
	if !seen && msg.Text != nil && *msg.Text != "" {
		turns = append(turns, turn{Role: "user", Content: *msg.Text})
	}
	if len(turns) == 0 {
		return "", domain.NewError("EMPTY_MESSAGE", "Нет текста для ИИ", 400)
	}
	b, err := json.Marshal(turns)
	return string(b), err
}

// ForwardMessage — пересылка в диалоги/пользователям: текст и файлы
// копируются (файлы — физически), плашки звонка и поста остаются плашками.
func (s *Service) ForwardMessage(ctx context.Context, senderID, messageID int64,
	conversationIDs, userIDs []int64) ([]dto.ForwardResult, error) {

	src, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, errMsgNotFound()
	}
	srcConv, err := s.repo.GetConversation(ctx, src.ConversationID)
	if err != nil {
		return nil, err
	}
	if srcConv == nil {
		return nil, errConvNotFound()
	}
	if err := s.ensureMember(ctx, srcConv, senderID); err != nil {
		return nil, err
	}

	type created struct {
		conv *domain.Conversation
		msg  *domain.Message
	}
	var results []created

	// Одна транзакция на диалоги и сообщения: иначе при ошибке в БД
	// остаётся пустой диалог без пересланного сообщения.
	err = s.repo.RunInTx(ctx, func(ctx context.Context) error {
		var targets []*domain.Conversation
		seen := map[int64]bool{}
		for _, cid := range conversationIDs {
			conv, err := s.conversationForUser(ctx, cid, senderID)
			if err != nil {
				return err
			}
			// Пересылать в соло-чат (техподдержку) смысла нет.
			if conv.IsSolo() || seen[conv.ID] {
				continue
			}
			targets = append(targets, conv)
			seen[conv.ID] = true
		}
		for _, uid := range userIDs {
			conv, err := s.ensureConversation(ctx, senderID, uid)
			if err != nil {
				return err
			}
			if seen[conv.ID] {
				continue
			}
			targets = append(targets, conv)
			seen[conv.ID] = true
		}
		if len(targets) == 0 {
			return domain.NewError("NO_TARGET", "Не выбран получатель", 400)
		}

		// Автор оригинала — кого показать в метке «Переслано от …».
		originUserID := src.ForwardedFromUserID
		if originUserID == nil {
			originUserID = src.SenderID
		}

		for _, conv := range targets {
			var newAttIDs []int64
			for i := range src.Attachments {
				att, err := s.copyAttachment(ctx, &src.Attachments[i], senderID)
				if err != nil {
					return err
				}
				newAttIDs = append(newAttIDs, att.ID)
			}
			nm := domain.NewMessage{
				ConversationID:      conv.ID,
				SenderID:            &senderID,
				Text:                src.Text,
				AttachmentIDs:       newAttIDs,
				ForwardedFromUserID: originUserID,
				Kind:                domain.KindText,
			}
			switch src.Kind {
			case domain.KindCall:
				nm.Kind, nm.CallID = domain.KindCall, src.CallID
			case domain.KindPost:
				// Плашка поста остаётся плашкой: замороженное превью
				// копируется вместе с kind (post_id может быть уже пуст,
				// превью — самодостаточный снапшот).
				nm.Kind, nm.PostID = domain.KindPost, src.PostID
				if src.Post != nil {
					nm.PostTitle = &src.Post.Title
					nm.PostExcerpt = &src.Post.Excerpt
					nm.PostCoverURL = src.Post.CoverURL
				}
			}
			msg, err := s.repo.CreateMessage(ctx, nm)
			if err != nil {
				return err
			}
			results = append(results, created{conv: conv, msg: msg})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	targetIDs := make([]int64, 0, len(results))
	out := make([]dto.ForwardResult, 0, len(results))
	for _, r := range results {
		targetIDs = append(targetIDs, r.conv.ID)
		payload := dto.NewMessage(r.msg)
		out = append(out, dto.ForwardResult{ConversationID: r.conv.ID, Message: payload})
		recipientID := r.conv.OtherUserID(senderID)
		s.pub.Publish(ctx, "message:new", rooms(*recipientID, senderID), dto.MessageNewEvent{
			ConversationID: r.conv.ID, Message: payload, FromUserID: &senderID,
		})
	}
	s.log.Info("message.forward", "source_message_id", messageID,
		"sender_id", senderID, "targets", targetIDs)
	return out, nil
}

// copyAttachment — физическая копия файла + новая запись вложения
// (message_id проставится при CreateMessage).
func (s *Service) copyAttachment(ctx context.Context, att *domain.Attachment,
	uploaderID int64) (*domain.Attachment, error) {

	newPath, err := s.files.Copy(att.FilePath)
	if err != nil {
		return nil, domain.NewError("COPY_FAILED", "Не удалось скопировать вложение", 500)
	}
	copied := &domain.Attachment{
		UploaderID: uploaderID,
		FilePath:   newPath,
		FileName:   att.FileName,
		MimeType:   att.MimeType,
		SizeBytes:  att.SizeBytes,
	}
	if err := s.repo.CreateAttachment(ctx, copied); err != nil {
		return nil, err
	}
	return copied, nil
}

// MarkRead — пометить входящие прочитанными; уведомить заинтересованных.
func (s *Service) MarkRead(ctx context.Context, convID, userID int64) (int, error) {
	conv, err := s.conversationForUser(ctx, convID, userID)
	if err != nil {
		return 0, err
	}
	n, err := s.repo.MarkRead(ctx, conv.ID, userID)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		payload := dto.MessageReadEvent{ConversationID: convID, ReaderID: userID}
		switch {
		case conv.IsDevChat:
			// О прочтении должны знать все, кто видит переписку: владелец и
			// все Администраторы системы. Читатель ВКЛЮЧЁН — его другие
			// устройства по этому событию гасят свой счётчик непрочитанных.
			devIDs, err := s.users.DevChatUserIDs(ctx, conv.UserAID)
			if err != nil {
				return n, err
			}
			if len(devIDs) > 0 {
				s.pub.Publish(ctx, "message:read", rooms(devIDs...), payload)
			}
		default:
			// Собеседнику — галочки «прочитано», самому читателю — синк
			// бейджа на остальных его устройствах.
			otherID := conv.OtherUserID(userID)
			s.pub.Publish(ctx, "message:read", rooms(*otherID, userID), payload)
		}
	}
	return n, nil
}

// UploadAttachment — сохранить файл и зарегистрировать вложение.
func (s *Service) UploadAttachment(ctx context.Context, uploaderID int64,
	fileName, mimeType string, data []byte) (*dto.Attachment, error) {

	if len(data) == 0 {
		return nil, domain.NewError("EMPTY_FILE", "Пустой файл", 400)
	}
	if len(data) > MaxAttachmentSize {
		return nil, domain.NewError("FILE_TOO_LARGE",
			"Файл превышает 25 МБ", 400)
	}
	// Только сам тип, без параметров ("; charset=...") — как
	// werkzeug FileStorage.mimetype.
	mime := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	if mime == "" {
		mime = "application/octet-stream"
	}
	allowed := false
	for _, p := range allowedMimePrefixes {
		if strings.HasPrefix(mime, p) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, domain.NewError("BAD_MIME", "Неподдерживаемый тип файла", 400)
	}

	original := fileName
	if original == "" {
		original = "file"
	}
	ext := truncateString(strings.ToLower(filepath.Ext(original)), 16)
	relPath, err := s.files.Save(data, ext)
	if err != nil {
		return nil, err
	}
	att := &domain.Attachment{
		UploaderID: uploaderID,
		FilePath:   relPath,
		FileName:   truncateString(original, 255),
		MimeType:   truncateString(mime, 120),
		SizeBytes:  int64(len(data)),
	}
	if err := s.repo.CreateAttachment(ctx, att); err != nil {
		return nil, err
	}
	return dto.NewAttachment(att), nil
}

// truncateString — срез по символам, как в Python (не рвёт UTF-8).
func truncateString(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// DeleteMessage — scope=me (скрыть у себя) или all (физически, только своё).
// Возвращает for_all: сообщение исчезло у всех (broadcast message:deleted).
func (s *Service) DeleteMessage(ctx context.Context, messageID, userID int64, scope string) (bool, error) {
	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return false, err
	}
	if msg == nil {
		return false, errMsgNotFound()
	}
	conv, err := s.repo.GetConversation(ctx, msg.ConversationID)
	if err != nil {
		return false, err
	}
	if conv == nil {
		return false, errConvNotFound()
	}
	if err := s.ensureMember(ctx, conv, userID); err != nil {
		return false, err
	}

	var forAll bool
	switch scope {
	case "all":
		if msg.SenderID == nil || *msg.SenderID != userID {
			return false, domain.NewError("FORBIDDEN",
				"Удалить «для всех» можно только своё сообщение", 403)
		}
		if err := s.destroyMessage(ctx, msg); err != nil {
			return false, err
		}
		forAll = true
	case "me":
		both, err := s.repo.HideMessage(ctx, msg.ID, conv.Side(userID))
		if err != nil {
			return false, err
		}
		if both {
			if err := s.destroyMessage(ctx, msg); err != nil {
				return false, err
			}
		}
		forAll = both
	default:
		return false, domain.NewError("BAD_SCOPE", "Неверный scope", 400)
	}

	s.log.Info("message.delete", "message_id", messageID, "user_id", userID, "scope", scope)

	if forAll {
		payload := dto.MessageDeletedEvent{ConversationID: conv.ID, MessageID: messageID}
		var targets []int64
		if otherID := conv.OtherUserID(userID); otherID != nil {
			targets = append(targets, *otherID)
		}
		targets = append(targets, userID)
		s.pub.Publish(ctx, "message:deleted", rooms(targets...), payload)
	}
	return forAll, nil
}

// destroyMessage — физическое удаление сообщения с файлами и пересчётом
// last_message_at (вложения каскадно уходят по FK).
func (s *Service) destroyMessage(ctx context.Context, msg *domain.Message) error {
	paths := make([]string, 0, len(msg.Attachments))
	for _, a := range msg.Attachments {
		paths = append(paths, a.FilePath)
	}
	if err := s.repo.DeleteMessage(ctx, msg.ID); err != nil {
		return err
	}
	if err := s.repo.RecomputeLastMessageAt(ctx, msg.ConversationID); err != nil {
		return err
	}
	s.files.Remove(paths)
	return nil
}

// ToggleMessagePin — общее закрепление сообщения (видят оба участника).
func (s *Service) ToggleMessagePin(ctx context.Context, messageID, userID int64) (*dto.Message, bool, error) {
	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, errMsgNotFound()
	}
	conv, err := s.repo.GetConversation(ctx, msg.ConversationID)
	if err != nil {
		return nil, false, err
	}
	if conv == nil {
		return nil, false, errConvNotFound()
	}
	if err := s.ensureMember(ctx, conv, userID); err != nil {
		return nil, false, err
	}
	// Звонки ведут себя как обычные сообщения (их тоже можно закреплять).
	if msg.Kind != domain.KindText && msg.Kind != domain.KindCall {
		return nil, false, domain.NewError("BAD_PIN", "Это сообщение нельзя закрепить", 400)
	}

	pinned := msg.PinnedAt == nil
	var byID *int64
	if pinned {
		byID = &userID
	}
	if err := s.repo.SetMessagePin(ctx, msg.ID, pinned, byID); err != nil {
		return nil, false, err
	}
	updated, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, false, err
	}
	payload := dto.NewMessage(updated)

	targets := []int64{conv.UserAID}
	if conv.UserBID != nil {
		targets = append(targets, *conv.UserBID)
	}
	s.pub.Publish(ctx, "message:pin", rooms(targets...), dto.MessagePinEvent{
		ConversationID: conv.ID, MessageID: messageID, Pinned: pinned, Message: payload,
	})
	return payload, pinned, nil
}

// ToggleMessageReaction — поставить/снять эмодзи-реакцию на сообщение.
// Событие — message:updated с полным снапшотом: клиенты обновляют сообщение
// тем же обработчиком, что и правку текста.
func (s *Service) ToggleMessageReaction(ctx context.Context, messageID, userID int64, emoji string) (*dto.Message, bool, error) {
	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, errMsgNotFound()
	}
	conv, err := s.repo.GetConversation(ctx, msg.ConversationID)
	if err != nil {
		return nil, false, err
	}
	if conv == nil {
		return nil, false, errConvNotFound()
	}
	if err := s.ensureMember(ctx, conv, userID); err != nil {
		return nil, false, err
	}

	// Лимит: не больше MaxReactionsPerUser разных эмодзи от одного
	// пользователя (снятие уже стоящей — всегда можно).
	mine, hasThis := 0, false
	for _, re := range msg.Reactions {
		if re.UserID == userID {
			mine++
			if re.Emoji == emoji {
				hasThis = true
			}
		}
	}
	if !hasThis && mine >= domain.MaxReactionsPerUser {
		return nil, false, domain.NewError("REACTION_LIMIT",
			"Не больше двух реакций на сообщение", 422)
	}

	added, err := s.repo.ToggleReaction(ctx, messageID, userID, emoji)
	if err != nil {
		return nil, false, err
	}
	updated, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, false, err
	}
	payload := dto.NewMessage(updated)

	targets := []int64{conv.UserAID}
	if conv.UserBID != nil {
		targets = append(targets, *conv.UserBID)
	}
	s.pub.Publish(ctx, "message:updated", rooms(targets...), dto.MessageNewEvent{
		ConversationID: conv.ID, Message: payload, FromUserID: &userID,
	})
	return payload, added, nil
}

// EditMessage — правка текста своего сообщения. Помечается edited_at, что
// клиенты показывают как «изменено». Редактировать можно только своё текстовое
// сообщение (не бота, не плашку звонка/задачи).
func (s *Service) EditMessage(ctx context.Context, messageID, userID int64, text string) (*dto.Message, error) {
	msg, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, errMsgNotFound()
	}
	conv, err := s.repo.GetConversation(ctx, msg.ConversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, errConvNotFound()
	}
	if err := s.ensureMember(ctx, conv, userID); err != nil {
		return nil, err
	}
	if msg.SenderID == nil || *msg.SenderID != userID || msg.IsBot {
		return nil, domain.NewError("FORBIDDEN", "Редактировать можно только своё сообщение", 403)
	}
	if msg.Kind != domain.KindText {
		return nil, domain.NewError("BAD_EDIT", "Это сообщение нельзя редактировать", 400)
	}

	if err := s.repo.UpdateMessageText(ctx, msg.ID, text); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	payload := dto.NewMessage(updated)

	targets := []int64{conv.UserAID}
	if conv.UserBID != nil {
		targets = append(targets, *conv.UserBID)
	}
	s.pub.Publish(ctx, "message:updated", rooms(targets...), dto.MessageNewEvent{
		ConversationID: conv.ID, Message: payload, FromUserID: &userID,
	})
	s.log.Info("message.edit", "message_id", messageID, "user_id", userID)
	return payload, nil
}

// ListPinnedMessages — закреплённые сообщения диалога (для баннера сверху).
func (s *Service) ListPinnedMessages(ctx context.Context, convID, userID int64) ([]*dto.Message, error) {
	conv, err := s.conversationForUser(ctx, convID, userID)
	if err != nil {
		return nil, err
	}
	msgs, err := s.repo.ListPinned(ctx, conv.ID, conv.Side(userID))
	if err != nil {
		return nil, err
	}
	return dto.NewMessages(msgs), nil
}
