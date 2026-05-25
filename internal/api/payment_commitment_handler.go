package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// CommitmentRequest define los parámetros de entrada para crear/actualizar un compromiso de pago.
type CommitmentRequest struct {
	Amount     decimal.Decimal `json:"amount"`
	CurrencyID int64           `json:"currency_id"`
	DueDateStr string          `json:"due_date"` // YYYY-MM-DD
	Status     string          `json:"status"`
	Concept    string          `json:"concept"`
}

// CommitmentResponse representa el DTO de un compromiso de pago devuelto.
type CommitmentResponse struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Amount     string `json:"amount"`
	CurrencyID int64  `json:"currency_id"`
	DueDate    string `json:"due_date"`
	Status     string `json:"status"`
	Concept    string `json:"concept"`
	CreatedAt  string `json:"created_at"`
}

// SegmentedCommitmentResponse define la estructura del mapa segmentado de compromisos.
type SegmentedCommitmentResponse struct {
	PorVencer []CommitmentResponse `json:"por_vencer"`
	Vencidos  []CommitmentResponse `json:"vencidos"`
	Cumplidos []CommitmentResponse `json:"cumplidos"`
}

type PaymentCommitmentHandler struct {
	useCase domain.PaymentCommitmentUseCase
}

func NewPaymentCommitmentHandler(useCase domain.PaymentCommitmentUseCase) *PaymentCommitmentHandler {
	return &PaymentCommitmentHandler{
		useCase: useCase,
	}
}

