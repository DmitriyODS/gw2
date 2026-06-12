// Package service — бизнес-логика звонков поверх LiveKit.
//
// Порт прежнего back/app/services/call_service.py. Медиа (SFU, reconnect,
// mute, data-чат) — целиком на LiveKit. Здесь:
//   - валидация и запись истории в БД (calls / call_participants);
//   - выдача access-токенов LiveKit участникам и внешним гостям;
//   - применение вебхуков LiveKit — источника истины «кто реально в комнате»;
//   - реконсиляция зависших звонков при старте сервиса.
//
// Ринг-фаза инициируется WS-командами call:* через gatewaysvc — он лишь
// транспорт и ходит сюда по gRPC.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/dto"
)

// CallService — публичный контракт сервисного слоя (его оборачивают
// go-kit endpoints; транспорты — gRPC для gateway и Fiber для REST/вебхуков).
type CallService interface {
	StartCall(ctx context.Context, req dto.StartCallRequest) (*dto.StartCallResponse, error)
	InviteToCall(ctx context.Context, req dto.InviteRequest) (*dto.InviteResponse, error)
	AcceptCall(ctx context.Context, req dto.AcceptRequest) (*dto.AcceptResponse, error)
	DeclineCall(ctx context.Context, req dto.HangupRequest) (*dto.HangupResponse, error)
	LeaveCall(ctx context.Context, req dto.HangupRequest) (*dto.HangupResponse, error)
	EndCall(ctx context.Context, req dto.HangupRequest) (*dto.HangupResponse, error)

	RejoinToken(ctx context.Context, callID, userID int64) (*dto.TokenResponse, error)
	ActiveCall(ctx context.Context, userID int64) (*dto.ActiveCallResponse, error)
	History(ctx context.Context, userID int64, limit int) ([]*dto.CallDTO, error)
	JoinInfo(ctx context.Context, code string) (*dto.JoinInfoResponse, error)
	JoinByCode(ctx context.Context, req dto.JoinByCodeRequest) (*dto.JoinByCodeResponse, error)

	HandleWebhook(ctx context.Context, event dto.WebhookEvent) error
	ReconcileStartup(ctx context.Context) error
}

type Service struct {
	repo  domain.CallRepository
	users domain.UserReader
	ring  domain.RingState
	media domain.MediaServer
	pub   domain.EventPublisher
	msgr  domain.MessengerClient
	log   *slog.Logger
}

var _ CallService = (*Service)(nil)

func New(repo domain.CallRepository, users domain.UserReader, ring domain.RingState,
	media domain.MediaServer, pub domain.EventPublisher, msgr domain.MessengerClient,
	log *slog.Logger) *Service {
	return &Service{repo: repo, users: users, ring: ring, media: media, pub: pub,
		msgr: msgr, log: log}
}

func now() time.Time { return time.Now().UTC() }

func shareCode() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err) // crypto/rand не отказывает на живой системе
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func guestIdentity() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return "g-" + hex.EncodeToString(buf)
}

func normalizeMedia(media string) string {
	if media == domain.MediaAudio {
		return domain.MediaAudio
	}
	return domain.MediaVideo
}

// memberToken — LiveKit-токен участника платформы.
func (s *Service) memberToken(call *domain.Call, user *domain.User) (string, error) {
	var avatar any
	if user.AvatarPath != nil {
		avatar = *user.AvatarPath
	}
	return s.media.AccessToken(
		domain.IdentityForUser(user.ID), user.FIO, call.RoomName,
		map[string]any{"user_id": user.ID, "avatar_path": avatar},
	)
}

func (s *Service) livekitDTO(token string) dto.LivekitDTO {
	return dto.LivekitDTO{Token: token, URL: s.media.ClientURL()}
}

// snapshot — собрать CallDTO (call + участники с ФИО из БД).
func (s *Service) snapshot(ctx context.Context, call *domain.Call) (*dto.CallDTO, error) {
	parts, err := s.repo.ListParticipants(ctx, call.ID)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	return dto.NewCallDTO(call, s.initiatorFIO(call, parts), parts), nil
}

func (s *Service) initiatorFIO(call *domain.Call, parts []*domain.Participant) string {
	for _, p := range parts {
		if p.UserID == call.InitiatorID {
			return p.FIO
		}
	}
	return ""
}

