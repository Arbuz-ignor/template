package get

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/Arbuz-ignor/template/internal/http/domainerror"
	"github.com/Arbuz-ignor/template/internal/http/response"
)

type Handler struct{ usecase usecase }

func New(u usecase) *Handler { return &Handler{usecase: u} }
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	result, err := h.usecase.Execute(r.Context(), id)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, response.Trip(result))
}
