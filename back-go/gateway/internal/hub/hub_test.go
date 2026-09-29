package hub

import "testing"

func drain(c *Client) int {
	n := 0
	for {
		select {
		case <-c.send:
			n++
		default:
			return n
		}
	}
}

// Клиент в нескольких адресованных комнатах получает кадр один раз.
func TestBroadcastDeduplicatesAcrossRooms(t *testing.T) {
	h := New()
	c := NewClient(1)
	h.Add(c, "all", "user_1")
	h.SetCompany(c, "company_5")

	h.Broadcast([]byte("x"), "user_1", "company_5")
	if n := drain(c); n != 1 {
		t.Fatalf("кадров = %d, ожидался 1", n)
	}
}

// Смена активной компании пересаживает клиента: события прежней компании
// до него больше не доходят.
func TestSetCompanyMovesClient(t *testing.T) {
	h := New()
	c := NewClient(1)
	h.Add(c, "all", "user_1")
	h.SetCompany(c, "company_5")
	h.SetCompany(c, "company_6")

	h.Broadcast([]byte("x"), "company_5")
	if n := drain(c); n != 0 {
		t.Fatalf("событие прежней компании дошло: %d", n)
	}
	h.Broadcast([]byte("x"), "company_6")
	if n := drain(c); n != 1 {
		t.Fatalf("событие новой компании: %d", n)
	}

	h.SetCompany(c, "")
	h.Remove(c)
	if len(h.rooms) != 0 || len(h.membership) != 0 {
		t.Fatalf("после Remove остались комнаты %v / членства %v", h.rooms, h.membership)
	}
}
