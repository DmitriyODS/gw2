package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// filesPrefix — каталог архива, куда складываются ВСЕ загруженные файлы под их
// storage-ключами (avatars/…, registry/…, calendar/…, notes/…, portal/…,
// вложения мессенджера). «Выгрузка всего»: полный бэкап медиа вместе с БД.
const filesPrefix = "files/"

// Универсальный бэкап: ZIP с data.json (карта «таблица → JSON-массив строк») и
// каталогом avatars/. Состав определяется выбранными разделами; список таблиц
// раздела обнаруживается из БД, поэтому новые таблицы не теряются.

// resolveSections — по выбранным разделам возвращает список существующих в БД
// таблиц (с учётом псевдо-раздела SectionOther) и фактически использованные
// ключи разделов. Пустой sections → все разделы.
func (s *Service) resolveSections(ctx context.Context, sections []string) (tables, used []string, err error) {
	all, err := s.backup.AllTables(ctx)
	if err != nil {
		return nil, nil, err
	}
	allSet := make(map[string]bool, len(all))
	for _, t := range all {
		allSet[t] = true
	}

	sec2tables := map[string][]string{}
	known := map[string]bool{}
	for _, sec := range domain.BackupSections {
		sec2tables[sec.Key] = sec.Tables
		for _, t := range sec.Tables {
			known[t] = true
		}
	}
	other := []string{}
	for _, t := range all {
		if !known[t] {
			other = append(other, t)
		}
	}
	sec2tables[domain.SectionOther] = other

	if len(sections) == 0 {
		for _, sec := range domain.BackupSections {
			sections = append(sections, sec.Key)
		}
		sections = append(sections, domain.SectionOther)
	}

	seen := map[string]bool{}
	for _, key := range sections {
		for _, t := range sec2tables[key] {
			if allSet[t] && !seen[t] {
				seen[t] = true
				tables = append(tables, t)
			}
		}
	}
	return tables, sections, nil
}

func hasSection(sections []string, key string) bool {
	for _, s := range sections {
		if s == key {
			return true
		}
	}
	return false
}

func (s *Service) ExportBackup(ctx context.Context, sections []string) ([]byte, error) {
	tables, used, err := s.resolveSections(ctx, sections)
	if err != nil {
		return nil, err
	}
	data, err := s.backup.ExportTables(ctx, tables)
	if err != nil {
		return nil, err
	}

	archive := domain.BackupArchive{Version: domain.BackupVersion, Sections: used, Tables: data}

	var jsonBuf bytes.Buffer
	enc := json.NewEncoder(&jsonBuf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(archive); err != nil {
		return nil, err
	}
	raw := bytes.TrimRight(jsonBuf.Bytes(), "\n")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "data.json", Method: zip.Deflate})
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(raw); err != nil {
		return nil, err
	}

	// Все загруженные файлы (медиа мессенджера, реестров/календарей/заметок/
	// портала, аватарки) — под files/<ключ>. «Выгрузка всего»: полный медиа-
	// архив вместе с БД. Файлы кладём как есть, независимо от разделов.
	fileCount := 0
	if s.files != nil {
		keys, err := s.files.List(ctx, "")
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			rc, err := s.files.Open(ctx, key)
			if err != nil {
				return nil, err
			}
			fw, err := zw.CreateHeader(&zip.FileHeader{Name: filesPrefix + key, Method: zip.Deflate})
			if err != nil {
				rc.Close() //nolint:errcheck
				return nil, err
			}
			if _, err := io.Copy(fw, rc); err != nil {
				rc.Close() //nolint:errcheck
				return nil, err
			}
			rc.Close() //nolint:errcheck
			fileCount++
		}
	} else if hasSection(used, "auth") {
		// Фолбэк без корневого хранилища — как раньше, только аватарки.
		avatars, err := s.avatars.ListFiles()
		if err != nil {
			return nil, err
		}
		for _, f := range avatars {
			w, err := zw.CreateHeader(&zip.FileHeader{Name: "avatars/" + f.Name, Method: zip.Deflate})
			if err != nil {
				return nil, err
			}
			if _, err := w.Write(f.Data); err != nil {
				return nil, err
			}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	s.log.Info("backup.export", "sections", used, "tables", len(tables), "files", fileCount)
	return buf.Bytes(), nil
}

// zipslipSafe — ключ не выходит за пределы хранилища (без ../ и ведущего /).
func zipslipSafe(key string) bool {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return false
	}
	return true
}

// ImportBackup — ДЕСТРУКТИВНОЕ восстановление выбранных разделов из архива.
func (s *Service) ImportBackup(ctx context.Context, zipBytes []byte, sections []string) error {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return err
	}

	var rawData []byte
	for _, f := range zr.File {
		if f.Name != "data.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(rc)
		rc.Close() //nolint:errcheck
		if err != nil {
			return err
		}
		rawData = buf.Bytes()
		break
	}
	if rawData == nil {
		return domain.NewError("IMPORT_ERROR", "data.json не найден в архиве", 400)
	}

	archive, err := parseArchive(rawData)
	if err != nil {
		return err
	}

	// Таблицы к восстановлению: выбранные разделы ∩ присутствующие в архиве.
	wantTables, used, err := s.resolveSections(ctx, sections)
	if err != nil {
		return err
	}
	restore := wantTables[:0]
	for _, t := range wantTables {
		if _, ok := archive.Tables[t]; ok {
			restore = append(restore, t)
		}
	}

	// Файлы восстанавливаем из архива: новый формат — files/<ключ> в корневое
	// хранилище (все медиа), совместимость со старыми архивами — avatars/<имя>.
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		switch {
		case strings.HasPrefix(f.Name, filesPrefix) && len(f.Name) > len(filesPrefix):
			if s.files == nil {
				continue
			}
			key := f.Name[len(filesPrefix):]
			if !zipslipSafe(key) {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			data, err := io.ReadAll(rc)
			rc.Close() //nolint:errcheck
			if err != nil {
				return err
			}
			if err := s.files.Put(ctx, key, data, ""); err != nil {
				return err
			}
		case strings.HasPrefix(f.Name, "avatars/") && len(f.Name) > len("avatars/") && hasSection(used, "auth"):
			rc, err := f.Open()
			if err != nil {
				return err
			}
			data, err := io.ReadAll(rc)
			rc.Close() //nolint:errcheck
			if err != nil {
				return err
			}
			if err := s.avatars.WriteFile(f.Name, data); err != nil {
				return err
			}
		}
	}

	if err := s.backup.ImportTables(ctx, restore, archive.Tables); err != nil {
		return err
	}
	s.log.Info("backup.import", "sections", used, "tables", len(restore))
	return nil
}

// parseArchive — разбор data.json. Версия 2 — объект {version, sections, tables}.
// Старые архивы (таблицы на верхнем уровне) поддерживаются опционально: ключи со
// значением-массивом трактуются как таблицы.
func parseArchive(raw []byte) (*domain.BackupArchive, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if _, ok := probe["tables"]; ok {
		var a domain.BackupArchive
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, err
		}
		if a.Tables == nil {
			a.Tables = map[string]json.RawMessage{}
		}
		return &a, nil
	}
	// Старый формат.
	a := &domain.BackupArchive{Version: 1, Tables: map[string]json.RawMessage{}}
	for k, v := range probe {
		if strings.HasPrefix(strings.TrimSpace(string(v)), "[") {
			a.Tables[k] = v
		}
	}
	return a, nil
}
