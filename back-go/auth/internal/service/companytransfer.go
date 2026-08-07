package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/companydata"
)

/* Перенос компании: администратор выгружает свою компанию одним файлом и
   поднимает её обратно — на другом сервере, в другом аккаунте или рядом,
   копией.

   Архив собирает authsvc, но СОДЕРЖИМОЕ разделов ему непрозрачно: каждый
   владелец отдаёт свой JSON (auth.v1.CompanyDataService), а authsvc добавляет
   к ним описание компании, справочник участников и сами файлы.

   Люди и геймификация не переносятся: аккаунты остаются на своей платформе, а
   ссылки на них восстанавливаются сопоставлением по логину. Не нашли человека —
   ссылка достаётся тому, кто импортирует: запись без автора выглядела бы битой.

   Импорт всегда создаёт НОВУЮ компанию. Вливать в существующую нельзя
   намеренно: разрешать конфликты имён отделов, этапов и меток пришлось бы
   вслепую, а откатить такое слияние человек уже не сможет. */

const (
	transferVersion  = 1
	transferMeta     = "company.json"
	transferDataDir  = "data/"
	transferFilesDir = "files/"
	// Предел разжатого архива: защита от zip-бомбы на импорте.
	transferMaxBytes = 2 << 30
)

var (
	errTransferUnavailable = domain.NewError("TRANSFER_UNAVAILABLE",
		"Перенос компаний сейчас недоступен: разделы не отвечают", 503)
	errBadArchive = domain.NewError("BAD_ARCHIVE",
		"Файл не похож на выгрузку компании", 400)
)

// CompanyTransferSections — разделы переноса в порядке вливания.
var CompanyTransferSections = []string{"tasks", "registry", "calendar", "portal"}

// TransferMeta — сопроводительная часть архива.
type TransferMeta struct {
	Version     int              `json:"version"`
	ExportedAt  time.Time        `json:"exported_at"`
	Company     TransferCompany  `json:"company"`
	Members     []TransferMember `json:"members"`
	Sections    map[string]int   `json:"sections"`
	FileCount   int              `json:"file_count"`
	SourceAppID string           `json:"source_app,omitempty"`
	Settings    map[string]any   `json:"-"`
}

type TransferCompany struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
}

// TransferMember — участник исходной компании. Пароли и контакты сюда не
// входят: это карта для сопоставления, а не выгрузка людей.
type TransferMember struct {
	UserID int64  `json:"user_id"`
	Login  string `json:"login"`
	FIO    string `json:"fio"`
}

// ExportCompany — архив компании; выгружает её создатель или супер-админ.
func (s *Service) ExportCompany(ctx context.Context, actor *domain.User, companyID int64) ([]byte, string, error) {
	if err := s.creatorAuthority(ctx, actor, companyID); err != nil {
		return nil, "", err
	}
	company, err := s.companies.GetCompany(ctx, companyID)
	if err != nil {
		return nil, "", err
	}
	if company == nil {
		return nil, "", errCompanyNotFound
	}
	if s.transfer == nil {
		return nil, "", errTransferUnavailable
	}

	meta := TransferMeta{
		Version:    transferVersion,
		ExportedAt: time.Now().UTC(),
		Company: TransferCompany{
			Name:        company.Name,
			Description: company.Description,
			Settings:    company.Settings,
		},
		Sections: map[string]int{},
	}

	members, err := s.repo.SearchDirectoryMembers(ctx, "", 0, companyID)
	if err != nil {
		return nil, "", err
	}
	for _, m := range members {
		meta.Members = append(meta.Members, TransferMember{UserID: m.ID, Login: m.Login, FIO: m.FIO})
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	fileKeys := map[string]bool{}
	for _, section := range CompanyTransferSections {
		if !s.transfer.Has(section) {
			continue
		}
		res, err := s.transfer.Export(ctx, section, companyID)
		if err != nil {
			return nil, "", fmt.Errorf("раздел %s: %w", section, err)
		}
		if err := writeZip(zw, transferDataDir+section+".json", res.Payload); err != nil {
			return nil, "", err
		}
		meta.Sections[section] = res.Count
		for _, key := range res.FileKeys {
			if key != "" && zipslipSafe(key) {
				fileKeys[key] = true
			}
		}
	}

	if s.files != nil {
		keys := make([]string, 0, len(fileKeys))
		for key := range fileKeys {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			rc, err := s.files.Open(ctx, key)
			if err != nil {
				// Пропавший объект не должен рушить выгрузку: ссылка на него
				// уже мертва, и архив без него честнее, чем отказ целиком.
				s.log.Warn("company_transfer.file_missing", "key", key, "error", err)
				continue
			}
			w, err := zw.CreateHeader(&zip.FileHeader{Name: transferFilesDir + key, Method: zip.Deflate})
			if err != nil {
				rc.Close() //nolint:errcheck
				return nil, "", err
			}
			if _, err := io.Copy(w, rc); err != nil {
				rc.Close() //nolint:errcheck
				return nil, "", err
			}
			rc.Close() //nolint:errcheck
			meta.FileCount++
		}
	}

	rawMeta, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, "", err
	}
	if err := writeZip(zw, transferMeta, rawMeta); err != nil {
		return nil, "", err
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}

	s.log.Info("company_transfer.export", "company_id", companyID,
		"sections", len(meta.Sections), "files", meta.FileCount)
	return buf.Bytes(), transferFileName(company.Name), nil
}

