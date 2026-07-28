package domain

import "math"

// Платформенный ИИ: ключ proxy-api и каталог моделей задаёт СУПЕР-АДМИН в
// разделе «Аудит платформы», а пользователю тариф выдаёт токены доступа
// (баланс ведёт billingsvc).
//
// Токен доступа — единица тарифа, а не токен модели: 1 токен доступа = 1000
// токенов САМОЙ ДЕШЁВОЙ модели каталога (эмбеддинги). Обращение к модели
// дороже пропорционально её цене — так «1000 токенов» тарифа значат одно и то
// же, какой бы моделью человек ни пользовался.

// Модели по умолчанию (совпадают с миграцией 00068).
const (
	PlatformModelChat      = "gpt-5.4-nano"
	PlatformModelEmbedding = "text-embedding-3-small"
)

// TokensPerAccessToken — сколько токенов базовой (самой дешёвой) модели
// умещается в одном токене доступа.
const TokensPerAccessToken = 1000

// PlatformAI — единственная строка ai_platform_settings.
type PlatformAI struct {
	Enabled        bool
	APIKeyEnc      []byte // nil — платформенный ИИ выключен
	KeyHint        string
	BaseURL        string
	ModelChat      string
	ModelEmbedding string
	ModelSupport   string
}

// Ready — можно ли обращаться к моделям на платформенном ключе.
func (p *PlatformAI) Ready() bool {
	return p != nil && p.Enabled && len(p.APIKeyEnc) > 0
}

// AIModel — строка каталога ai_models.
type AIModel struct {
	Code         string `json:"code"`
	Title        string `json:"title"`
	Kind         string `json:"kind"` // chat | embedding
	PricePerMTok int64  `json:"price_per_mtok"`
	Selectable   bool   `json:"selectable"`
	IsActive     bool   `json:"is_active"`
	Sort         int    `json:"sort"`
}

// ModelCatalog — каталог моделей с расчётом стоимости обращения.
type ModelCatalog struct {
	models []*AIModel
	base   int64 // цена самой дешёвой активной модели, копейки за 1 млн токенов
}

func NewModelCatalog(models []*AIModel) *ModelCatalog {
	c := &ModelCatalog{models: models}
	for _, m := range models {
		if !m.IsActive || m.PricePerMTok <= 0 {
			continue
		}
		if c.base == 0 || m.PricePerMTok < c.base {
			c.base = m.PricePerMTok
		}
	}
	return c
}

func (c *ModelCatalog) All() []*AIModel { return c.models }

// Selectable — модели, которые пользователь выбирает в настройках.
func (c *ModelCatalog) Selectable() []*AIModel {
	out := []*AIModel{}
	for _, m := range c.models {
		if m.IsActive && m.Selectable {
			out = append(out, m)
		}
	}
	return out
}

// Get — модель по коду; nil — такой нет.
func (c *ModelCatalog) Get(code string) *AIModel {
	for _, m := range c.models {
		if m.Code == code {
			return m
		}
	}
	return nil
}

// Allowed — можно ли работать этой моделью (есть в каталоге и включена).
func (c *ModelCatalog) Allowed(code string) bool {
	m := c.Get(code)
	return m != nil && m.IsActive
}

// AccessTokens — во сколько токенов ДОСТУПА обходится расход tokens токенов
// модели. Округление вверх: бесплатных обращений не бывает.
func (c *ModelCatalog) AccessTokens(model string, tokens int) int64 {
	if tokens <= 0 {
		return 0
	}
	base := c.base
	m := c.Get(model)
	// Неизвестная модель или пустой каталог: считаем один к одному по базовой
	// единице — лучше списать по-честному минимум, чем не списать вовсе.
	if base <= 0 || m == nil || m.PricePerMTok <= 0 {
		return int64(math.Ceil(float64(tokens) / TokensPerAccessToken))
	}
	ratio := float64(m.PricePerMTok) / float64(base)
	return int64(math.Ceil(float64(tokens) * ratio / TokensPerAccessToken))
}

// Coefficient — во сколько раз модель дороже базовой (для карточки настроек).
func (c *ModelCatalog) Coefficient(model string) float64 {
	m := c.Get(model)
	if m == nil || c.base <= 0 || m.PricePerMTok <= 0 {
		return 1
	}
	return float64(m.PricePerMTok) / float64(c.base)
}

// Возможности ИИ, которые включаются по отдельности.
const (
	// Личные: чат Hola-ассистента и инструменты текста в заметках.
	FeatureAssistant = "assistant"
	FeatureNotes     = "notes"
	// Компанийные: умный поиск задач и факты ТВ-режима.
	FeatureSearch = "search"
	FeatureTVFact = "tv_fact"
)
