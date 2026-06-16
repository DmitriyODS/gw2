package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

func TestGenLogin(t *testing.T) {
	cases := []struct{ fio, want string }{
		{"Осиповский Дмитрий Сергеевич", "osipov.ds"},
		{"Иванов Пётр", "ivanov.p"},   // нет отчества
		{"Ли Анна Петровна", "li.ap"}, // фамилия короче 6 букв
		{"Кузнецова Юлия Юрьевна", "kuznec.yy"},
		{"", ""},
	}
	for _, c := range cases {
		if got := genLogin(c.fio); got != c.want {
			t.Errorf("genLogin(%q) = %q, ожидалось %q", c.fio, got, c.want)
		}
	}
}

func TestSuggestLoginCollision(t *testing.T) {
	svc, repo, _ := newTestService(t)
	repo.add(&domain.User{FIO: "Иванов Пётр", Login: "ivanov.p"})

	got, err := svc.SuggestLogin(context.Background(), "Иванов Пётр")
	if err != nil {
		t.Fatalf("SuggestLogin: %v", err)
	}
	if got != "ivanov.p2" {
		t.Errorf("при коллизии ожидался ivanov.p2, получено %q", got)
	}
}
