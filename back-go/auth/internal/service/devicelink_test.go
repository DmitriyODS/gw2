package service

import (
	"context"
	"testing"
)

func TestLinkLoginFlow(t *testing.T) {
	svc, repo, _ := newTestService(t)
	u := employee(repo, "petrov", nil) // без компаний → сессия без login-gate

	start, err := svc.LinkStart(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if start.Kind != "login" || start.Code == "" || start.Secret == "" {
		t.Fatalf("некорректный start: %+v", start)
	}

	// До подтверждения claim с верным секретом — pending.
	claim, err := svc.LinkClaim(context.Background(), start.Code, start.Secret)
	if err != nil || claim.Status != "pending" {
		t.Fatalf("ожидался pending: %+v, %v", claim, err)
	}

	// Чужой секрет — отказ (посторонний, засветивший QR, сессию не заберёт).
	if _, err := svc.LinkClaim(context.Background(), start.Code, "deadbeef"); err != errLinkForbidden {
		t.Fatalf("ожидался LINK_FORBIDDEN при чужом секрете, получено %v", err)
	}

	if err := svc.LinkApprove(context.Background(), start.Code, u.ID, nil); err != nil {
		t.Fatalf("approve: %v", err)
	}

	claim, err = svc.LinkClaim(context.Background(), start.Code, start.Secret)
	if err != nil || claim.Status != "ok" || claim.Session == nil || claim.Session.UserID != u.ID {
		t.Fatalf("ожидалась сессия петрова: %+v, %v", claim, err)
	}
	if claim.Session.AccessToken == "" {
		t.Fatalf("нет access-токена в claim-сессии")
	}

	// Одноразовость: повторный claim — код уже погашен.
	claim, err = svc.LinkClaim(context.Background(), start.Code, start.Secret)
	if err != nil || claim.Status != "expired" {
		t.Fatalf("ожидался expired после одноразового claim: %+v, %v", claim, err)
	}
}

func TestLinkTVRequiresCompany(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(7)
	u := employee(repo, "director", &cid) // член компании 7

	start, err := svc.LinkStart(context.Background(), "tv")
	if err != nil || start.Kind != "tv" {
		t.Fatalf("tv start: %+v, %v", start, err)
	}

	// Подтверждающий без активной компании — отказ.
	if err := svc.LinkApprove(context.Background(), start.Code, u.ID, nil); err != errLinkNeedCompany {
		t.Fatalf("ожидался LINK_NEED_COMPANY без активной компании, получено %v", err)
	}

	// С активной компанией — киоск входит сразу в неё.
	if err := svc.LinkApprove(context.Background(), start.Code, u.ID, &cid); err != nil {
		t.Fatalf("approve tv: %v", err)
	}
	claim, err := svc.LinkClaim(context.Background(), start.Code, start.Secret)
	if err != nil || claim.Status != "ok" || claim.Session == nil {
		t.Fatalf("ожидалась tv-сессия: %+v, %v", claim, err)
	}
	if claim.Session.CompanyID == nil || *claim.Session.CompanyID != cid {
		t.Fatalf("tv-сессия должна быть в компании %d: %+v", cid, claim.Session.CompanyID)
	}
}

func TestLinkApproveExpired(t *testing.T) {
	svc, repo, _ := newTestService(t)
	u := employee(repo, "ghost", nil)
	if err := svc.LinkApprove(context.Background(), "ZZZZZZ", u.ID, nil); err != errLinkExpired {
		t.Fatalf("ожидался LINK_EXPIRED для неизвестного кода, получено %v", err)
	}
}
