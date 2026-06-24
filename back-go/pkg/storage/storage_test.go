package storage

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sort"
	"testing"
)

func newTestLocal(t *testing.T) Storage {
	t.Helper()
	return NewLocal(t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestLocalPutOpenRoundTrip(t *testing.T) {
	st := newTestLocal(t)
	ctx := context.Background()
	want := []byte("hello")
	if err := st.Put(ctx, "registry/a/b.txt", want, "text/plain"); err != nil {
		t.Fatalf("put: %v", err)
	}
	rc, err := st.Open(ctx, "registry/a/b.txt")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if string(got) != string(want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestLocalCopyAndRemove(t *testing.T) {
	st := newTestLocal(t)
	ctx := context.Background()
	_ = st.Put(ctx, "messages/2026/06/src.bin", []byte("data"), "")
	if err := st.Copy(ctx, "messages/2026/06/src.bin", "messages/2026/06/dst.bin"); err != nil {
		t.Fatalf("copy: %v", err)
	}
	rc, err := st.Open(ctx, "messages/2026/06/dst.bin")
	if err != nil {
		t.Fatalf("open copy: %v", err)
	}
	rc.Close()

	// Remove одной копии не задевает другую.
	st.Remove(ctx, "messages/2026/06/src.bin")
	if _, err := st.Open(ctx, "messages/2026/06/src.bin"); !os.IsNotExist(err) {
		t.Fatalf("src must be gone, err=%v", err)
	}
	if _, err := st.Open(ctx, "messages/2026/06/dst.bin"); err != nil {
		t.Fatalf("dst must survive: %v", err)
	}

	// Remove несуществующего и пути с ".." — без паники и без эффекта.
	st.Remove(ctx, "messages/2026/06/dst.bin", "", "../escape")
}

func TestLocalList(t *testing.T) {
	st := newTestLocal(t)
	ctx := context.Background()
	_ = st.Put(ctx, "avatars/a.jpg", []byte("1"), "")
	_ = st.Put(ctx, "avatars/b.png", []byte("2"), "")
	_ = st.Put(ctx, "registry/c.bin", []byte("3"), "")

	keys, err := st.List(ctx, "avatars")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "avatars/a.jpg" || keys[1] != "avatars/b.png" {
		t.Fatalf("unexpected keys: %v", keys)
	}

	// Отсутствующий префикс — пустой список без ошибки.
	empty, err := st.List(ctx, "nope")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty list expected, got %v err=%v", empty, err)
	}
}