// ImportCompany — поднять компанию из архива НОВОЙ компанией; актор становится
// её создателем и администратором. name — своё имя вместо имени из архива.
func (s *Service) ImportCompany(ctx context.Context, actor *domain.User, zipBytes []byte, name string) (*TransferResult, error) {
	if s.transfer == nil {
		return nil, errTransferUnavailable
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, errBadArchive
	}

	var meta TransferMeta
	payloads := map[string][]byte{}
	fileEntries := map[string]*zip.File{}
	var total int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		total += int64(f.UncompressedSize64)
		if total > transferMaxBytes {
			return nil, errBadArchive
		}
		switch {
		case f.Name == transferMeta:
			raw, err := readZip(f)
			if err != nil {
				return nil, errBadArchive
			}
			if err := json.Unmarshal(raw, &meta); err != nil {
				return nil, errBadArchive
			}
		case strings.HasPrefix(f.Name, transferDataDir):
			raw, err := readZip(f)
			if err != nil {
				return nil, errBadArchive
			}
			payloads[strings.TrimSuffix(strings.TrimPrefix(f.Name, transferDataDir), ".json")] = raw
		case strings.HasPrefix(f.Name, transferFilesDir):
			key := strings.TrimPrefix(f.Name, transferFilesDir)
			if zipslipSafe(key) {
				fileEntries[key] = f
			}
		}
	}
	if meta.Version == 0 || meta.Company.Name == "" {
		return nil, errBadArchive
	}

	companyName := strings.TrimSpace(name)
	if companyName == "" {
		companyName = meta.Company.Name
	}
	companyName = s.freeCompanyName(ctx, companyName)

	if err := s.ensureCompanyLimit(ctx, actor.ID); err != nil {
		return nil, err
	}
	company := &domain.Company{
		Name:        companyName,
		Description: meta.Company.Description,
		CreatedBy:   &actor.ID,
		Settings:    meta.Company.Settings,
	}
	if err := s.companies.CreateCompany(ctx, company); err != nil {
		return nil, err
	}
	if err := s.ensureAdminMembership(ctx, actor.ID, company.ID); err != nil {
		return nil, err
	}

	users := s.matchMembers(ctx, meta.Members)

	// Файлы кладём под НОВЫМИ ключами: старый ключ занят исходной компанией, а
	// при импорте рядом две компании ссылались бы на один объект — удаление в
	// одной убивало бы картинку в другой.
	files := map[string]string{}
	if s.files != nil {
		for key, f := range fileEntries {
			data, err := readZip(f)
			if err != nil {
				continue
			}
			newKey := reKey(key)
			if err := s.files.Put(ctx, newKey, data, ""); err != nil {
				s.log.Warn("company_transfer.file_put_failed", "key", newKey, "error", err)
				continue
			}
			files[key] = newKey
		}
	}

	result := &TransferResult{CompanyID: company.ID, CompanyName: company.Name, Sections: map[string]int{}, Files: len(files)}
	for _, section := range CompanyTransferSections {
		payload, ok := payloads[section]
		if !ok || !s.transfer.Has(section) {
			continue
		}
		count, err := s.transfer.Import(ctx, section, companydata.Import{
			CompanyID: company.ID,
			ActorID:   actor.ID,
			Payload:   payload,
			Users:     users,
			Files:     files,
		})
		if err != nil {
			// Компания уже создана: сносим её вместе с тем, что успело влиться,
			// иначе у человека останется половина переноса.
			if delErr := s.companies.DeleteCompany(ctx, company.ID); delErr != nil {
				s.log.Error("company_transfer.rollback_failed", "company_id", company.ID, "error", delErr)
			}
			return nil, fmt.Errorf("раздел %s: %w", section, err)
		}
		result.Sections[section] = count
	}

	s.log.Info("company_transfer.import", "company_id", company.ID, "actor_id", actor.ID,
		"sections", len(result.Sections), "files", result.Files)
	return result, nil
}

// TransferResult — что получилось из архива.
type TransferResult struct {
	CompanyID   int64          `json:"company_id"`
	CompanyName string         `json:"company_name"`
	Sections    map[string]int `json:"sections"`
	Files       int            `json:"files"`
}

// matchMembers — исходный user_id → идентификатор в этой системе. Сопоставляем
// по логину: он уникален и переживает переименование человека.
func (s *Service) matchMembers(ctx context.Context, members []TransferMember) map[int64]int64 {
	out := make(map[int64]int64, len(members))
	for _, m := range members {
		if m.Login == "" {
			continue
		}
		user, err := s.repo.GetByLogin(ctx, m.Login)
		if err != nil || user == nil {
			continue
		}
		out[m.UserID] = user.ID
	}
	return out
}

// freeCompanyName — «Имя», «Имя (2)», … : имя компании уникально, а импорт
// рядом с исходной компанией — обычный сценарий (проверка копии перед сносом).
func (s *Service) freeCompanyName(ctx context.Context, base string) string {
	name := base
	for i := 2; i < 100; i++ {
		existing, err := s.companies.GetCompanyByName(ctx, name)
		if err != nil || existing == nil {
			return name
		}
		name = fmt.Sprintf("%s (%d)", base, i)
	}
	return name
}

// reKey — новый ключ объекта в той же папке сервиса: раздел-владелец узнаёт
// свои файлы по префиксу, поэтому папку сохраняем, а имя делаем случайным.
func reKey(key string) string {
	dir := path.Dir(key)
	ext := path.Ext(key)
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return key
	}
	name := hex.EncodeToString(buf) + ext
	if dir == "." || dir == "/" {
		return name
	}
	return dir + "/" + name
}

func writeZip(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readZip(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close() //nolint:errcheck
	return io.ReadAll(io.LimitReader(rc, transferMaxBytes))
}

// transferFileName — имя файла выгрузки: латиница и цифры исходного названия,
// иначе браузеры и файловые менеджеры получают неудобное имя.
func transferFileName(company string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(company) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "company"
	}
	return fmt.Sprintf("%s-%s.gwcompany", slug, time.Now().Format("2006-01-02"))
}