// StartCall — создать звонок: запись в БД, комната LiveKit, ринг-state.
// Пустой список приглашённых разрешён: это «пустой звонок» — комната с одним
// инициатором, людей зовут позже (invite из звонка или ссылка-приглашение).
func (s *Service) StartCall(ctx context.Context, req dto.StartCallRequest) (*dto.StartCallResponse, error) {
	inviteeIDs := dedupe(req.InviteeIDs, req.InitiatorID)
	if len(inviteeIDs) > domain.MaxParticipants-1 {
		return nil, domain.NewError("TOO_MANY_INVITEES",
			fmt.Sprintf("Максимум %d участников в одном звонке", domain.MaxParticipants-1), 400)
	}

	if s.ring.IsUserBusy(req.InitiatorID) {
		return nil, domain.NewError("BUSY", "Вы уже в звонке", 409)
	}
	for _, uid := range inviteeIDs {
		if s.ring.IsUserBusy(uid) {
			return nil, domain.NewError("INVITEE_BUSY", "Один из участников уже разговаривает", 409)
		}
	}

	invitees, err := s.users.ListVisibleUsers(ctx, inviteeIDs)
	if err != nil {
		return nil, err
	}
	if len(invitees) != len(inviteeIDs) {
		return nil, domain.NewError("USER_NOT_FOUND", "Один из участников не найден", 404)
	}

	initiator, err := s.users.GetUser(ctx, req.InitiatorID)
	if err != nil {
		return nil, err
	}
	if initiator == nil {
		return nil, domain.NewError("USER_NOT_FOUND", "Пользователь не найден", 404)
	}

	// Multi-tenancy: звонок принадлежит компании инициатора; если инициатор —
	// Администратор системы (без компании) — компании первого приглашённого.
	// Все участники должны быть из одной компании.
	companyID := initiator.CompanyID
	if companyID == nil && len(invitees) > 0 {
		companyID = invitees[0].CompanyID
	}
	if companyID == nil {
		return nil, domain.NewError("NO_COMPANY", "Звонок возможен только в рамках компании", 400)
	}
	for _, u := range invitees {
		if u.CompanyID == nil || *u.CompanyID != *companyID {
			return nil, domain.NewError("CROSS_COMPANY", "Все участники должны быть из одной компании", 422)
		}
	}

	kind := domain.KindGroup
	if len(inviteeIDs) == 1 {
		kind = domain.KindP2P
	}

	// Парный диалог — домен мессенджера: создаём ДО записи звонка, чтобы FK
	// conversation_id уже существовал. msgsvc недоступен — звонок не
	// блокируем: пройдёт без привязки к чату и без плашки (она вторична).
	if kind == domain.KindP2P && req.ConversationID == 0 {
		convID, err := s.msgr.EnsureDialog(ctx, req.InitiatorID, inviteeIDs[0])
		if err != nil {
			s.log.Warn("call.ensure_dialog_failed",
				"initiator_id", req.InitiatorID, "error", err)
		} else {
			req.ConversationID = convID
		}
	}

	ts := now()
	call := &domain.Call{
		InitiatorID: req.InitiatorID,
		CompanyID:   *companyID,
		Kind:        kind,
		Status:      domain.StatusRinging,
		Media:       normalizeMedia(req.Media),
		StartedAt:   ts,
		ShareCode:   shareCode(),
	}
	// Парная привязка к диалогу — только для p2p.
	if kind == domain.KindP2P && req.ConversationID > 0 {
		call.ConversationID = &req.ConversationID
	}

	parts := make([]*domain.Participant, 0, len(inviteeIDs)+1)
	joined := ts
	parts = append(parts, &domain.Participant{
		UserID: req.InitiatorID, Role: domain.RoleInitiator,
		InvitedAt: ts, JoinedAt: &joined,
	})
	for _, uid := range inviteeIDs {
		parts = append(parts, &domain.Participant{
			UserID: uid, Role: domain.RoleInvitee, InvitedAt: ts,
		})
	}
	if err := s.repo.CreateCall(ctx, call, parts); err != nil {
		return nil, fmt.Errorf("create call: %w", err)
	}

	s.ring.CreateCall(call.ID, req.InitiatorID, inviteeIDs, kind, call.Media)

	// Комнату создаём заранее ради лимита участников и empty_timeout; если
	// LiveKit API недоступен — комната автосоздастся при первом подключении.
	s.media.CreateRoom(ctx, call.RoomName, domain.MaxParticipants)

	token, err := s.memberToken(call, initiator)
	if err != nil {
		s.abortStart(ctx, call)
		return nil, fmt.Errorf("member token: %w", err)
	}
	snap, err := s.snapshot(ctx, call)
	if err != nil {
		s.abortStart(ctx, call)
		return nil, err
	}

	// Системная плашка звонка в чате — только p2p; рассылку message:new
	// делает паблишер (fire-and-forget).
	if call.Kind == domain.KindP2P && call.ConversationID != nil {
		s.pub.PillCreated(ctx, *call.ConversationID, req.InitiatorID, call.ID)
	}
	return &dto.StartCallResponse{Call: snap, Livekit: s.livekitDTO(token)}, nil
}

