package service

import (
	"strings"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// Виды товаров, которые витрина умеет применять.
var productKinds = map[string]bool{
	"theme": true, "wallpaper": true, "gradient": true,
	"pet_skin": true, "pet_decor": true, "other": true,
}

// ListProducts — витрина: только опубликованные товары.
func (s *Service) ListProducts(ctx domain.Ctx, kind, search string, viewerID int64, limit, offset int) ([]*domain.Product, int, error) {
	return s.Products.ListShowcase(ctx, kind, strings.TrimSpace(search), viewerID, limit, offset)
}

// GetProductCard — карточка товара: сам товар и признак «уже куплен».
func (s *Service) GetProductCard(ctx domain.Ctx, id, viewerID int64) (*domain.Product, error) {
	p, err := s.Products.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	// Чужой неопубликованный товар не показываем — он ещё не на витрине.
	if p.Status != domain.ProductPublished && (p.AuthorID == nil || *p.AuthorID != viewerID) {
		return nil, domain.ErrNotFound
	}
	owned, err := s.Products.IsOwned(ctx, p.ID, viewerID)
	if err != nil {
		return nil, err
	}
	p.Owned = owned
	return p, nil
}

// MyStore — раздел «Мои товары»: купленное, выставленное на продажу и кошелёк.
type MyStore struct {
	Purchases []*domain.ProductPurchase `json:"purchases"`
	Products  []*domain.Product         `json:"products"`
	Balance   *domain.SellerBalance     `json:"balance"`
	Payouts   []*domain.Payout          `json:"payouts"`
	Settings  *domain.Settings          `json:"settings"`
}

func (s *Service) MyStore(ctx domain.Ctx, userID int64) (*MyStore, error) {
	purchases, err := s.Products.ListPurchases(ctx, userID)
	if err != nil {
		return nil, err
	}
	mine, err := s.Products.ListByAuthor(ctx, userID)
	if err != nil {
		return nil, err
	}
	balance, err := s.Products.GetSellerBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	payouts, err := s.Products.ListPayouts(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	settings, err := s.Settings.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &MyStore{Purchases: purchases, Products: mine, Balance: balance,
		Payouts: payouts, Settings: settings}, nil
}

// ProductInput — что автор задаёт своему товару.
type ProductInput struct {
	Kind        string         `json:"kind"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Price       int64          `json:"price"`
	CoverPath   *string        `json:"cover_path"`
	Payload     map[string]any `json:"payload"`
}

func (in ProductInput) validate() error {
	if strings.TrimSpace(in.Title) == "" || !productKinds[in.Kind] || in.Price < 0 {
		return domain.ErrValidation
	}
	return nil
}

// CreateProduct — автор заводит товар (черновик).
func (s *Service) CreateProduct(ctx domain.Ctx, authorID int64, in ProductInput) (*domain.Product, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	p := &domain.Product{
		Kind:        in.Kind,
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		Price:       in.Price,
		AuthorID:    &authorID,
		Status:      domain.ProductDraft,
		CoverPath:   in.CoverPath,
		Payload:     in.Payload,
	}
	if err := s.Products.CreateProduct(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateProduct — правка своего товара. Опубликованный и отправленный на
// модерацию не редактируются: сначала снимите его с продажи.
func (s *Service) UpdateProduct(ctx domain.Ctx, authorID, id int64, in ProductInput) (*domain.Product, error) {
	p, err := s.authorProduct(ctx, authorID, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.ProductDraft && p.Status != domain.ProductRejected {
		return nil, domain.ErrProductLocked
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	p.Kind, p.Title, p.Description, p.Price = in.Kind, strings.TrimSpace(in.Title), in.Description, in.Price
	p.CoverPath, p.Payload = in.CoverPath, in.Payload
	if err := s.Products.UpdateProduct(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// SubmitProduct — отправить товар на модерацию.
func (s *Service) SubmitProduct(ctx domain.Ctx, authorID, id int64) error {
	p, err := s.authorProduct(ctx, authorID, id)
	if err != nil {
		return err
	}
	if p.Status == domain.ProductReview || p.Status == domain.ProductPublished {
		return domain.ErrProductLocked
	}
	return s.Products.SetProductStatus(ctx, id, domain.ProductReview, "")
}

// WithdrawProduct — снять товар с продажи (купленные копии остаются у людей).
func (s *Service) WithdrawProduct(ctx domain.Ctx, authorID, id int64) error {
	if _, err := s.authorProduct(ctx, authorID, id); err != nil {
		return err
	}
	return s.Products.SetProductStatus(ctx, id, domain.ProductRemoved, "")
}

// DeleteProduct — удалить свой товар; проданный только снимается с витрины,
// иначе покупатели потеряют купленное.
func (s *Service) DeleteProduct(ctx domain.Ctx, authorID, id int64) error {
	p, err := s.authorProduct(ctx, authorID, id)
	if err != nil {
		return err
	}
	if p.SalesCount > 0 {
		return s.Products.SetProductStatus(ctx, id, domain.ProductRemoved, "")
	}
	return s.Products.DeleteProduct(ctx, id)
}

func (s *Service) authorProduct(ctx domain.Ctx, authorID, id int64) (*domain.Product, error) {
	p, err := s.Products.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.AuthorID == nil || *p.AuthorID != authorID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

// RequestPayout — заявка автора на вывод выручки (подтверждает супер-админ).
func (s *Service) RequestPayout(ctx domain.Ctx, userID int64, amount int64, requisites string) (*domain.Payout, error) {
	if amount <= 0 || strings.TrimSpace(requisites) == "" {
		return nil, domain.ErrValidation
	}
	p := &domain.Payout{UserID: userID, Amount: amount, Requisites: strings.TrimSpace(requisites)}
	if err := s.Products.CreatePayout(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
