package handler

import (
	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	"github.com/blueship581/cybuildprice/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AlertHandler struct {
	service  *service.AlertService
	validate *validator.Validate
}

func NewAlertHandler(s *service.AlertService, v *validator.Validate) *AlertHandler {
	return &AlertHandler{s, v}
}

func (h *AlertHandler) Create(c *gin.Context) {
	var req dto.CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	row, err := h.service.Subscribe(c.GetString(constants.UserIDContextKey), req)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, row)
}

// List returns ongoing and triggered subscriptions for the personal center.
func (h *AlertHandler) List(c *gin.Context) {
	data, err := h.service.List(c.GetString(constants.UserIDContextKey))
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}
