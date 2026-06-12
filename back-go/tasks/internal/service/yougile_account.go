package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/yougile"
)

// Подключение пользователя к YouGile и работа с его ключом
// (порт account_service.py). Хранение — user_yougile_accounts (1:1 к user_id).

// Status — GET /api/yougile/status.
//
// company_enabled означает «интеграция реально работоспособна»: включён флаг
// + директор выбрал компанию + есть доска + резолвлена первая колонка. Если
// что-то не задано, фронт показывает старое простое поле «ссылка на задачу
// YouGile» и не пытается дёргать импорт/экспорт.
func (y *Yougile) Status(ctx context.Context, user *domain.User) (*dto.YougileStatus, error) {
	companyEnabled := false
	if user.CompanyID != nil {
		company, err := y.repo.GetYougileCompany(ctx, *user.CompanyID)
		if err != nil {
			return nil, err
		}
		companyEnabled = company != nil && company.UsesYougile &&
			company.YgCompanyID != nil && company.YgBoardID != nil &&
			company.YgFirstColumnID != nil
	}

	acc, err := y.repo.GetYougileAccount(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return &dto.YougileStatus{CompanyEnabled: companyEnabled}, nil
	}
	var lastValidated *dto.JSONTime
	if acc.LastValidatedAt != nil {
		t := dto.JSONTime(*acc.LastValidatedAt)
		lastValidated = &t
	}
	return &dto.YougileStatus{
		CompanyEnabled:  companyEnabled,
		Connected:       true,
		KeyFingerprint:  &acc.KeyFingerprint,
		LastValidatedAt: lastValidated,
		YgCompanyID:     &acc.YgCompanyID,
		YgLogin:         &acc.YgLogin,
	}, nil
}

// LookupCompanies — прозрачный прокси `/auth/companies` (админ-визард).
// Все ошибки YG конвертим в понятный код.
func (y *Yougile) LookupCompanies(ctx context.Context, login, password string) ([]dto.YougileCompanyItem, error) {
	items, err := y.listCompaniesForCredentials(login, password)
	if err != nil {
		return nil, err
	}
	out := make([]dto.YougileCompanyItem, 0, len(items))
	for _, c := range items {
		if id, _ := c["id"].(string); id != "" {
			name, _ := c["name"].(string)
			out = append(out, dto.YougileCompanyItem{ID: id, Name: name})
		}
	}
	return out, nil
}

func (y *Yougile) listCompaniesForCredentials(login, password string) ([]map[string]any, error) {
	items, err := y.newClient("").ListCompanies(login, password)
	if err != nil {
		if yougile.IsAuth(err) {
			return nil, domain.NewError("BAD_CREDENTIALS", "Неверный логин или пароль", 400)
		}
		return nil, domain.NewError("YOUGILE_ERROR", "Ошибка YouGile: "+err.Error(), 400)
	}
	return items, nil
}

// selectYgCompany — найти yg_company по id среди компаний пользователя.
// nil — у пользователя нет такой компании (админ ещё не пригласил его).
func (y *Yougile) selectYgCompany(login, password, targetID string) (map[string]any, error) {
	items, err := y.listCompaniesForCredentials(login, password)
	if err != nil {
		return nil, err
	}
	for _, c := range items {
		if id, _ := c["id"].(string); id == targetID {
			return c, nil
		}
	}
	return nil, nil
}

