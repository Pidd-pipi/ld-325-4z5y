package handler

import (
	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// DemoAuthHandler hands out short-lived tokens for local role demos
// (e.g. submitting supplier quotes from the browser without a login flow).
type DemoAuthHandler struct{ secret string }

func NewDemoAuthHandler(secret string) *DemoAuthHandler { return &DemoAuthHandler{secret} }

func (h *DemoAuthHandler) SupplierToken(c *gin.Context) {
	token, err := middleware.NewDemoToken(h.secret, "demo-supplier", constants.RoleSupplier)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, gin.H{"token": token, "role": constants.RoleSupplier})
}
