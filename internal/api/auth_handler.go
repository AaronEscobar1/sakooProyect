package api

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/aaron/sakoo-backend/internal/infrastructure/security"
)

// PublicKeyResponse representa la clave pública RSA devuelta.
type PublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}

// ProfileResponse representa los datos de perfil de usuario seguros.
type ProfileResponse struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	Username       string  `json:"username"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	MiddleName     *string `json:"middle_name,omitempty"`
	SecondLastName *string `json:"second_last_name,omitempty"`
	AvatarIndex    int     `json:"avatar_index"`
	UserTypeID     int64   `json:"user_type_id"`
	DocumentTypeID *int64  `json:"document_type_id,omitempty"`
	DocumentNumber *string `json:"document_number,omitempty"`
	RegistrationIP *string `json:"registration_ip,omitempty"`
	Country        *string `json:"country,omitempty"`
}

// RequestOTPRequest representa el cuerpo para solicitar un OTP.
type RequestOTPRequest struct {
	Email  string `json:"email"`
	Action string `json:"action"` // 'REGISTER', 'RECOVER', 'DELETE'
}

// ResetPasswordRequest representa el cuerpo para restablecer la contraseña.
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	NewPassword string `json:"new_password"`
	OTPCode     string `json:"otp_code"`
}

// DeleteAccountRequest representa la confirmación de eliminación con OTP.
type DeleteAccountRequest struct {
	OTPCode string `json:"otp_code"`
}

// DeleteLegacyAccountRequest representa la confirmación de eliminación de cuenta heredada.
type DeleteLegacyAccountRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthHandler expone los controladores HTTP del módulo de autenticación y seguridad.
type AuthHandler struct {
	authUseCase domain.AuthUseCase
}

// NewAuthHandler crea una instancia del controlador AuthHandler.
func NewAuthHandler(authUseCase domain.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

// HandlePublicKey expone la clave pública RSA en formato PEM para que el frontend la consuma.
// @Summary      Obtener clave pública RSA
// @Description  Retorna la clave pública RSA en formato PEM para cifrar credenciales de usuario antes de transmitirlas.
// @Tags         Autenticación
// @Produce      json
// @Success      200  {object}  response.APIResponse[PublicKeyResponse]  "Clave pública obtenida correctamente"
// @Router       /api/auth/public-key [get]
func (h *AuthHandler) HandlePublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	pubKey, err := security.GetPublicKeyPEM()
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "error al recuperar la clave pública de tránsito")
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Clave pública RSA obtenida correctamente para cifrado de credenciales", map[string]string{
		"public_key": pubKey,
	})
}

// HandleRequestOTP maneja la petición POST /api/v1/auth/otp/request para generar y enviar un OTP.
// @Summary      Solicitar OTP
// @Description  Genera y envía un código OTP por correo electrónico al usuario para registrarse, recuperar contraseña o eliminar cuenta.
// @Tags         Autenticación
// @Accept       json
// @Produce      json
// @Param        body  body  RequestOTPRequest  true  "Cuerpo de la solicitud con email y acción"
// @Success      200   {object}  response.APIResponse[any]  "Código OTP generado y enviado exitosamente"
// @Failure      400   {object}  response.APIResponse[any]  "Solicitud incorrecta o error de negocio"
// @Router       /api/v1/auth/otp/request [post]
func (h *AuthHandler) HandleRequestOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	var req RequestOTPRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "formato de cuerpo JSON inválido")
		return
	}

	if req.Email == "" || req.Action == "" {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "el correo electrónico y la acción son campos requeridos")
		return
	}

	err := h.authUseCase.RequestOTP(r.Context(), req.Email, req.Action)
	if err != nil {
		slog.Error("Fallo al procesar solicitud de OTP", "error", err, "email", req.Email, "action", req.Action)
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Código de seguridad OTP generado y enviado exitosamente", nil)
}

// HandleRegister maneja la petición POST /api/v1/auth/register para registrar usuarios exigiendo OTP y contraseñas cifradas en tránsito.
// @Summary      Registrar un nuevo usuario
// @Description  Registra un nuevo usuario en la plataforma validando el código OTP y descifrando la contraseña.
// @Tags         Autenticación
// @Accept       json
// @Produce      json
// @Param        body  body  domain.RegisterRequest  true  "Datos de registro"
// @Success      200   {object}  response.APIResponse[any]  "Usuario registrado exitosamente"
// @Failure      200   {object}  response.APIResponse[any]  "Error al registrar usuario o datos duplicados"
// @Router       /api/v1/auth/register [post]
// @Router       /api/auth/register [post]
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "formato de cuerpo JSON inválido")
		return
	}

	// Extraer la dirección IP real del cliente de forma resiliente
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP != "" {
		ips := strings.Split(clientIP, ",")
		clientIP = strings.TrimSpace(ips[0])
	}
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			clientIP = host
		} else {
			clientIP = r.RemoteAddr
		}
	}
	req.RegistrationIP = clientIP

	// 1. Descifrar la contraseña cifrada en RSA que viene en formato Base64 desde el cliente
	decryptedPassword, err := security.DecryptPassword(req.Password)
	if err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "las credenciales enviadas no tienen el formato de seguridad esperado")
		return
	}
	req.Password = decryptedPassword

	// 2. Invocar lógica de negocio (registro de usuario exigiendo y consumiendo OTP)
	err = h.authUseCase.Register(r.Context(), req)
	if err != nil {
		slog.Error("Fallo al registrar usuario con OTP", "error", err, "email", req.Email)
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "users_email_key") || strings.Contains(err.Error(), "users_username_key") {
			if strings.Contains(err.Error(), "users_username_key") {
				response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "el nombre de usuario ingresado ya se encuentra registrado")
				return
			}
			response.Error(w, r.Context(), http.StatusOK, "USER_ALREADY_EXISTS", "el correo electrónico ingresado ya se encuentra registrado")
			return
		}
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "CREATED", "usuario registrado y verificado exitosamente en el sistema", nil)
}

// HandleLogin maneja la petición POST /api/auth/login para autenticar usuarios con contraseñas cifradas en tránsito.
// @Summary      Iniciar sesión de usuario
// @Description  Autentica a un usuario mediante su correo electrónico y contraseña (cifrada con RSA en tránsito) y devuelve un token JWT.
// @Tags         Autenticación
// @Accept       json
// @Produce      json
// @Param        body  body  domain.LoginRequest  true  "Credenciales de acceso"
// @Success      200   {object}  response.APIResponse[domain.AuthResponse]  "Sesión iniciada correctamente"
// @Failure      401   {object}  response.APIResponse[any]                 "Credenciales incorrectas"
// @Router       /api/auth/login [post]
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "formato de cuerpo JSON inválido")
		return
	}

	// 1. Descifrar la contraseña cifrada en RSA que viene en formato Base64 desde el cliente
	decryptedPassword, err := security.DecryptPassword(req.Password)
	if err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "las credenciales enviadas no tienen el formato de seguridad esperado")
		return
	}
	req.Password = decryptedPassword

	// 2. Autenticar e iniciar sesión
	res, err := h.authUseCase.Login(r.Context(), req)
	if err != nil {
		response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "correo electrónico o contraseña incorrectos")
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "sesión iniciada correctamente", res)
}

// HandleResetPassword maneja la petición POST /api/v1/auth/password/reset para restablecer contraseña exigiendo OTP.
// @Summary      Restablecer contraseña
// @Description  Restablece la contraseña de un usuario validando el código OTP enviado a su correo.
// @Tags         Autenticación
// @Accept       json
// @Produce      json
// @Param        body  body  ResetPasswordRequest  true  "Datos para restablecer la contraseña"
// @Success      200   {object}  response.APIResponse[any]  "Contraseña restablecida correctamente"
// @Failure      400   {object}  response.APIResponse[any]  "Código OTP inválido o credenciales inválidas"
// @Router       /api/v1/auth/password/reset [post]
func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	var req ResetPasswordRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "formato de cuerpo JSON inválido")
		return
	}

	if req.Email == "" || req.NewPassword == "" || req.OTPCode == "" {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "los campos email, new_password y otp_code son requeridos")
		return
	}

	// 1. Descifrar la nueva contraseña cifrada en RSA
	decryptedPassword, err := security.DecryptPassword(req.NewPassword)
	if err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "la contraseña enviada no tiene el formato de seguridad esperado")
		return
	}
	req.NewPassword = decryptedPassword

	// 2. Invocar lógica de negocio
	err = h.authUseCase.ResetPassword(r.Context(), req.Email, req.NewPassword, req.OTPCode)
	if err != nil {
		slog.Error("Fallo al restablecer contraseña con OTP", "error", err, "email", req.Email)
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Contraseña restablecida correctamente", nil)
}

// HandleDeleteAccount maneja la petición DELETE /api/auth/me para realizar el borrado lógico heredado (Legacy).
// @Summary      Eliminar cuenta (Legacy)
// @Description  Elimina de forma lógica la cuenta del usuario autenticado validando su correo electrónico y contraseña (cifrada en tránsito).
// @Security     ApiKeyAuth
// @Tags         Autenticación
// @Accept       json
// @Produce      json
// @Param        body  body  DeleteLegacyAccountRequest  true  "Credenciales de confirmación de eliminación"
// @Success      200   {object}  response.APIResponse[any]  "Cuenta eliminada lógicamente de manera exitosa"
// @Failure      401   {object}  response.APIResponse[any]  "No autorizado"
// @Failure      500   {object}  response.APIResponse[any]  "Error interno del servidor"
// @Router       /api/auth/me [delete]
func (h *AuthHandler) HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere DELETE)")
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "autorización denegada: no se pudo recuperar el ID del usuario")
		return
	}

	var req DeleteLegacyAccountRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "formato de cuerpo JSON inválido")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "el correo electrónico y la contraseña son requeridos")
		return
	}

	decryptedPassword, err := security.DecryptPassword(req.Password)
	if err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "las credenciales enviadas no tienen el formato de seguridad esperado")
		return
	}
	req.Password = decryptedPassword

	err = h.authUseCase.DeleteMyAccount(r.Context(), userID, req.Email, req.Password)
	if err != nil {
		if err.Error() == "credenciales incorrectas" || err.Error() == "no estás autorizado para eliminar esta cuenta" {
			response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
			return
		}
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "error al realizar el borrado lógico de la cuenta")
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "cuenta desactivada y eliminada lógicamente de manera correcta", nil)
}

// HandleDeleteAccountV1 maneja la petición DELETE /api/v1/account protegida por JWT para realizar el borrado lógico con OTP.
// @Summary      Eliminar cuenta (v1)
// @Description  Elimina de forma lógica la cuenta del usuario autenticado validando un código OTP.
// @Security     ApiKeyAuth
// @Tags         Autenticación
// @Accept       json
// @Produce      json
// @Param        body  body  DeleteAccountRequest  true  "Confirmación de eliminación con código OTP"
// @Success      200   {object}  response.APIResponse[any]  "Cuenta eliminada lógicamente de manera exitosa"
// @Failure      401   {object}  response.APIResponse[any]  "No autorizado"
// @Failure      400   {object}  response.APIResponse[any]  "Código OTP inválido o error"
// @Router       /api/v1/account [delete]
func (h *AuthHandler) HandleDeleteAccountV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere DELETE)")
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "autorización denegada: no se pudo recuperar el ID del usuario")
		return
	}

	var req DeleteAccountRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "formato de cuerpo JSON inválido")
		return
	}

	if req.OTPCode == "" {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "el código OTP es requerido para confirmar la eliminación de la cuenta")
		return
	}

	err := h.authUseCase.DeleteAccount(r.Context(), userID, req.OTPCode)
	if err != nil {
		slog.Error("Fallo al eliminar lógicamente la cuenta con OTP", "error", err, "user_id", userID)
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "cuenta eliminada lógicamente de manera exitosa", nil)
}

// HandleGetProfile maneja la petición GET /api/v1/me para obtener el perfil completo del usuario autenticado.
// @Summary      Obtener perfil del usuario
// @Description  Retorna los datos de perfil del usuario actualmente autenticado (mediante JWT).
// @Security     ApiKeyAuth
// @Tags         Autenticación
// @Produce      json
// @Success      200   {object}  response.APIResponse[ProfileResponse]  "Perfil del usuario recuperado exitosamente"
// @Failure      401   {object}  response.APIResponse[any]              "No autorizado"
// @Router       /api/v1/me [get]
func (h *AuthHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	// Extraer el userID del contexto (inyectado por AuthMiddleware)
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "autorización denegada: no se pudo recuperar el ID del usuario")
		return
	}

	user, err := h.authUseCase.GetProfile(r.Context(), userID)
	if err != nil {
		slog.Error("Fallo al obtener perfil del usuario", "error", err, "user_id", userID)
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", "error al recuperar el perfil del usuario")
		return
	}

	// Mapear entidad de dominio a un mapa JSON limpio para el cliente, evitando exponer campos sensibles
	profile := map[string]interface{}{
		"id":               user.ID,
		"email":            user.Email,
		"username":         user.Username,
		"first_name":       user.FirstName,
		"last_name":        user.LastName,
		"middle_name":      user.MiddleName,
		"second_last_name": user.SecondLastName,
		"avatar_index":     user.AvatarIndex,
		"user_type_id":     user.UserTypeID,
		"document_type_id": user.DocumentTypeID,
		"document_number":  user.DocumentNumber,
		"registration_ip":  user.RegistrationIP,
		"country":          user.Country,
	}

	response.Success(w, r.Context(), "SUCCESS", "perfil del usuario recuperado exitosamente", profile)
}