// abortStart — откат не состоявшегося звонка (после записи в БД упала выписка
// токена инициатору): снять ринг-state и стереть запись, иначе участники
// навсегда «заняты», а в истории висит звонок, которого не было.
func (s *Service) abortStart(ctx context.Context, call *domain.Call) {
	s.ring.EndCall(call.ID)
	s.media.DeleteRoom(ctx, call.RoomName)
	if err := s.repo.DeleteCall(ctx, call.ID); err != nil {
		s.log.Error("call.start_abort_failed", "call_id", call.ID, "error", err)
	}
}

// InviteToCall — позвать новых участников в идущий звонок. Любой участник
// может позвать ещё людей.
func (s *Service) InviteToCall(ctx context.Context, req dto.InviteRequest) (*dto.InviteResponse, error) {
	ring, ok := s.ring.Snapshot(req.CallID)
	if !ok || !domain.Has(ring.Invited, req.InviterID) {
		return nil, domain.NewError("NOT_IN_CALL", "Вы не в этом звонке", 404)
	}

	newIDs := make([]int64, 0, len(req.InviteeIDs))
	for _, uid := range dedupe(req.InviteeIDs, req.InviterID) {
		if !domain.Has(ring.Invited, uid) {
			newIDs = append(newIDs, uid)
		}
	}
	call, err := s.repo.GetCall(ctx, req.CallID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return nil, domain.NewError("NOT_IN_CALL", "Звонок не найден", 404)
	}
	if len(newIDs) == 0 {
		snap, err := s.snapshot(ctx, call)
		if err != nil {
			return nil, err
		}
		return &dto.InviteResponse{Call: snap, NewInviteeIDs: []int64{}, NotifyUserIDs: ring.Joined}, nil
	}

	if len(ring.Invited)+len(ring.Guests)+len(newIDs) > domain.MaxParticipants {
		return nil, domain.NewError("TOO_MANY_INVITEES", "В звонке слишком много участников", 400)
	}
	for _, uid := range newIDs {
		if s.ring.IsUserBusy(uid) {
			return nil, domain.NewError("INVITEE_BUSY", "Один из приглашённых уже разговаривает", 409)
		}
	}
	users, err := s.users.ListVisibleUsers(ctx, newIDs)
	if err != nil {
		return nil, err
	}
	if len(users) != len(newIDs) {
		return nil, domain.NewError("USER_NOT_FOUND", "Один из участников не найден", 404)
	}
	for _, u := range users {
		if u.CompanyID == nil || *u.CompanyID != call.CompanyID {
			return nil, domain.NewError("CROSS_COMPANY", "Все участники должны быть из одной компании", 422)
		}
	}

	ts := now()
	for _, uid := range newIDs {
		// CallParticipant мог остаться от прежнего выхода — переиспользуем.
		part, err := s.repo.GetParticipant(ctx, req.CallID, uid)
		if err != nil {
			return nil, err
		}
		if part == nil {
			err = s.repo.CreateParticipant(ctx, &domain.Participant{
				CallID: req.CallID, UserID: uid, Role: domain.RoleInvitee, InvitedAt: ts,
			})
		} else {
			part.InvitedAt = ts
			part.LeftAt = nil
			part.Declined = false
			err = s.repo.UpdateParticipant(ctx, part)
		}
		if err != nil {
			return nil, err
		}
		s.ring.AddInvitee(req.CallID, uid)
	}

	// Звонок на двоих превратился в групповой.
	if call.Kind == domain.KindP2P {
		call.Kind = domain.KindGroup
		s.ring.SetKind(req.CallID, domain.KindGroup)
		if err := s.repo.UpdateCall(ctx, call); err != nil {
			return nil, err
		}
	}

	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	return &dto.InviteResponse{Call: snap, NewInviteeIDs: newIDs, NotifyUserIDs: ring.Joined}, nil
}

