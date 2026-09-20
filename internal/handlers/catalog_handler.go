package handlers

import (
	"accesspath/internal/services"
	"accesspath/pkg/response"

	"github.com/gin-gonic/gin"
)

type CatalogHandler struct {
	service *services.CatalogService
}

func NewCatalogHandler(service *services.CatalogService) *CatalogHandler {
	return &CatalogHandler{service: service}
}

// GetDimensions godoc
// @Summary      Catalogo del formulario
// @Description  Dimensiones con sus criterios, opciones y depends_on. Reemplaza /categories.
// @Tags         catalog
// @Produce      json
// @Success      200  {object}  map[string][]models.DimensionDetail
// @Failure      500  {object}  map[string]string
// @Router       /dimensions [get]
func (h *CatalogHandler) GetDimensions(c *gin.Context) {
	catalog, err := h.service.GetCatalog(c.Request.Context())
	if Respond(c, err) {
		return
	}
	response.OK(c, catalog)
}