// Package service — бизнес-логика диска поверх портов домена.
package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

type Deps struct {
	Repo  domain.Repository
	Users domain.UserReader
	Files domain.FileStore
	Bus   domain.EventBus
	Log   *slog.Logger
	// TempDir — где копятся куски больших файлов (пусто — системный temp).
}

type Service struct {
	repo  domain.Repository
	users domain.UserReader
	files domain.FileStore
	bus   domain.EventBus
	log   *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, users: d.Users, files: d.Files, bus: d.Bus, log: d.Log}
}

// rooms — события диска адресуются ПОИМЁННО: владельцу и тем, кому он открыл
// доступ. Комната all тут не годится — содержимое личное.
func rooms(userIDs ...int64) []string {
	out := make([]string, 0, len(userIDs))
	seen := map[int64]bool{}
	for _, id := range userIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, "user_"+strconv.FormatInt(id, 10))
	}
	return out
}

// companiesOf — компании пользователя (доступ, выданный компании). Ошибку не
// поднимаем: без списка человек просто увидит только то, что открыто лично.
func (s *Service) companiesOf(ctx context.Context, userID int64) []int64 {
	ids, err := s.users.UserCompanies(ctx, userID)
	if err != nil {
		s.log.Warn("drive.companies_failed", "user_id", userID, "error", err)
		return nil
	}
	return ids
}

// RunTrashCleaner — фоновая чистка корзины: то, что пролежало дольше
// TrashKeepDays, удаляется вместе с объектами в хранилище.
func (s *Service) RunTrashCleaner(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		s.cleanTrash(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) cleanTrash(ctx context.Context) {
	files, folders, err := s.repo.ExpiredTrash(ctx)
	if err != nil {
		s.log.Warn("drive.trash_scan_failed", "error", err)
		return
	}
	if len(files) == 0 && len(folders) == 0 {
		return
	}
	byOwner := map[int64][]string{}
	ids := make([]int64, 0, len(files))
	for _, f := range files {
		byOwner[f.OwnerID] = append(byOwner[f.OwnerID], f.Key)
		ids = append(ids, f.ID)
	}
	if err := s.repo.DeleteFiles(ctx, ids); err != nil {
		s.log.Warn("drive.trash_delete_failed", "error", err)
		return
	}
	if err := s.repo.DeleteFolders(ctx, folders); err != nil {
		s.log.Warn("drive.trash_delete_folders_failed", "error", err)
	}
	for ownerID, keys := range byOwner {
		s.files.RemoveFor(ctx, ownerID, 0, keys)
	}
	s.log.Info("drive.trash_cleaned", "files", len(ids), "folders", len(folders))
}
