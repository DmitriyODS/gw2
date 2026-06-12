package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// Портировано из back/app/services/backup_service.py: ZIP с data.json
// (JSON c indent=2, без экранирования не-ASCII — как json.dumps(...,
// ensure_ascii=False, indent=2)) + каталог avatars/.

func (s *Service) ExportBackup(ctx context.Context) ([]byte, error) {
	data, err := s.backup.ExportData(ctx)
	if err != nil {
		return nil, err
	}

	var jsonBuf bytes.Buffer
	enc := json.NewEncoder(&jsonBuf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	// Encoder добавляет завершающий \n — json.dumps его не пишет.
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
	if err := zw.Close(); err != nil {
		return nil, err
	}

	s.log.Info("backup.export")
	return buf.Bytes(), nil
}

// ImportBackup — ДЕСТРУКТИВНАЯ операция: полная замена данных из архива.
// Любая ошибка наружу — транспорт отвечает 400 IMPORT_ERROR (как Flask).
func (s *Service) ImportBackup(ctx context.Context, zipBytes []byte) error {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return err
	}

	var data *domain.BackupData
	for _, f := range zr.File {
		if f.Name != "data.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = json.NewDecoder(rc).Decode(&data)
		rc.Close() //nolint:errcheck
		if err != nil {
			return err
		}
		break
	}
	if data == nil {
		return domain.NewError("IMPORT_ERROR", "data.json не найден в архиве", 400)
	}

	// Аватарки восстанавливаем до данных — как во Flask (файлы пишутся при
	// чтении архива, ошибки записи валят импорт целиком).
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || len(f.Name) <= len("avatars/") ||
			f.Name[:len("avatars/")] != "avatars/" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		content := new(bytes.Buffer)
		_, err = content.ReadFrom(rc)
		rc.Close() //nolint:errcheck
		if err != nil {
			return err
		}
		if err := s.avatars.WriteFile(f.Name, content.Bytes()); err != nil {
			return err
		}
	}

	if err := s.backup.ImportData(ctx, data); err != nil {
		return err
	}
	s.log.Info("backup.import")
	return nil
}
