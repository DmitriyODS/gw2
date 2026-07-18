package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

func TestExtractTags(t *testing.T) {
	cases := []struct {
		body string
		want []string
	}{
		{"Запускаем #Релиз и #hr сегодня", []string{"релиз", "hr"}},
		{"дубли #tag #tag #TAG", []string{"tag"}},
		{"середина сло#ва не тег", nil},
		{"url https://x.com/page#anchor не тег", nil},
		{"без тегов вовсе", nil},
		{"#2024 год и #план_на_год", []string{"2024", "план_на_год"}},
	}
	for _, c := range cases {
		got := extractTags(c.body)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("extractTags(%q) = %v, want %v", c.body, got, c.want)
		}
	}
}

func TestExtractTags_LimitCount(t *testing.T) {
	body := ""
	for i := 0; i < domain.MaxPostTags+5; i++ {
		body += " #tag" + string(rune('a'+i))
	}
	if got := extractTags(body); len(got) != domain.MaxPostTags {
		t.Fatalf("ожидалось не более %d тегов, получено %d", domain.MaxPostTags, len(got))
	}
}

func TestCreatePost_StoresTags_FilterByTag(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.CreatePost(ctx, 1, 10, nil, nil, "Новость про #релиз и #команду"); err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if _, err := svc.CreatePost(ctx, 1, 10, nil, nil, "Ещё пост про #команду"); err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	feed, err := svc.ListPosts(ctx, 1, 10, PostListParams{Tag: "релиз"})
	if err != nil {
		t.Fatalf("ListPosts(tag=релиз): %v", err)
	}
	if len(feed.Posts) != 1 {
		t.Fatalf("по тегу #релиз ожидался 1 пост, получено %d", len(feed.Posts))
	}

	tags, err := svc.PopularTags(ctx, 1, 20)
	if err != nil {
		t.Fatalf("PopularTags: %v", err)
	}
	if len(tags) == 0 || tags[0].Tag != "команду" || tags[0].Count != 2 {
		t.Fatalf("ожидался топ #команду×2, получено %+v", tags)
	}
}