// AcceptCall — принять входящий: пометить участие и выдать LiveKit-токен.
func (s *Service) AcceptCall(ctx context.Context, req dto.AcceptRequest) (*dto.AcceptResponse, error) {
	ring, ok := s.ring.Snapshot(req.CallID)
	if !ok || !domain.Has(ring.Invited, req.UserID) {
		return nil, domain.NewError("NOT_INVITED", "Вы не приглашены в этот звонок", 404)
	}

	s.ring.MarkJoined(req.CallID, req.UserID)

	call, err := s.repo.GetCall(ctx, req.CallID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return nil, domain.NewError("NOT_IN_CALL", "Звонок не найден", 404)
	}
	user, err := s.users.GetUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.NewError("USER_NOT_FOUND", "Пользователь не найден", 404)
	}

	if call.Status == domain.StatusRinging {
		call.Status = domain.StatusActive
		if err := s.repo.UpdateCall(ctx, call); err != nil {
			return nil, err
		}
	}
	part, err := s.repo.GetParticipant(ctx, req.CallID, req.UserID)
	if err != nil {
		return nil, err
	}
	if part != nil && part.JoinedAt == nil {
		ts := now()
		part.JoinedAt = &ts
		if err := s.repo.UpdateParticipant(ctx, part); err != nil {
			return nil, err
		}
	}

	token, err := s.memberToken(call, user)
	if err != nil {
		return nil, err
	}
	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	// Плашка в чате: status ringing → active.
	s.pub.PillUpdated(ctx, req.CallID)
	return &dto.AcceptResponse{Call: snap, Livekit: s.livekitDTO(token)}, nil
}

// DeclineCall — явный отказ. Пустой Call в ответе = звонка уже нет (no-op).
func (s *Service) DeclineCall(ctx context.Context, req dto.HangupRequest) (*dto.HangupResponse, error) {
	ring, ok := s.ring.Snapshot(req.CallID)
	if !ok || !domain.Has(ring.Invited, req.UserID) {
		return &dto.HangupResponse{}, nil
	}
	notify := unionExcept(ring.Joined, ring.Invited, req.UserID)

	s.ring.MarkDeclined(req.CallID, req.UserID)

	call, err := s.repo.GetCall(ctx, req.CallID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return &dto.HangupResponse{}, nil
	}
	ts := now()
	part, err := s.repo.GetParticipant(ctx, req.CallID, req.UserID)
	if err != nil {
		return nil, err
	}
	if part != nil {
		part.Declined = true
		part.LeftAt = &ts
		if err := s.repo.UpdateParticipant(ctx, part); err != nil {
			return nil, err
		}
	}

	ended := false
	// Если в p2p отказался единственный приглашённый — это «не дозвонился».
	if call.Kind == domain.KindP2P && call.Status == domain.StatusRinging {
		call.Status = domain.StatusMissed
		call.EndedAt = &ts
		if err := s.repo.UpdateCall(ctx, call); err != nil {
			return nil, err
		}
		s.media.DeleteRoom(ctx, call.RoomName)
		s.ring.EndCall(req.CallID)
		ended = true
	} else if s.ring.ShouldEnd(req.CallID) {
		if err := s.finalize(ctx, call); err != nil {
			return nil, err
		}
		s.ring.EndCall(req.CallID)
		ended = true
	}
	if ended {
		// Завершение должно дойти до всех, кто когда-либо был в звонке.
		notify = s.endedNotifyIDs(ctx, call, ring)
	}

	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	s.pub.PillUpdated(ctx, req.CallID)
	return &dto.HangupResponse{Call: snap, Ended: ended, NotifyUserIDs: notify}, nil
}

