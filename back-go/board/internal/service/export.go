package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"path"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
)

// Форматы выгрузки доски: svg — векторная картинка (открывается где угодно),
// json — сама сцена (её же принимает импорт). Растр (png) делает клиент из
// холста — серверу растеризатор не нужен.
const (
	FormatSVG  = "svg"
	FormatJSON = "json"
)

// ExportFile — выгруженный файл: содержимое + имя (без расширения) + расширение.
type ExportFile struct {
	Data []byte
	Name string
	Ext  string
}

// imageResolver — читатель картинок холста для встраивания в SVG (data-URI).
func (s *Service) imageResolver() func(string) (string, []byte, error) {
	return func(key string) (string, []byte, error) {
		data, err := s.files.Open(key)
		if err != nil {
			return "", nil, err
		}
		return mimeByKey(key), data, nil
	}
}

// mimeByKey — тип картинки по расширению ключа (в холсте только растр).
func mimeByKey(key string) string {
	switch strings.ToLower(path.Ext(key)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// boardFile — содержимое выгрузки одной доски в выбранном формате.
func (s *Service) boardFile(b *domain.Board, format string) ([]byte, string) {
	if format == FormatJSON {
		return []byte(b.Scene), FormatJSON
	}
	return domain.SceneSVG(b.Scene, s.imageResolver()), FormatSVG
}

// Export — доска в svg или json. Доступен и адресатам шаринга (чтение есть —
// выгрузка тоже).
func (s *Service) Export(ctx context.Context, userID, id int64, format string) (*ExportFile, error) {
	b, _, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(b.Title)
	if name == "" {
		name = "Доска"
	}
	data, ext := s.boardFile(b, format)
	return &ExportFile{Data: data, Name: name, Ext: ext}, nil
}

// ExportFolder — zip со всем поддеревом папки: подпапки как каталоги, доски —
// файлами. Только владелец.
func (s *Service) ExportFolder(ctx context.Context, userID, id int64, format string) (*ExportFile, error) {
	f, err := s.requireFolderOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	count := 0
	if err := s.zipFolder(ctx, zw, userID, id, "", format, &count); err != nil {
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

// writeBoardFile — записать доску файлом в архив (по её id, со сценой).
// Возвращает true, если файл действительно записан (доска существует).
func (s *Service) writeBoardFile(ctx context.Context, zw *zip.Writer, boardID int64, prefix, format string, used map[string]int) (bool, error) {
	b, err := s.repo.GetBoard(ctx, boardID)
	if err != nil || b == nil {
		return false, err
	}
	data, ext := s.boardFile(b, format)
	fileName := uniqueName(used, sanitizeName(b.Title, "Доска"), "."+ext)
	w, err := zw.Create(path.Join(prefix, fileName))
	if err != nil {
		return false, err
	}
	if _, err := w.Write(data); err != nil {
		return false, err
	}
	return true, nil
}

// zipFolder — рекурсивно упаковать доски папки и её подпапки; prefix — путь
// внутри архива. count — счётчик записанных досок (для проверки пустоты).
func (s *Service) zipFolder(ctx context.Context, zw *zip.Writer, userID, folderID int64, prefix, format string, count *int) error {
	used := map[string]int{}
	boards, err := s.repo.ListBoards(ctx, domain.BoardListFilter{OwnerID: userID, FolderID: &folderID, FolderSet: true})
	if err != nil {
		return err
	}
	for _, tile := range boards {
		wrote, err := s.writeBoardFile(ctx, zw, tile.ID, prefix, format, used)
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
		if err := s.zipFolder(ctx, zw, userID, c.ID, path.Join(prefix, dir), format, count); err != nil {
			return err
		}
	}
	return nil
}

// ExportScope — zip особой группировки: all (все свои доски, с деревом папок),
// archive (архивные, плоско), shared (расшаренные мне, плоско).
func (s *Service) ExportScope(ctx context.Context, userID int64, scope, format string) (*ExportFile, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	used := map[string]int{}
	name := "Доски"
	count := 0

	writeTiles := func(tiles []*domain.Board) error {
		for _, t := range tiles {
			wrote, err := s.writeBoardFile(ctx, zw, t.ID, "", format, used)
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
		tiles, err := s.repo.ListBoards(ctx, domain.BoardListFilter{OwnerID: userID, Archived: true})
		if err != nil {
			return nil, err
		}
		if err := writeTiles(tiles); err != nil {
			return nil, err
		}
	default: // all — доски корня + всё дерево папок
		var root *int64
		tiles, err := s.repo.ListBoards(ctx, domain.BoardListFilter{OwnerID: userID, FolderSet: true, FolderID: root})
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
			if err := s.zipFolder(ctx, zw, userID, f.ID, dir, format, &count); err != nil {
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

// Import — доска из файла: .json — ранее выгруженная сцена (как есть), .txt —
// текст надписями по строкам. folderID — целевая папка.
func (s *Service) Import(ctx context.Context, userID int64, title string, data []byte, isScene bool, folderID *int64) (*domain.Board, error) {
	if err := s.checkOwnFolder(ctx, userID, folderID); err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	scene := domain.EmptyScene()
	if isScene {
		if !json.Valid(data) {
			return nil, domain.ErrBadScene
		}
		// Прогон через ParseScene нормализует чужую/старую структуру, а Marshal
		// отсекает лишние поля — в БД не попадает произвольный JSON.
		normalized, err := json.Marshal(domain.ParseScene(data))
		if err != nil {
			return nil, domain.ErrBadScene
		}
		scene = normalized
	} else {
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		firstLine, body, _ := strings.Cut(text, "\n")
		if title == "" {
			title = strings.TrimSpace(firstLine)
			text = strings.TrimLeft(body, "\n")
		}
		scene = domain.TextToScene(text)
	}
	if r := []rune(title); len(r) > 300 {
		title = string(r[:300])
	}
	b := &domain.Board{
		OwnerID: userID, FolderID: folderID, Title: title, Scene: scene,
		TextContent: domain.SceneText(scene),
	}
	if err := s.repo.CreateBoard(ctx, b); err != nil {
		return nil, err
	}
	s.publishBoard(ctx, "board:created", b)
	return b, nil
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
