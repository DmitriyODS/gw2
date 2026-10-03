package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
)

// copyFiles — хранилище, выдающее каждому сохранению свой ключ.
type copyFiles struct {
	nopFiles
	saved int
}

func (f *copyFiles) SaveFor(_ context.Context, _, _ int64, _ string, _ []byte) (string, error) {
	f.saved++
	return fmt.Sprintf("boards/copy%d.png", f.saved), nil
}

// Копия доски обязана получить СВОИ файлы картинок: общий ключ удаление
// оригинала стирало бы и у копии.
func TestCopyBoardDuplicatesImages(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	files := &copyFiles{}
	svc := New(Deps{Repo: repo, Users: users, Files: files, Bus: nopBus{}, Limiter: allowLimiter{}, Log: discardLogger()})
	scene := json.RawMessage(`{"version":4,"objects":[
		{"id":"a","type":"image","src":"/uploads/boards/orig.png","extra":1},
		{"id":"b","type":"image","src":"/uploads/boards/orig.png"},
		{"id":"c","type":"rect"}]}`)
	src := &domain.Board{OwnerID: 1, Title: "Схема", Scene: scene}
	if err := repo.CreateBoard(ctx(), src); err != nil {
		t.Fatal(err)
	}

	cp, err := svc.CopyBoard(ctx(), 1, src.ID)
	if err != nil {
		t.Fatal(err)
	}
	keys := domain.SceneImageKeys(cp.Scene)
	if len(keys) != 2 || keys[0] != "boards/copy1.png" || keys[1] != "boards/copy1.png" {
		t.Fatalf("копия смотрит на чужие файлы: %v", keys)
	}
	if files.saved != 1 {
		t.Fatalf("одна картинка дважды на сцене — один файл, сохранено %d", files.saved)
	}
	var root map[string]any
	_ = json.Unmarshal(cp.Scene, &root)
	if obj := root["objects"].([]any)[0].(map[string]any); obj["extra"] == nil {
		t.Fatal("переписывание ключей потеряло поля клиента")
	}
	if got := domain.SceneImageKeys(src.Scene); got[0] != "boards/orig.png" {
		t.Fatalf("оригинал изменился: %v", got)
	}
}