// LeaveCall — выход из звонка (повесить трубку).
func (s *Service) LeaveCall(ctx context.Context, req dto.HangupRequest) (*dto.HangupResponse, error) {
	ring, ok := s.ring.Snapshot(req.CallID)
	if !ok {
		return &dto.HangupResponse{}, nil
	}
	notify := unionExcept(ring.Joined, nil, req.UserID)

	s.ring.RemoveUserFromCall(req.CallID, req.UserID)

	call, err := s.repo.GetCall(ctx, req.CallID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return &dto.HangupResponse{}, nil
	}
	part, err := s.repo.GetParticipant(ctx, req.CallID, req.UserID)
	if err != nil {
		return nil, err
	}
	if part != nil && part.LeftAt == nil {
		ts := now()
		part.LeftAt = &ts
		if err := s.repo.UpdateParticipant(ctx, part); err != nil {
			return nil, err
		}
	}

	ended := false
	if s.ring.ShouldEnd(req.CallID) {
		if err := s.finalize(ctx, call); err != nil {
			return nil, err
		}
		s.ring.EndCall(req.CallID)
		ended = true
	}
	if ended {
		// Завершение должно дойти до всех, кто когда-либо был в звонке, а не
		// только до оставшихся в ринг-state: у отвалившегося по сети участника
		// иначе навсегда висит баннер «Вернуться» / live-плашка.
		notify = s.endedNotifyIDs(ctx, call, ring)
	}

	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	s.pub.PillUpdated(ctx, req.CallID)
	return &dto.HangupResponse{Call: snap, Ended: ended, NotifyUserIDs: notify}, nil
}

// EndCall — инициатор завершает звонок целиком (для всех).
func (s *Service) EndCall(ctx context.Context, req dto.HangupRequest) (*dto.HangupResponse, error) {
	ring, ok := s.ring.Snapshot(req.CallID)
	if !ok || ring.InitiatorID != req.UserID {
		return &dto.HangupResponse{}, nil
	}

	call, err := s.repo.GetCall(ctx, req.CallID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return &dto.HangupResponse{}, nil
	}
	notify := s.endedNotifyIDs(ctx, call, ring)
	ts := now()
	for _, uid := range ring.Joined {
		part, err := s.repo.GetParticipant(ctx, req.CallID, uid)
		if err != nil {
			return nil, err
		}
		if part != nil && part.LeftAt == nil {
			part.LeftAt = &ts
			if err := s.repo.UpdateParticipant(ctx, part); err != nil {
				return nil, err
			}
		}
	}
	if err := s.finalize(ctx, call); err != nil {
		return nil, err
	}
	s.ring.EndCall(req.CallID)

	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	s.pub.PillUpdated(ctx, req.CallID)
	return &dto.HangupResponse{Call: snap, Ended: true, NotifyUserIDs: notify}, nil
}

// RejoinToken — токен для возврата в живой звонок (плашка в чате, баннер
// «Вернуться» после F5). Доступен любому участнику звонка.
func (s *Service) RejoinToken(ctx context.Context, callID, userID int64) (*dto.TokenResponse, error) {
	call, err := s.repo.GetCall(ctx, callID)
	if err != nil {
		return nil, err
	}
	if call == nil || call.Finished() {
		return nil, domain.NewError("NOT_IN_CALL", "Звонок уже завершён", 404)
	}

	ring, _ := s.ring.Snapshot(callID)
	if ring == nil || !domain.Has(ring.Invited, userID) {
		// State мог потеряться при рестарте — пускаем по записи в БД.
		part, err := s.repo.GetParticipant(ctx, callID, userID)
		if err != nil {
			return nil, err
		}
		if part == nil {
			return nil, domain.NewError("NOT_INVITED", "Вы не приглашены в этот звонок", 404)
		}
		s.ring.AddInvitee(callID, userID)
	}

	if other, ok := s.ring.UserActiveCall(userID); ok && other != callID {
		return nil, domain.NewError("BUSY", "Вы уже в другом звонке", 409)
	}

	user, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.NewError("USER_NOT_FOUND", "Пользователь не найден", 404)
	}

	s.ring.MarkJoined(callID, userID)
	part, err := s.repo.GetParticipant(ctx, callID, userID)
	if err != nil {
		return nil, err
	}
	if part != nil {
		if part.JoinedAt == nil {
			ts := now()
			part.JoinedAt = &ts
		}
		part.LeftAt = nil
		if err := s.repo.UpdateParticipant(ctx, part); err != nil {
			return nil, err
		}
	}

	token, err := s.memberToken(call, user)
	if err != nil {
		return nil, err
	}
	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	return &dto.TokenResponse{Call: snap, Livekit: s.livekitDTO(token)}, nil
}

