package service

import (
	"archive/zip"
	"bytes"
	"context"
	"path"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/docx"
	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// Форматы экспорта заметки.
const (
	FormatTXT  = "txt"
	FormatDOCX = "docx"
)

// ExportFile — выгруженный файл: содержимое + имя (без расширения) + расширение.
type ExportFile struct {
	Data []byte
	Name string
	Ext  string
}

// noteText — «заголовок + пустая строка + текст» заметки.
func noteText(n *domain.Note) string {
	title := strings.TrimSpace(n.Title)
	content := title
	if n.TextContent != "" {
		if content != "" {
			content += "\n\n"
		}
		content += n.TextContent
	}
	return content
}

// Export — заметка в txt или docx. Доступен и адресатам шаринга (чтение есть —
// выгрузка тоже).
func (s *Service) Export(ctx context.Context, userID, id int64, format string) (*ExportFile, error) {
	n, _, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(n.Title)
	if name == "" {
		name = "Заметка"
	}
	if format == FormatDOCX {
		data, err := docx.Build(n.Title, n.TextContent)
		if err != nil {
			return nil, err
		}
		return &ExportFile{Data: data, Name: name, Ext: "docx"}, nil
	}
	return &ExportFile{Data: []byte(noteText(n)), Name: name, Ext: "txt"}, nil
}

// ExportFolder — zip со всем поддеревом папки: подпапки как каталоги, заметки —
// файлами (txt или docx). Только владелец.
func (s *Service) ExportFolder(ctx context.Context, userID, id int64, format string) (*ExportFile, error) {
	f, err := s.requireFolderOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	ext := "txt"
	if format == FormatDOCX {
		ext = "docx"
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	count := 0
	if err := s.zipFolder(ctx, zw, userID, id, "", ext, &count); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, domain.ErrNothingToExport
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(f.Name)
	if name == "" {
		name = "Папка"
	}
	return &ExportFile{Data: buf.Bytes(), Name: name, Ext: "zip"}, nil
}

// uniqueName — уникальное имя в рамках набора used (дедуп в одном каталоге).
func uniqueName(used map[string]int, base, suffix string) string {
	name := base + suffix
	for used[strings.ToLower(name)] > 0 {
		used[strings.ToLower(base+suffix)]++
		name = base + " (" + strconv.Itoa(used[strings.ToLower(base+suffix)]) + ")" + suffix
	}
	used[strings.ToLower(name)] = 1
	return name
}

// writeNoteFile — записать заметку файлом в архив (по её id, с полным текстом).
// Возвращает true, если файл действительно записан (заметка существует).
func (s *Service) writeNoteFile(ctx context.Context, zw *zip.Writer, noteID int64, prefix, ext string, used map[string]int) (bool, error) {
	n, err := s.repo.GetNote(ctx, noteID)
	if err != nil || n == nil {
		return false, err
	}
	fileName := uniqueName(used, sanitizeName(n.Title, "Заметка"), "."+ext)
	var data []byte
	if ext == "docx" {
		if data, err = docx.Build(n.Title, n.TextContent); err != nil {
			return false, err
		}
	} else {
		data = []byte(noteText(n))
	}
	w, err := zw.Create(path.Join(prefix, fileName))
	if err != nil {
		return false, err
	}
	if _, err := w.Write(data); err != nil {
		return false, err
	}
	return true, nil
}

// zipFolder — рекурсивно упаковать заметки папки и её подпапки; prefix — путь
// внутри архива. count — счётчик записанных заметок (для проверки пустоты).
func (s *Service) zipFolder(ctx context.Context, zw *zip.Writer, userID, folderID int64, prefix, ext string, count *int) error {
	used := map[string]int{}
	notes, err := s.repo.ListNotes(ctx, domain.NoteListFilter{OwnerID: userID, FolderID: &folderID, FolderSet: true})
	if err != nil {
		return err
	}
	for _, tile := range notes {
		wrote, err := s.writeNoteFile(ctx, zw, tile.ID, prefix, ext, used)
		if err != nil {
			return err
		}
		if wrote {
			*count++
		}
	}
	children, err := s.repo.ListChildFolders(ctx, folderID)
	if err != nil {
		return err
	}
	for _, c := range children {
		dir := uniqueName(used, sanitizeName(c.Name, "Папка"), "")
		if err := s.zipFolder(ctx, zw, userID, c.ID, path.Join(prefix, dir), ext, count); err != nil {
			return err
		}
	}
	return nil
}

// ExportScope — zip особой группировки: all (все свои заметки, с деревом папок),
// archive (архивные, плоско), shared (расшаренные мне, плоско).
func (s *Service) ExportScope(ctx context.Context, userID int64, scope, format string) (*ExportFile, error) {
	ext := "txt"
	if format == FormatDOCX {
		ext = "docx"
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	used := map[string]int{}
	name := "Заметки"
	count := 0

	writeTiles := func(tiles []*domain.Note) error {
		for _, t := range tiles {
			wrote, err := s.writeNoteFile(ctx, zw, t.ID, "", ext, used)
			if err != nil {
				return err
			}
			if wrote {
				count++
			}
		}
		return nil
	}

	switch scope {
	case "shared":
		name = "Поделились со мной"
		tiles, err := s.repo.ListSharedWithMe(ctx, userID, s.companyIDs(ctx, userID), "")
		if err != nil {
			return nil, err
		}
		if err := writeTiles(tiles); err != nil {
			return nil, err
		}
	case "archive":
		name = "Архив"
		tiles, err := s.repo.ListNotes(ctx, domain.NoteListFilter{OwnerID: userID, Archived: true})
		if err != nil {
			return nil, err
		}
		if err := writeTiles(tiles); err != nil {
			return nil, err
		}
	default: // all — заметки корня + всё дерево папок
		var root *int64
		tiles, err := s.repo.ListNotes(ctx, domain.NoteListFilter{OwnerID: userID, FolderSet: true, FolderID: root})
		if err != nil {
			return nil, err
		}
		if err := writeTiles(tiles); err != nil {
			return nil, err
		}
		folders, err := s.repo.ListFolders(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, f := range folders {
			if f.ParentID != nil {
				continue // только корневые — дальше рекурсия внутри zipFolder
			}
			dir := uniqueName(used, sanitizeName(f.Name, "Папка"), "")
			if err := s.zipFolder(ctx, zw, userID, f.ID, dir, ext, &count); err != nil {
				return nil, err
			}
		}
	}

	if count == 0 {
		return nil, domain.ErrNothingToExport
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return &ExportFile{Data: buf.Bytes(), Name: name, Ext: "zip"}, nil
}

// Import — заметка из .txt/.docx (текст уже извлечён транспортом): первая строка
// → заголовок, остальное → документ из параграфов. folderID — целевая папка.
func (s *Service) Import(ctx context.Context, userID int64, text string, folderID *int64) (*domain.Note, error) {
	if err := s.checkOwnFolder(ctx, userID, folderID); err != nil {
		return nil, err
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	title, body, _ := strings.Cut(text, "\n")
	title = strings.TrimSpace(title)
	if r := []rune(title); len(r) > 300 {
		title = string(r[:300])
	}
	body = strings.TrimLeft(body, "\n")
	doc := domain.TextToDoc(body)
	n := &domain.Note{
		OwnerID: userID, FolderID: folderID, Title: title, Doc: doc,
		TextContent: domain.DocText(doc), TagIDs: []int64{},
	}
	if err := s.repo.CreateNote(ctx, n); err != nil {
		return nil, err
	}
	s.publishNote(ctx, "note:created", n)
	return n, nil
}

// sanitizeName — безопасное имя файла в архиве (без разделителей путей).
func sanitizeName(name, fallback string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\n', '\r', '\t':
			return '_'
		}
		return r
	}, name)
	if name == "" {
		return fallback
	}
	if r := []rune(name); len(r) > 100 {
		name = string(r[:100])
	}
	return name
}
