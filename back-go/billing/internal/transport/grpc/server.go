// Package grpc — контракт биллинга для остальных микросервисов: лимиты
// тарифа, проверка хранилища и токены доступа к ИИ. Ошибки инфраструктуры
// уезжают обычным gRPC-статусом: вызывающая сторона держит fail-open политику
// (недоступный биллинг не должен останавливать работу платформы).
package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/billing/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/billingpb"
)

type Server struct {
	billingpb.UnimplementedBillingServiceServer
	svc *service.Service
}

func NewServer(svc *service.Service) *Server { return &Server{svc: svc} }

func pbLimits(l domain.Limits) *billingpb.Limits {
	return &billingpb.Limits{
		Tasks:            l.Tasks,
		Companies:        int32(l.Companies),
		Members:          int32(l.Members),
		StorageBytes:     l.StorageBytes,
		AiTokens:         l.AITokens,
		Calendars:        int32(l.Calendars),
		Diaries:          int32(l.Diaries),
		Boards:           int32(l.Boards),
		Registries:       int32(l.Registries),
		ChatFolders:      int32(l.ChatFolders),
		CallParticipants: int32(l.CallParticipants),
		DataTransfer:     l.DataTransfer,
		AdvancedStats:    l.AdvancedStats,
		Portal:           l.Portal,
		UserStatuses:     l.UserStatuses,
		PremiumThemes:    l.PremiumThemes,
		PremiumPetSkins:  l.PremiumPetSkins,
		PremiumPetHouse:  l.PremiumPetHouse,
		PremiumPetGoods:  l.PremiumPetGoods,
	}
}

func (s *Server) GetEntitlements(ctx context.Context, in *billingpb.GetEntitlementsRequest) (*billingpb.GetEntitlementsResponse, error) {
	ent, err := s.svc.Entitlements(ctx, in.GetUserId(), in.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := &billingpb.GetEntitlementsResponse{
		Plan:        ent.Plan,
		PlanName:    ent.PlanName,
		Limits:      pbLimits(ent.Limits),
		StorageUsed: ent.StorageUsed,
		TokensUsed:  ent.TokensUsed,
		TokensLeft:  ent.TokensLeft,
		OwnerId:     ent.OwnerID,
	}
	if ent.ExpiresAt != nil {
		out.ExpiresAt = ent.ExpiresAt.Unix()
	}
	return out, nil
}

func (s *Server) CheckStorage(ctx context.Context, in *billingpb.CheckStorageRequest) (*billingpb.CheckStorageResponse, error) {
	allowed, free, ent, err := s.svc.CheckStorage(ctx, in.GetUserId(), in.GetCompanyId(), in.GetBytes())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &billingpb.CheckStorageResponse{
		Allowed:    allowed,
		FreeBytes:  free,
		LimitBytes: ent.Limits.StorageBytes,
		Plan:       ent.Plan,
		OwnerId:    ent.OwnerID,
	}, nil
}

func (s *Server) TrackStorage(ctx context.Context, in *billingpb.TrackStorageRequest) (*billingpb.TrackStorageResponse, error) {
	total, err := s.svc.TrackStorage(ctx, in.GetUserId(), in.GetCompanyId(), in.GetService(), in.GetDeltaBytes())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &billingpb.TrackStorageResponse{TotalBytes: total}, nil
}

func (s *Server) CheckAI(ctx context.Context, in *billingpb.CheckAIRequest) (*billingpb.CheckAIResponse, error) {
	payer, left, err := s.svc.CheckAI(ctx, in.GetUserId(), in.GetCompanyId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &billingpb.CheckAIResponse{Allowed: left > 0, PayerId: payer, TokensLeft: left}, nil
}

func (s *Server) ConsumeAI(ctx context.Context, in *billingpb.ConsumeAIRequest) (*billingpb.ConsumeAIResponse, error) {
	rec := domain.AIUsageRecord{
		UserID:           in.GetPayerId(),
		Feature:          in.GetFeature(),
		Model:            in.GetModel(),
		PromptTokens:     int(in.GetPromptTokens()),
		CompletionTokens: int(in.GetCompletionTokens()),
		BilledTokens:     in.GetBilledTokens(),
		OwnKey:           in.GetOwnKey(),
	}
	if id := in.GetActorId(); id > 0 {
		rec.ActorID = &id
	}
	if id := in.GetCompanyId(); id > 0 {
		rec.CompanyID = &id
	}
	ok, left, err := s.svc.ConsumeAI(ctx, rec)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &billingpb.ConsumeAIResponse{Ok: ok, TokensLeft: left}, nil
}

func (s *Server) LogAction(ctx context.Context, in *billingpb.LogActionRequest) (*billingpb.LogActionResponse, error) {
	entry := &domain.AuditEntry{
		Action: in.GetAction(), TargetKind: in.GetTargetKind(),
		TargetID: in.GetTargetId(), Summary: in.GetSummary(),
	}
	if id := in.GetActorId(); id > 0 {
		entry.ActorID = &id
	}
	if err := s.svc.LogExternalAction(ctx, entry); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &billingpb.LogActionResponse{Ok: true}, nil
}