// ActiveCall — живой звонок пользователя (восстановление UI после F5).
func (s *Service) ActiveCall(ctx context.Context, userID int64) (*dto.ActiveCallResponse, error) {
	callID, ok := s.ring.UserActiveCall(userID)
	if !ok {
		return &dto.ActiveCallResponse{}, nil
	}
	call, err := s.repo.GetCall(ctx, callID)
	if err != nil {
		return nil, err
	}
	if call == nil || call.Finished() {
		return &dto.ActiveCallResponse{}, nil
	}
	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	return &dto.ActiveCallResponse{Call: snap}, nil
}

// History — звонки, в которых пользователь был участником (новые сверху).
func (s *Service) History(ctx context.Context, userID int64, limit int) ([]*dto.CallDTO, error) {
	calls, err := s.repo.ListHistoryForUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.CallDTO, 0, len(calls))
	for _, c := range calls {
		snap, err := s.snapshot(ctx, c)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, nil
}

// JoinInfo — публичная информация о звонке по ссылке-приглашению.
func (s *Service) JoinInfo(ctx context.Context, code string) (*dto.JoinInfoResponse, error) {
	call, err := s.repo.GetCallByShareCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return nil, domain.NewError("CALL_NOT_FOUND", "Звонок не найден", 404)
	}
	started := dto.FormatTime(call.StartedAt)
	resp := &dto.JoinInfoResponse{
		Status:          call.Status,
		Media:           call.Media,
		Kind:            call.Kind,
		StartedAt:       &started,
		Occupants:       s.occupants(ctx, call),
		MaxParticipants: domain.MaxParticipants,
		Live:            call.Status == domain.StatusRinging || call.Status == domain.StatusActive,
	}
	if initiator, err := s.users.GetUser(ctx, call.InitiatorID); err == nil && initiator != nil {
		resp.InitiatorFIO = &initiator.FIO
	}
	return resp, nil
}

// occupants — сколько людей в комнате: спрашиваем LiveKit (точно), при
// недоступности — по ринг-state.
func (s *Service) occupants(ctx context.Context, call *domain.Call) int {
	if call.RoomName != "" {
		if identities, ok := s.media.ListParticipantIdentities(ctx, call.RoomName); ok {
			return len(identities)
		}
	}
	return s.ring.OccupantsCount(call.ID)
}

// JoinByCode — вход по ссылке-приглашению: пользователь платформы — под
// собой (дозаписывается в участники), внешний гость — под введённым именем.
func (s *Service) JoinByCode(ctx context.Context, req dto.JoinByCodeRequest) (*dto.JoinByCodeResponse, error) {
	call, err := s.repo.GetCallByShareCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if call == nil || call.Finished() {
		return nil, domain.NewError("CALL_NOT_FOUND", "Звонок не найден или уже завершён", 404)
	}
	if s.occupants(ctx, call) >= domain.MaxParticipants {
		return nil, domain.NewError("CALL_FULL", "В звонке нет свободных мест", 409)
	}

	if req.UserID > 0 {
		return s.joinMemberByLink(ctx, call, req.UserID)
	}

	name := strings.TrimSpace(req.GuestName)
	if name == "" {
		name = "Гость"
	}
	if len([]rune(name)) > 64 {
		name = string([]rune(name)[:64])
	}
	identity := guestIdentity()
	token, err := s.media.AccessToken(identity, name, call.RoomName, map[string]any{"guest": true})
	if err != nil {
		return nil, err
	}
	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	return &dto.JoinByCodeResponse{
		Call: snap, Livekit: s.livekitDTO(token), Identity: identity, Guest: true,
	}, nil
}

