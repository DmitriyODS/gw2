package pasetoauth

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Активная компания уезжает слою сервиса контекстом запроса: на этом держится
// company-scope реестров и форм, а связка «Locals Fiber ≡ user values
// fasthttp-контекста» неочевидна и легко ломается сменой версии.
func TestCompanyFromContext(t *testing.T) {
	app := fiber.New()
	var got int64
	app.Get("/with", func(c *fiber.Ctx) error {
		c.Locals(localCompanyID, int64(42))
		got = CompanyFromContext(c.Context())
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/without", func(c *fiber.Ctx) error {
		got = CompanyFromContext(c.Context())
		return c.SendStatus(fiber.StatusOK)
	})

	for _, tc := range []struct {
		path string
		want int64
	}{{"/with", 42}, {"/without", 0}} {
		got = -1
		if _, err := app.Test(httptest.NewRequest("GET", tc.path, nil)); err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		if got != tc.want {
			t.Fatalf("%s: активная компания %d, ожидалась %d", tc.path, got, tc.want)
		}
	}

	if id := CompanyFromContext(context.Background()); id != 0 {
		t.Fatalf("вне запроса компании быть не может, получено %d", id)
	}
	if id := CompanyFromContext(WithCompany(context.Background(), 7)); id != 7 {
		t.Fatalf("WithCompany: получено %d", id)
	}
	if id := CompanyFromContext(nil); id != 0 {
		t.Fatalf("nil-контекст: получено %d", id)
	}
}