// HandleCommitments maneja POST para crear y GET para listar de forma segmentada en /api/v1/payments/commitments
// @Summary      Gestionar compromisos de pago (Crear / Listar Segmentados)
// @Description  Permite registrar un nuevo compromiso de pago (POST) o listar todos los compromisos segmentados por estado (por_vencer, vencidos, cumplidos) del usuario (GET).
// @Security     ApiKeyAuth
// @Tags         Compromisos de Pago
// @Accept       json
// @Produce      json
// @Param        body  body  CommitmentRequest  false  "Datos del compromiso (requerido solo para POST)"
// @Success      200  {object}  response.APIResponse[SegmentedCommitmentResponse]  "Operación realizada con éxito (devuelve compromiso creado o mapa segmentado)"
// @Failure      200  {object}  response.APIResponse[any]                          "Error al procesar la solicitud"
// @Router       /api/v1/payments/commitments [post]
// @Router       /api/v1/payments/commitments [get]
func (h *PaymentCommitmentHandler) HandleCommitments(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "Autorización denegada: no se pudo recuperar el ID del usuario")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req CommitmentRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "Formato de cuerpo JSON inválido")
			return
		}

		parsedDate, err := time.Parse("2006-01-02", req.DueDateStr)
		if err != nil {
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "Formato de fecha límite 'due_date' inválido (debe ser YYYY-MM-DD)")
			return
		}

		pc, err := h.useCase.Create(r.Context(), userID, req.Amount, req.CurrencyID, parsedDate, req.Status, req.Concept)
		if err != nil {
			slog.Error("Fallo al crear compromiso de pago", "error", err, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		res := CommitmentResponse{
			ID:         pc.ID,
			UserID:     pc.UserID,
			Amount:     pc.Amount.String(),
			CurrencyID: pc.CurrencyID,
			DueDate:    pc.DueDate.Format("2006-01-02"),
			Status:     pc.Status,
			Concept:    pc.Concept,
			CreatedAt:  pc.CreatedAt.Format(time.RFC3339),
		}

		response.Success(w, r.Context(), "CREATED", "Compromiso de pago creado exitosamente", res)

	case http.MethodGet:
		segmented, err := h.useCase.GetSegmentedCommitments(r.Context(), userID)
		if err != nil {
			slog.Error("Fallo al obtener compromisos segmentados", "error", err, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", "Error al recuperar compromisos de pago")
			return
		}

		dto := SegmentedCommitmentResponse{
			PorVencer: []CommitmentResponse{},
			Vencidos:  []CommitmentResponse{},
			Cumplidos: []CommitmentResponse{},
		}

		if list, found := segmented["por_vencer"]; found {
			for _, pc := range list {
				dto.PorVencer = append(dto.PorVencer, CommitmentResponse{
					ID:         pc.ID,
					UserID:     pc.UserID,
					Amount:     pc.Amount.String(),
					CurrencyID: pc.CurrencyID,
					DueDate:    pc.DueDate.Format("2006-01-02"),
					Status:     pc.Status,
					Concept:    pc.Concept,
					CreatedAt:  pc.CreatedAt.Format(time.RFC3339),
				})
			}
		}

		if list, found := segmented["vencidos"]; found {
			for _, pc := range list {
				dto.Vencidos = append(dto.Vencidos, CommitmentResponse{
					ID:         pc.ID,
					UserID:     pc.UserID,
					Amount:     pc.Amount.String(),
					CurrencyID: pc.CurrencyID,
					DueDate:    pc.DueDate.Format("2006-01-02"),
					Status:     pc.Status,
					Concept:    pc.Concept,
					CreatedAt:  pc.CreatedAt.Format(time.RFC3339),
				})
			}
		}

		if list, found := segmented["cumplidos"]; found {
			for _, pc := range list {
				dto.Cumplidos = append(dto.Cumplidos, CommitmentResponse{
					ID:         pc.ID,
					UserID:     pc.UserID,
					Amount:     pc.Amount.String(),
					CurrencyID: pc.CurrencyID,
					DueDate:    pc.DueDate.Format("2006-01-02"),
					Status:     pc.Status,
					Concept:    pc.Concept,
					CreatedAt:  pc.CreatedAt.Format(time.RFC3339),
				})
			}
		}

		response.Success(w, r.Context(), "SUCCESS", "Compromisos de pago segmentados obtenidos exitosamente", dto)

	default:
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido")
	}
}

// HandleCommitmentDetail maneja PUT y DELETE para /api/v1/payments/commitments/{id}
// @Summary      Gestionar detalle de compromiso de pago (Actualizar / Eliminar)
// @Description  Permite actualizar por completo un compromiso de pago (PUT) o eliminarlo de forma lógica (DELETE) especificando su ID en la ruta.
// @Security     ApiKeyAuth
// @Tags         Compromisos de Pago
// @Accept       json
// @Produce      json
// @Param        id    path  int64              true   "ID del compromiso de pago"
// @Param        body  body  CommitmentRequest  false  "Datos actualizados del compromiso (requerido solo para PUT)"
// @Success      200  {object}  response.APIResponse[CommitmentResponse]  "Operación realizada con éxito"
// @Failure      200  {object}  response.APIResponse[any]                 "ID inválido, no autorizado o error de negocio"
// @Router       /api/v1/payments/commitments/{id} [put]
// @Router       /api/v1/payments/commitments/{id} [delete]
func (h *PaymentCommitmentHandler) HandleCommitmentDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "Autorización denegada: no se pudo recuperar el ID del usuario")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "ID de compromiso de pago inválido o ausente")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req CommitmentRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "Formato de cuerpo JSON inválido")
			return
		}

		parsedDate, err := time.Parse("2006-01-02", req.DueDateStr)
		if err != nil {
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "Formato de fecha límite 'due_date' inválido (debe ser YYYY-MM-DD)")
			return
		}

		pc, err := h.useCase.Update(r.Context(), id, userID, req.Amount, req.CurrencyID, parsedDate, req.Status, req.Concept)
		if err != nil {
			slog.Error("Fallo al actualizar compromiso de pago", "error", err, "id", id, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		res := CommitmentResponse{
			ID:         pc.ID,
			UserID:     pc.UserID,
			Amount:     pc.Amount.String(),
			CurrencyID: pc.CurrencyID,
			DueDate:    pc.DueDate.Format("2006-01-02"),
			Status:     pc.Status,
			Concept:    pc.Concept,
			CreatedAt:  pc.CreatedAt.Format(time.RFC3339),
		}

		response.Success(w, r.Context(), "SUCCESS", "Compromiso de pago actualizado exitosamente", res)

	case http.MethodDelete:
		err := h.useCase.Delete(r.Context(), id, userID)
		if err != nil {
			slog.Error("Fallo al eliminar compromiso de pago", "error", err, "id", id, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		response.Success(w, r.Context(), "SUCCESS", "Compromiso de pago eliminado exitosamente", nil)

	default:
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido")
	}
}
