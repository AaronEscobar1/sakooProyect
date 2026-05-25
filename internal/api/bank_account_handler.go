package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/domain"
)

// OwnAccountRequest define los campos requeridos para crear/actualizar una cuenta bancaria propia.
type OwnAccountRequest struct {
	BankID        int64  `json:"bank_id"`
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	HolderName    string `json:"holder_name"`
}

// ThirdPartyAccountRequest define los campos requeridos para crear/actualizar una cuenta bancaria de terceros.
type ThirdPartyAccountRequest struct {
	BankID         int64  `json:"bank_id"`
	AccountNumber  string `json:"account_number"`
	AccountType    string `json:"account_type"`
	HolderName     string `json:"holder_name"`
	Alias          string `json:"alias"`
	DocumentNumber string `json:"document_number"`
	PhoneNumber    string `json:"phone_number"`
}

type BankAccountHandler struct {
	useCase domain.BankAccountUseCase
}

func NewBankAccountHandler(useCase domain.BankAccountUseCase) *BankAccountHandler {
	return &BankAccountHandler{
		useCase: useCase,
	}
}

// HandleOwnAccounts maneja POST y GET para /api/v1/accounts/own
// @Summary      Gestionar cuentas bancarias propias (Crear / Listar)
// @Description  Permite registrar una nueva cuenta bancaria propia (POST) o listar todas las cuentas propias asociadas al usuario autenticado (GET).
// @Security     ApiKeyAuth
// @Tags         Cuentas Bancarias - Propias
// @Accept       json
// @Produce      json
// @Param        body  body  OwnAccountRequest  false  "Datos de la cuenta (requerido solo para POST)"
// @Success      200  {object}  response.APIResponse[domain.BankAccount]  "Operación realizada con éxito (devuelve la cuenta creada o la lista de cuentas)"
// @Failure      200  {object}  response.APIResponse[any]                 "Error de autenticación o datos incorrectos"
// @Router       /api/v1/accounts/own [post]
// @Router       /api/v1/accounts/own [get]
func (h *BankAccountHandler) HandleOwnAccounts(w http.ResponseWriter, r *http.Request) {
	// Extraer el userID del contexto (inyectado por AuthMiddleware)
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "Autorización denegada: no se pudo recuperar el ID del usuario")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req OwnAccountRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "Formato de cuerpo JSON inválido")
			return
		}

		acc, err := h.useCase.CreateOwn(r.Context(), userID, req.BankID, req.AccountNumber, req.AccountType, req.HolderName)
		if err != nil {
			slog.Error("Fallo al crear cuenta propia", "error", err, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		response.Success(w, r.Context(), "CREATED", "Cuenta propia creada exitosamente", acc)

	case http.MethodGet:
		accounts, err := h.useCase.ListOwn(r.Context(), userID)
		if err != nil {
			slog.Error("Fallo al listar cuentas propias", "error", err, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", "Error al listar cuentas bancarias")
			return
		}

		response.Success(w, r.Context(), "SUCCESS", "Cuentas propias obtenidas exitosamente", accounts)

	default:
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido")
	}
}

// HandleOwnAccountDetail maneja PUT y DELETE para /api/v1/accounts/own/{id}
// @Summary      Gestionar detalle de cuenta propia (Actualizar / Eliminar)
// @Description  Permite actualizar por completo una cuenta bancaria propia (PUT) o eliminarla de forma lógica (DELETE) especificando su ID en la ruta.
// @Security     ApiKeyAuth
// @Tags         Cuentas Bancarias - Propias
// @Accept       json
// @Produce      json
// @Param        id    path  int64              true   "ID de la cuenta bancaria"
// @Param        body  body  OwnAccountRequest  false  "Datos actualizados de la cuenta (requerido solo para PUT)"
// @Success      200  {object}  response.APIResponse[domain.BankAccount]  "Operación realizada con éxito"
// @Failure      200  {object}  response.APIResponse[any]                 "ID inválido, no autorizado o error de negocio"
// @Router       /api/v1/accounts/own/{id} [put]
// @Router       /api/v1/accounts/own/{id} [delete]
func (h *BankAccountHandler) HandleOwnAccountDetail(w http.ResponseWriter, r *http.Request) {
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
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "ID de cuenta inválido o ausente")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req OwnAccountRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "Formato de cuerpo JSON inválido")
			return
		}

		acc, err := h.useCase.UpdateOwn(r.Context(), id, userID, req.BankID, req.AccountNumber, req.AccountType, req.HolderName)
		if err != nil {
			slog.Error("Fallo al actualizar cuenta propia", "error", err, "id", id, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		response.Success(w, r.Context(), "SUCCESS", "Cuenta propia actualizada exitosamente", acc)

	case http.MethodDelete:
		err := h.useCase.DeleteOwn(r.Context(), id, userID)
		if err != nil {
			slog.Error("Fallo al eliminar cuenta propia", "error", err, "id", id, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		response.Success(w, r.Context(), "SUCCESS", "Cuenta propia eliminada exitosamente", nil)

	default:
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido")
	}
}

// HandleThirdPartyAccounts maneja POST y GET para /api/v1/accounts/third-party
// @Summary      Gestionar cuentas bancarias de terceros (Crear / Listar)
// @Description  Permite registrar una nueva cuenta bancaria de un tercero (POST) o listar todas las de terceros asociadas al usuario autenticado (GET).
// @Security     ApiKeyAuth
// @Tags         Cuentas Bancarias - Terceros
// @Accept       json
// @Produce      json
// @Param        body  body  ThirdPartyAccountRequest  false  "Datos de la cuenta (requerido solo para POST)"
// @Success      200  {object}  response.APIResponse[domain.BankAccount]  "Operación realizada con éxito"
// @Failure      200  {object}  response.APIResponse[any]                 "Error al procesar la solicitud"
// @Router       /api/v1/accounts/third-party [post]
// @Router       /api/v1/accounts/third-party [get]
func (h *BankAccountHandler) HandleThirdPartyAccounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "Autorización denegada: no se pudo recuperar el ID del usuario")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req ThirdPartyAccountRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "Formato de cuerpo JSON inválido")
			return
		}

		acc, err := h.useCase.CreateThirdParty(r.Context(), userID, req.BankID, req.AccountNumber, req.AccountType, req.HolderName, req.Alias, req.DocumentNumber, req.PhoneNumber)
		if err != nil {
			slog.Error("Fallo al crear cuenta de terceros", "error", err, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		response.Success(w, r.Context(), "CREATED", "Cuenta de terceros creada exitosamente", acc)

	case http.MethodGet:
		accounts, err := h.useCase.ListThirdParty(r.Context(), userID)
		if err != nil {
			slog.Error("Fallo al listar cuentas de terceros", "error", err, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", "Error al listar cuentas bancarias de terceros")
			return
		}

		response.Success(w, r.Context(), "SUCCESS", "Cuentas de terceros obtenidas exitosamente", accounts)

	default:
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido")
	}
}

// HandleThirdPartyAccountDetail maneja PUT y DELETE para /api/v1/accounts/third-party/{id}
// @Summary      Gestionar detalle de cuenta de terceros (Actualizar / Eliminar)
// @Description  Permite actualizar por completo una cuenta de terceros (PUT) o eliminarla físicamente (DELETE) especificando su ID en la ruta.
// @Security     ApiKeyAuth
// @Tags         Cuentas Bancarias - Terceros
// @Accept       json
// @Produce      json
// @Param        id    path  int64                     true   "ID de la cuenta de terceros"
// @Param        body  body  ThirdPartyAccountRequest  false  "Datos actualizados de la cuenta (requerido solo para PUT)"
// @Success      200  {object}  response.APIResponse[domain.BankAccount]  "Operación realizada con éxito"
// @Failure      200  {object}  response.APIResponse[any]                 "ID inválido, no autorizado o error de negocio"
// @Router       /api/v1/accounts/third-party/{id} [put]
// @Router       /api/v1/accounts/third-party/{id} [delete]
func (h *BankAccountHandler) HandleThirdPartyAccountDetail(w http.ResponseWriter, r *http.Request) {
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
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "ID de cuenta de terceros inválido o ausente")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req ThirdPartyAccountRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "Formato de cuerpo JSON inválido")
			return
		}

		acc, err := h.useCase.UpdateThirdParty(r.Context(), id, userID, req.BankID, req.AccountNumber, req.AccountType, req.HolderName, req.Alias, req.DocumentNumber, req.PhoneNumber)
		if err != nil {
			slog.Error("Fallo al actualizar cuenta de terceros", "error", err, "id", id, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		response.Success(w, r.Context(), "SUCCESS", "Cuenta de terceros actualizada exitosamente", acc)

	case http.MethodDelete:
		err := h.useCase.DeleteThirdParty(r.Context(), id, userID)
		if err != nil {
			slog.Error("Fallo al eliminar cuenta de terceros", "error", err, "id", id, "user_id", userID)
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
			return
		}

		response.Success(w, r.Context(), "SUCCESS", "Cuenta de terceros eliminada exitosamente", nil)

	default:
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido")
	}
}
