package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

/* Согласие с правовыми документами (152-ФЗ).

   Проверяем то, ради чего гейт заведён: пока согласия нет, сессия несёт
   legal_required (мидлварь по нему закрывает все сервисы); принятие снимает
   флаг и оставляет строку журнала — доказательство факта согласия; частичное
   согласие (лицензия без ПДн) не принимается. */

// enableLegalGate — включить гейт на время теста (по умолчанию он снят, см.
// domain.LegalGateEnabled) и вернуть как было.
func enableLegalGate(t *testing.T) {
	t.Helper()
	prev := domain.LegalGateEnabled
	domain.LegalGateEnabled = true
	t.Cleanup(func() { domain.LegalGateEnabled = prev })
}

func TestLegalGateBlocksUntilAccepted(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	u := employee(repo, "ivanov", &cid)
	ctx := context.Background()

	// Гейт придержан до отдельного выпуска — включаем его на время теста,
	// иначе блокировка осталась бы непокрытой.
	enableLegalGate(t)

	state, err := svc.LegalState(ctx, u.ID)
	if err != nil {
		t.Fatalf("LegalState: %v", err)
	}
	if !state.Required || state.AcceptedVersion != nil {
		t.Fatalf("новый аккаунт обязан спрашивать согласие: %+v", state)
	}

	sess, err := svc.session(ctx, repo.users[u.ID], &cid, false)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if !sess.LegalRequired {
		t.Fatal("сессия без согласия обязана нести legal_required")
	}

	accepted, err := svc.AcceptLegal(ctx, u.ID, &cid, dto.LegalAcceptRequest{
		Version:   domain.LegalVersion,
		Documents: domain.LegalRequiredDocuments,
	}, domain.Consent{IP: "10.0.0.7", UserAgent: "Firefox"})
	if err != nil {
		t.Fatalf("AcceptLegal: %v", err)
	}
	if accepted.LegalRequired {
		t.Fatal("после согласия сессия обязана перевыпускаться без legal_required")
	}
	// Активная компания не должна теряться при перевыпуске.
	if accepted.CompanyID == nil || *accepted.CompanyID != cid {
		t.Fatalf("перевыпуск потерял активную компанию: %+v", accepted.CompanyID)
	}

	state, err = svc.LegalState(ctx, u.ID)
	if err != nil {
		t.Fatalf("LegalState: %v", err)
	}
	if state.Required || state.AcceptedAt == nil ||
		state.AcceptedVersion == nil || *state.AcceptedVersion != domain.LegalVersion {
		t.Fatalf("согласие не сохранилось: %+v", state)
	}

	// Журнал — доказательство: редакция, документы и обстоятельства согласия.
	if len(repo.consents) != 1 {
		t.Fatalf("ожидалась одна строка журнала, получено %d", len(repo.consents))
	}
	c := repo.consents[0]
	if c.UserID != u.ID || c.Version != domain.LegalVersion ||
		len(c.Documents) != len(domain.LegalRequiredDocuments) ||
		c.IP != "10.0.0.7" || c.UserAgent != "Firefox" {
		t.Fatalf("журнал согласий заполнен неверно: %+v", c)
	}
}

func TestLegalAcceptRejectsPartialAndStaleVersion(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	u := employee(repo, "petrov", &cid)
	ctx := context.Background()

	// Согласие на обработку ПДн отделено от лицензии намеренно (ч.1 ст.9
	// 152-ФЗ): принять лицензию и «пропустить» ПДн нельзя.
	_, err := svc.AcceptLegal(ctx, u.ID, &cid, dto.LegalAcceptRequest{
		Version:   domain.LegalVersion,
		Documents: []string{domain.DocLicense, domain.DocPrivacy},
	}, domain.Consent{})
	wantCode(t, err, "LEGAL_INCOMPLETE")

	// Клиент со старой редакцией: человек читал не тот текст, что действует.
	_, err = svc.AcceptLegal(ctx, u.ID, &cid, dto.LegalAcceptRequest{
		Version:   "0.9",
		Documents: domain.LegalRequiredDocuments,
	}, domain.Consent{})
	wantCode(t, err, "LEGAL_VERSION_MISMATCH")

	if len(repo.consents) != 0 {
		t.Fatalf("отклонённое согласие не должно попадать в журнал: %+v", repo.consents)
	}
	enableLegalGate(t)
	state, _ := svc.LegalState(ctx, u.ID)
	if !state.Required {
		t.Fatal("после отказа гейт обязан оставаться закрытым")
	}
}

// Пока гейт придержан, никто ничего не обязан принимать: клейм не ставится, и
// сервисы никого не блокируют. Тест сторожит выпуск от случайного включения.
func TestLegalGateDisabledLetsEveryoneIn(t *testing.T) {
	svc, repo, _ := newTestService(t)
	cid := int64(1)
	u := employee(repo, "sidorov", &cid)
	ctx := context.Background()

	state, err := svc.LegalState(ctx, u.ID)
	if err != nil {
		t.Fatalf("LegalState: %v", err)
	}
	if state.Required != domain.LegalGateEnabled {
		t.Fatalf("состояние не следует за флагом гейта: %+v", state)
	}
	sess, err := svc.session(ctx, repo.users[u.ID], &cid, false)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if sess.LegalRequired != domain.LegalGateEnabled {
		t.Fatalf("клейм сессии не следует за флагом гейта: %v", sess.LegalRequired)
	}
}