// Connect — универсальный коннект: используется и обычным юзером, и админом.
// explicitYgCompanyID задан — берём его (админ в визарде); иначе —
// company.yg_company_id (обычный юзер).
func (y *Yougile) Connect(ctx context.Context, user *domain.User, login, password string,
	explicitYgCompanyID *string) (*dto.YougileConnectResult, error) {

	if user.CompanyID == nil {
		return nil, domain.NewError("NO_COMPANY",
			"Эта функция доступна только пользователям компании. "+
				"Администратору системы нужно войти как директор конкретной компании.", 400)
	}
	company, err := y.repo.GetYougileCompany(ctx, *user.CompanyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.NewError("NO_COMPANY",
			"Эта функция доступна только пользователям компании. "+
				"Администратору системы нужно войти как директор конкретной компании.", 400)
	}

	targetYgID := strOrEmpty(explicitYgCompanyID)
	if targetYgID == "" {
		targetYgID = strOrEmpty(company.YgCompanyID)
	}
	if targetYgID == "" {
		return nil, domain.NewError("COMPANY_NOT_CONFIGURED",
			"В компании ещё не выбрана YouGile-компания — обратитесь к администратору", 400)
	}

	ygCompany, err := y.selectYgCompany(login, password, targetYgID)
	if err != nil {
		return nil, err
	}
	if ygCompany == nil {
		return nil, domain.NewError("NO_ACCESS_TO_COMPANY",
			"У вашего аккаунта YouGile нет доступа к этой компании. "+
				"Попросите администратора пригласить вас.", 400)
	}

	// Шаг 2 — собственно ключ. /auth/keys на одних и тех же кредах
	// возвращает тот же ключ, но всё равно перешифровываем — вдруг логика
	// YG изменится.
	key, err := y.newClient("").CreateKey(login, password, targetYgID)
	if err != nil {
		if yougile.IsAuth(err) {
			return nil, domain.NewError("BAD_CREDENTIALS", "Неверный логин или пароль", 400)
		}
		return nil, domain.NewError("YOUGILE_ERROR", "Ошибка YouGile: "+err.Error(), 400)
	}

	// yg_user_id — пригодится при «назначить себя» во время экспорта.
	// Не блокируем подключение, но логируем.
	var ygUserID *string
	if me, err := y.newClient(key).Me(); err != nil {
		y.log.Warn("yougile.me_failed_on_connect", "user_id", user.ID, "error", err)
	} else if id, _ := me["id"].(string); id != "" {
		ygUserID = &id
	}

	// Переподключение с другим логином — отзовём прошлый ключ в YG, чтобы
	// не плодить ключи в его аккаунте. Best-effort: если YG лежит, всё
	// равно сохраним новую привязку.
	existing, err := y.repo.GetYougileAccount(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		oldKey, err := y.cipher.DecryptKey(existing.KeyCiphertext)
		if err != nil {
			return nil, wrapMisconfig(err)
		}
		if oldKey != "" && oldKey != key {
			if err := y.newClient("").DeleteKey(oldKey); err != nil {
				y.log.Warn("yougile.old_key_revoke_failed", "user_id", user.ID, "error", err)
			}
		}
	}

	ciphertext, err := y.cipher.EncryptKey(key)
	if err != nil {
		return nil, wrapMisconfig(err)
	}
	now := time.Now().UTC()
	acc := &domain.YougileAccount{
		UserID:          user.ID,
		CompanyID:       *user.CompanyID,
		YgCompanyID:     targetYgID,
		YgUserID:        ygUserID,
		YgLogin:         login,
		KeyCiphertext:   ciphertext,
		KeyFingerprint:  yougile.MakeFingerprint(key),
		LastValidatedAt: &now,
	}
	if err := y.repo.UpsertYougileAccount(ctx, acc); err != nil {
		return nil, err
	}
	y.log.Info("yougile.connected", "user_id", user.ID, "yg_company_id", targetYgID)
	return &dto.YougileConnectResult{
		Connected:      true,
		KeyFingerprint: acc.KeyFingerprint,
		YgCompanyID:    acc.YgCompanyID,
		YgLogin:        acc.YgLogin,
	}, nil
}

// Disconnect — отозвать ключ в YG и удалить локальную привязку. Отзыв
// best-effort: пользователь должен отвязаться, даже если YG недоступен.
func (y *Yougile) Disconnect(ctx context.Context, userID int64) error {
	acc, err := y.repo.GetYougileAccount(ctx, userID)
	if err != nil {
		return err
	}
	if acc == nil {
		return nil
	}
	key, err := y.cipher.DecryptKey(acc.KeyCiphertext)
	if err != nil {
		return wrapMisconfig(err)
	}
	if key != "" {
		if err := y.newClient("").DeleteKey(key); err != nil {
			y.log.Warn("yougile.revoke_failed", "user_id", userID, "error", err)
		}
	}
	if err := y.repo.DeleteYougileAccount(ctx, userID); err != nil {
		return err
	}
	y.log.Info("yougile.disconnected", "user_id", userID)
	return nil
}

// Rotate — перевыпустить ключ. Принципиально требуем пароль повторно.
func (y *Yougile) Rotate(ctx context.Context, user *domain.User, password string) (*dto.YougileRotateResult, error) {
	acc, err := y.repo.GetYougileAccount(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, domain.NewError("NOT_CONNECTED", "Аккаунт YouGile не подключён", 400)
	}
	// Уже сохранённый login + переданный заново password.
	res, err := y.Connect(ctx, user, acc.YgLogin, password, &acc.YgCompanyID)
	if err != nil {
		return nil, err
	}
	return &dto.YougileRotateResult{Connected: true, KeyFingerprint: res.KeyFingerprint}, nil
}
