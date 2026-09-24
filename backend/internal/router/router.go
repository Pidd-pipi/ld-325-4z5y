package router

import (
	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/handler"
	"github.com/blueship581/cybuildprice/backend/internal/middleware"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
	"github.com/blueship581/cybuildprice/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"log/slog"
)

func New(db *gorm.DB, logger *slog.Logger, jwtSecret string) *gin.Engine {
	v := validator.New()
	productRepo := repository.NewProductRepository(db)
	offerRepo := repository.NewOfferRepository(db)
	supplierRepo := repository.NewSupplierRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	product := handler.NewProductHandler(service.NewProductService(productRepo, logger), v)
	offers := handler.NewOfferHandler(service.NewOfferService(offerRepo, supplierRepo, productRepo, alertRepo), v)
	trend := handler.NewTrendHandler(service.NewPriceHistoryService(repository.NewPriceHistoryRepository(db)))
	user := handler.NewUserDataHandler(service.NewUserDataService(repository.NewUserDataRepository(db)), v)
	alerts := handler.NewAlertHandler(service.NewAlertService(alertRepo, offerRepo, productRepo), v)
	supplier := handler.NewSupplierHandler(service.NewSupplierService(supplierRepo), v)
	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.RequestLogger(logger), middleware.ErrorHandler(), middleware.JWTOrDemoAuth(jwtSecret))
	r.GET(constants.HealthPath, func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	api := r.Group(constants.APIPrefix)
	api.GET("/products", product.List)
	api.GET("/products/:id", product.Get)
	api.POST("/products/compare", product.Compare)
	api.GET("/products/:id/offers", offers.List)
	api.GET("/products/:id/trend", trend.Get)
	api.GET("/suppliers", supplier.List)
	api.PATCH("/admin/suppliers/:id/status", middleware.RequireRole(constants.RoleAdmin), supplier.UpdateStatus)
	api.POST("/supplier/offers", middleware.RequireRole(constants.RoleSupplier, constants.RoleAdmin), offers.Submit)
	api.PATCH("/supplier/offers/:id/status", middleware.RequireRole(constants.RoleSupplier, constants.RoleAdmin), offers.UpdateStatus)
	api.GET("/favorites", user.ListFavorites)
	api.POST("/favorites", user.CreateFavorite)
	api.GET("/alerts", alerts.List)
	api.POST("/alerts", alerts.Create)
	api.POST("/budgets", user.CreateBudget)
	return r
}
