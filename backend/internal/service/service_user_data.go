package service

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
)

type UserDataService struct{ repo repository.UserDataRepository }

func NewUserDataService(repo repository.UserDataRepository) *UserDataService {
	return &UserDataService{repo}
}
func (s *UserDataService) Favorite(user string, input dto.CreateFavoriteRequest) (model.Favorite, error) {
	return s.repo.CreateFavorite(model.Favorite{UserID: user, ProductID: input.ProductID, Folder: input.Folder})
}
func (s *UserDataService) Favorites(user string) ([]model.Favorite, error) {
	return s.repo.ListFavorites(user)
}
func (s *UserDataService) Budget(user string, input dto.BudgetRequest) (model.Budget, error) {
	rate := 350.0
	if input.RoomType == constants.BudgetRoomKitchen {
		rate = 580
	}
	if input.RoomType == constants.BudgetRoomBathroom {
		rate = 720
	}
	lineItems := map[string]float64{"materials": input.Area * rate, "allowance": input.Area * rate * 0.08}
	payload, err := json.Marshal(lineItems)
	if err != nil {
		return model.Budget{}, fmt.Errorf("encode budget payload: %w", err)
	}
	return s.repo.CreateBudget(model.Budget{UserID: user, RoomType: input.RoomType, Area: input.Area, Estimate: input.Area * rate * 1.08, Payload: string(payload)})
}

// AlertService owns the price-alert subscription lifecycle: baselines at
// subscribe time, listing with live comparability, and offer evaluation.
type AlertService struct {
	alerts  repository.AlertRepository
	offers  repository.OfferRepository
	product repository.ProductRepository
}

func NewAlertService(alerts repository.AlertRepository, offers repository.OfferRepository, product repository.ProductRepository) *AlertService {
	return &AlertService{alerts: alerts, offers: offers, product: product}
}

// Subscribe snapshots the current lowest valid (approved, in-stock) offer
// as the baseline. Products with no valid offer cannot be compared yet and
// are rejected so the baseline stays meaningful.
func (s *AlertService) Subscribe(user string, input dto.CreateAlertRequest) (dto.AlertView, error) {
	product, err := s.product.Get(input.ProductID)
	if err != nil {
		return dto.AlertView{}, err
	}
	baseline := lowestValidOffer(product.Offers)
	if baseline == nil {
		return dto.AlertView{}, fmt.Errorf("%w: 该商品暂时没有有效报价（店铺需已审核且有货），暂不可比较", apperrors.ErrInvalidInput)
	}
	row, err := s.alerts.Create(model.PriceAlert{
		UserID:        user,
		ProductID:     input.ProductID,
		TargetPrice:   input.TargetPrice,
		DropPercent:   input.DropPercent,
		Status:        constants.AlertStatusActive,
		BaselinePrice: baseline.UnitPrice,
	})
	if err != nil {
		return dto.AlertView{}, err
	}
	return s.toView(row, *baseline), nil
}

func (s *AlertService) List(user string) ([]dto.AlertView, error) {
	rows, err := s.alerts.ListByUser(user)
	if err != nil {
		return nil, err
	}
	productIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		productIDs = append(productIDs, row.ProductID)
	}
	lowest, err := s.offers.LowestValidByProducts(productIDs)
	if err != nil {
		return nil, err
	}
	views := make([]dto.AlertView, 0, len(rows))
	for _, row := range rows {
		views = append(views, s.toView(row, lowest[row.ProductID]))
	}
	return views, nil
}

func (s *AlertService) toView(row model.PriceAlert, lowest model.Offer) dto.AlertView {
	view := dto.AlertView{
		ID:          row.ID,
		ProductID:   row.ProductID,
		Product:     dto.AlertProductView{ID: row.ProductID, Name: row.Product.Name, Unit: row.Product.Unit},
		Status:      row.Status,
		Baseline:    row.BaselinePrice,
		TargetPrice: row.TargetPrice,
		DropPercent: row.DropPercent,
	}
	if lowest.ID != 0 {
		view.Comparable = true
		view.Purchasable = true
		view.Current = lowest.UnitPrice
	}
	if row.Status == constants.AlertStatusTriggered {
		if row.TriggeredAt != nil {
			view.TriggeredAt = row.TriggeredAt.Format(time.RFC3339)
		}
		if row.TriggeredOffer != nil {
			view.Current = row.TriggeredOffer.UnitPrice
			view.Supplier = row.TriggeredOffer.Supplier.Name
			// The triggering quote stays purchasable only while its shop
			// remains approved and the quote itself stays in stock.
			view.Purchasable = row.TriggeredOffer.StockStatus == constants.StatusInStock &&
				row.TriggeredOffer.Supplier.Status == constants.SupplierApproved
			view.Comparable = true
		}
	}
	return view
}

// lowestValidOffer selects the cheapest offer whose supplier is approved
// and whose stock status is in_stock. nil when nothing qualifies.
func lowestValidOffer(offers []model.Offer) *model.Offer {
	var best *model.Offer
	for i := range offers {
		offer := &offers[i]
		if offer.Supplier.Status != constants.SupplierApproved || offer.StockStatus != constants.StatusInStock {
			continue
		}
		if best == nil || offer.UnitPrice < best.UnitPrice {
			best = offer
		}
	}
	return best
}

// DropPercent computes the drop from baseline to price, rounded to 1 decimal.
func dropPercent(baseline, price float64) float64 {
	if baseline <= 0 {
		return 0
	}
	return math.Round((baseline-price)/baseline*1000) / 10
}