func (s *Service) joinMemberByLink(ctx context.Context, call *domain.Call, userID int64) (*dto.JoinByCodeResponse, error) {
	if other, ok := s.ring.UserActiveCall(userID); ok && other != call.ID {
		return nil, domain.NewError("BUSY", "Вы уже в другом звонке", 409)
	}
	user, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, domain.NewError("USER_NOT_FOUND", "Пользователь не найден", 404)
	}

	ts := now()
	part, err := s.repo.GetParticipant(ctx, call.ID, userID)
	if err != nil {
		return nil, err
	}
	if part == nil {
		err = s.repo.CreateParticipant(ctx, &domain.Participant{
			CallID: call.ID, UserID: userID, Role: domain.RoleInvitee,
			InvitedAt: ts, JoinedAt: &ts,
		})
	} else {
		if part.JoinedAt == nil {
			part.JoinedAt = &ts
		}
		part.LeftAt = nil
		part.Declined = false
		err = s.repo.UpdateParticipant(ctx, part)
	}
	if err != nil {
		return nil, err
	}

	// Вошедший по ссылке третий участник делает звонок групповым.
	if ring, ok := s.ring.Snapshot(call.ID); ok && !domain.Has(ring.Invited, userID) {
		s.ring.AddInvitee(call.ID, userID)
		if updated, ok := s.ring.Snapshot(call.ID); ok &&
			call.Kind == domain.KindP2P && len(updated.Invited) > 2 {
			call.Kind = domain.KindGroup
			s.ring.SetKind(call.ID, domain.KindGroup)
			if err := s.repo.UpdateCall(ctx, call); err != nil {
				return nil, err
			}
		}
	}
	s.ring.MarkJoined(call.ID, userID)

	token, err := s.memberToken(call, user)
	if err != nil {
		return nil, err
	}
	snap, err := s.snapshot(ctx, call)
	if err != nil {
		return nil, err
	}
	return &dto.JoinByCodeResponse{
		Call: snap, Livekit: s.livekitDTO(token),
		Identity: domain.IdentityForUser(userID), Guest: false,
	}, nil
}

// endedNotifyIDs — кому слать call_ended: ВСЕ участники звонка из БД (включая
// вышедших ранее — у них мог остаться баннер «Вернуться» или live-плашка на
// другой вкладке), объединённые с ринг-state на случай рассинхрона записи.
// Списки из ring.Joined для этого не годятся: отвалившийся по сети участник
// уже удалён из ринг-state и никогда не узнал бы о завершении.
func (s *Service) endedNotifyIDs(ctx context.Context, call *domain.Call, ring *domain.RingSnapshot) []int64 {
	ids := []int64{call.InitiatorID}
	if ring != nil {
		ids = unionExcept(ids, ring.Invited, 0)
	}
	parts, err := s.repo.ListParticipants(ctx, call.ID)
	if err != nil {
		s.log.Error("calls.ended_notify_failed", "call_id", call.ID, "error", err)
		return ids
	}
	dbIDs := make([]int64, 0, len(parts))
	for _, p := range parts {
		dbIDs = append(dbIDs, p.UserID)
	}
	return unionExcept(ids, dbIDs, 0)
}

// finalize — закрыть звонок в БД и погасить комнату LiveKit (выкидывает
// оставшихся, включая гостей).
func (s *Service) finalize(ctx context.Context, call *domain.Call) error {
	if !call.Finished() {
		call.Status = domain.StatusEnded
	}
	if call.EndedAt == nil {
		ts := now()
		call.EndedAt = &ts
	}
	if err := s.repo.UpdateCall(ctx, call); err != nil {
		return err
	}
	if call.RoomName != "" {
		s.media.DeleteRoom(ctx, call.RoomName)
	}
	return nil
}

// dedupe — уникальные id без excluded.
func dedupe(ids []int64, excluded int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id == excluded || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// unionExcept — объединение двух списков без excluded (0 = никого не исключать).
func unionExcept(a, b []int64, excluded int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	out := make([]int64, 0, len(a)+len(b))
	for _, list := range [][]int64{a, b} {
		for _, id := range list {
			if id == excluded {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
