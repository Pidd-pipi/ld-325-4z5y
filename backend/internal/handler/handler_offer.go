package handler

import (
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	"github.com/blueship581/cybuildprice/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type OfferHandler struct {
	service  *service.OfferService
	validate *validator.Validate
}

func NewOfferHandler(s *service.OfferService, v *validator.Validate) *OfferHandler {
	return &OfferHandler{s, v}
}
func (h *OfferHandler) List(c *gin.Context) {
	var path struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(err)
		return
	}
	data, err := h.service.List(path.ID)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}
func (h *OfferHandler) UpdateStatus(c *gin.Context) {
	var path struct {
		ID uint `uri:"id" binding:"required"`
	}
	var req dto.UpdateOfferStatusRequest
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(err)
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	data, err := h.service.UpdateStatus(path.ID, req.StockStatus)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}

// Submit is the supplier price-change entry point that drives alert
// evaluation for the submitted offer version.
func (h *OfferHandler) Submit(c *gin.Context) {
	var req dto.SubmitOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	offer, events, err := h.service.Submit(req)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, gin.H{"offer": offer, "triggered": len(events)})
}
