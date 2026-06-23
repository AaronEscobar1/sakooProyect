package usecase

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	commonMiddleware "github.com/AaronEscobar1/common/middleware"
	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// defaultCountry es el valor del campo país del perfil cuando no se puede geolocalizar la IP
// (IP local/privada en dev, IP inválida o servicio GeoIP no disponible). NO restringe el acceso.
const defaultCountry = "Desconocido"

type authUseCase struct {
	userRepo  domain.UserRepository
	otpRepo   domain.OTPRepository
	emailSrv  domain.EmailService
	notifRepo domain.NotificationRepository
	jwtSecret string
}

// NewAuthUseCase crea una instancia concreta del caso de uso de autenticación con todas sus dependencias.
func NewAuthUseCase(
	userRepo domain.UserRepository,
	otpRepo domain.OTPRepository,
	emailSrv domain.EmailService,
	notifRepo domain.NotificationRepository,
	jwtSecret string,
) domain.AuthUseCase {
	return &authUseCase{
		userRepo:  userRepo,
		otpRepo:   otpRepo,
		emailSrv:  emailSrv,
		notifRepo: notifRepo,
		jwtSecret: jwtSecret,
	}
}

// generateNumericOTP genera un código OTP numérico seguro y aleatorio de 6 dígitos.
func generateNumericOTP() (string, error) {
	code := ""
	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += num.String()
	}
	return code, nil
}

// RequestOTP genera un OTP, lo persiste con 5 minutos de vigencia, y lo envía por email.
// SEGURIDAD: el código nunca se retorna ni se expone fuera del envío de correo; el único
// canal de entrega al usuario es el email (Resend/SMTP).
func (s *authUseCase) RequestOTP(ctx context.Context, email, action, username, documentNumber string) error {
	slog.Info("Procesando solicitud de OTP", "email", email, "action", action)

	if email == "" || action == "" {
		return errors.New("El correo electrónico y la acción son requeridos")
	}

	if action != "REGISTER" && action != "RECOVER" && action != "DELETE" {
		return fmt.Errorf("Acción de OTP inválida: %s", action)
	}

	// Pre-validación de unicidad para REGISTER: si el correo, usuario o cédula ya existen,
	// se rechaza ANTES de generar/enviar el OTP. Así no se desperdicia un código ni se activa
	// el throttle de 60s, y el cliente puede devolver al usuario al campo a corregir.
	if action == "REGISTER" {
		normalizedEmail := strings.ToLower(strings.TrimSpace(email))
		emailExists, err := s.userRepo.ExistsByEmail(ctx, normalizedEmail)
		if err != nil {
			return err
		}
		if emailExists {
			return domain.ErrEmailTaken
		}

		if u := strings.TrimSpace(username); u != "" {
			usernameExists, err := s.userRepo.ExistsByUsername(ctx, u)
			if err != nil {
				return err
			}
			if usernameExists {
				return domain.ErrUsernameTaken
			}
		}

		if d := strings.TrimSpace(documentNumber); d != "" {
			docExists, err := s.userRepo.ExistsByDocument(ctx, d)
			if err != nil {
				return err
			}
			if docExists {
				return domain.ErrDocumentTaken
			}
		}
	}

	// Verificar si ya se ha solicitado un OTP recientemente en los últimos 60 segundos
	recent, err := s.otpRepo.HasRecentOTP(ctx, email, action, 60)
	if err != nil {
		slog.Error("Error al verificar OTP reciente", "error", err, "email", email)
		return err
	}
	if recent {
		return errors.New("Por favor, espera 60 segundos antes de solicitar otro código de verificación")
	}

	// 1. Generar código OTP
	code, err := generateNumericOTP()
	if err != nil {
		slog.Error("Error al generar código OTP seguro", "error", err, "email", email)
		return fmt.Errorf("error al generar código de seguridad")
	}

	// 2. Persistir en la base de datos (tiempo de vida de 5 minutos)
	otp := &domain.UserOTP{
		Email:     email,
		OTPCode:   code,
		Action:    action,
		ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		Used:      false,
	}

	if err := s.otpRepo.CreateOTP(ctx, otp); err != nil {
		return err
	}

	// 3. Invocar al servicio de email (único canal de entrega del código al usuario)
	if err := s.emailSrv.SendOTP(ctx, email, code); err != nil {
		return err
	}

	return nil
}

// determineCountry consulta de manera resiliente un servicio de GeoIP para obtener el país de origen de una IP.
func determineCountry(ip string) string {
	ip = strings.TrimSpace(ip)
	// Fallback por defecto si no hay IP, si es loopback o si es una subred privada local
	if ip == "" || ip == "127.0.0.1" || ip == "::1" || ip == "localhost" ||
		strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "172.16.") ||
		strings.HasPrefix(ip, "172.17.") ||
		strings.HasPrefix(ip, "172.18.") ||
		strings.HasPrefix(ip, "172.19.") ||
		strings.HasPrefix(ip, "172.2") ||
		strings.HasPrefix(ip, "172.3") {
		return defaultCountry
	}

	// SEGURIDAD: la IP proviene de cabeceras controlables por el cliente (X-Forwarded-For/X-Real-IP).
	// Validar que sea una IP real antes de concatenarla a la URL saliente evita inyección de URL / SSRF.
	if net.ParseIP(ip) == nil {
		slog.Warn("IP de origen inválida para GeoIP (usando fallback 'Desconocido')", "ip", ip)
		return defaultCountry
	}

	client := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}

	resp, err := client.Get("http://ip-api.com/json/" + ip)
	if err != nil {
		slog.Warn("GeoIP lookup falló (usando fallback 'Desconocido')", "ip", ip, "error", err)
		return defaultCountry
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("GeoIP lookup retornó status no exitoso (usando fallback 'Desconocido')", "ip", ip, "status", resp.Status)
		return defaultCountry
	}

	var result struct {
		Status  string `json:"status"`
		Country string `json:"country"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Warn("Error al decodificar respuesta de GeoIP (usando fallback 'Desconocido')", "ip", ip, "error", err)
		return defaultCountry
	}

	if result.Status == "success" && result.Country != "" {
		return result.Country
	}

	return defaultCountry
}

// Register refactorizado: ahora consume y valida un OTP de REGISTER antes de la creación del usuario y devuelve el token de inicio de sesión.
func (s *authUseCase) Register(ctx context.Context, req domain.RegisterRequest) (domain.AuthResponse, error) {
	// Normalizar y sanitizar entradas del usuario
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.TrimSpace(req.Username)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	if req.MiddleName != nil {
		trimmed := strings.TrimSpace(*req.MiddleName)
		req.MiddleName = &trimmed
	}
	if req.SecondLastName != nil {
		trimmed := strings.TrimSpace(*req.SecondLastName)
		req.SecondLastName = &trimmed
	}

	slog.Debug("Ejecutando caso de uso de Registro", "email", req.Email, "username", req.Username)

	var res domain.AuthResponse

	// Validaciones básicas de negocio
	if req.Email == "" || req.Username == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" || req.OTPCode == "" {
		return res, errors.New("Los campos email, username, password, first_name, last_name y otp_code son requeridos")
	}

	// Validar la estructura del correo electrónico
	if !emailRegex.MatchString(req.Email) {
		return res, errors.New("el formato del correo electrónico es inválido")
	}

	// Validar el formato del nombre de usuario (solo se permiten letras, números, guiones y guiones bajos)
	if !usernameRegex.MatchString(req.Username) {
		return res, errors.New("el nombre de usuario contiene caracteres no permitidos (solo se permiten letras, números, guiones y guiones bajos)")
	}

	// Validar fortaleza y complejidad de la contraseña (exigida por políticas de seguridad)
	if err := validatePasswordStrength(req.Password); err != nil {
		return res, err
	}

	// 1. Validar y consumir OTP para el registro
	if err := s.otpRepo.ValidateAndConsumeOTP(ctx, req.Email, req.OTPCode, "REGISTER"); err != nil {
		return res, err
	}

	// 2. Generar hash seguro de la contraseña con bcrypt (costo recomendado 12)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		slog.Error("Error al encriptar contraseña con bcrypt", "error", err)
		return res, fmt.Errorf("error al procesar credenciales")
	}

	// 3. Determinar país de forma automática por GeoIP a partir de la IP de registro
	country := determineCountry(req.RegistrationIP)

	// 4. Mapear DTO RegisterRequest a la entidad de dominio User (ID se genera en BD)
	user := &domain.User{
		Email:          req.Email,
		Username:       req.Username,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		MiddleName:     req.MiddleName,
		SecondLastName: req.SecondLastName,
		AvatarIndex:    0,
		UserTypeID:     req.UserTypeID,
		DocumentTypeID: req.DocumentTypeID,
		DocumentNumber: req.DocumentNumber,
		PasswordHash:   string(hashedPassword),
		RegistrationIP: &req.RegistrationIP,
		Country:        &country,
	}

	// 5. Persistir en repositorio
	if err := s.userRepo.Create(ctx, user); err != nil {
		return res, err
	}

	// Registrar la contraseña inicial en el historial de contraseñas
	if err := s.userRepo.AddPasswordHistory(ctx, user.ID, user.PasswordHash); err != nil {
		slog.Error("Fallo al registrar contraseña inicial en historial", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al registrar en historial de contraseñas")
	}

	// 5.5 Limpiar sesiones previas del usuario antes de crear la nueva para forzar login único
	if err := s.userRepo.DeleteUserSessions(ctx, user.ID); err != nil {
		slog.Error("Fallo al limpiar sesiones previas tras registro", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al procesar sesiones de usuario")
	}

	// 6. Crear token JWT con claims estándar (duración de 10 años para app móvil/CUSTOMER)
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().AddDate(10, 0, 0).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Firmar el token usando el secreto inyectado
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		slog.Error("Fallo crítico al firmar el token JWT de usuario tras registro", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al emitir el token de sesión")
	}

	// Registrar la sesión activa en la base de datos con expiración DESLIZANTE (sliding window).
	// El token del cliente no cambia (sigue siendo de larga vida), pero la sesión en BD caduca
	// tras SessionSlidingWindow de inactividad y se renueva en cada request autenticado.
	// Esto mantiene el dispositivo "vinculado" mientras se use, y acota un token filtrado.
	sessionExpiresAt := time.Now().Add(commonMiddleware.SessionSlidingWindow)
	if err := s.userRepo.CreateSession(ctx, user.ID, tokenString, sessionExpiresAt); err != nil {
		slog.Error("Fallo al registrar la sesión activa tras registro", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al iniciar sesión (fallo de registro de sesión)")
	}

	slog.Info("Registro exitoso e inicio de sesión automático. Emisión de token JWT autorizada", "user_id", user.ID)

	res.Token = tokenString
	return res, nil
}

// Login autentica un usuario y genera un token de sesión firmado en formato JWT.
func (s *authUseCase) Login(ctx context.Context, req domain.LoginRequest) (domain.AuthResponse, error) {
	slog.Debug("Ejecutando caso de uso de Inicio de Sesión", "email", req.Email)

	var res domain.AuthResponse

	if req.Email == "" || req.Password == "" {
		return res, errors.New("El correo electrónico y la contraseña son requeridos")
	}

	// 1. Buscar el usuario registrado por email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		slog.Warn("Login denegado: usuario no encontrado en base de datos", "email", req.Email)
		return res, errors.New("Credenciales incorrectas")
	}

	// 2. Comparar el hash bcrypt con la contraseña recibida
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		slog.Warn("Login denegado: contraseña incorrecta", "email", req.Email, "user_id", user.ID)
		return res, errors.New("Credenciales incorrectas")
	}

	// 2.5 Limpiar sesiones previas del usuario antes de crear la nueva para forzar login único
	if err := s.userRepo.DeleteUserSessions(ctx, user.ID); err != nil {
		slog.Error("Fallo al limpiar sesiones previas tras login", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al procesar sesiones de usuario")
	}

	// 3. Crear token JWT con claims estándar (duración de 10 años para app móvil/CUSTOMER)
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().AddDate(10, 0, 0).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Firmar el token usando el secreto inyectado
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		slog.Error("Fallo crítico al firmar el token JWT de usuario", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al emitir el token de sesión")
	}

	// Registrar la sesión activa en la base de datos con expiración DESLIZANTE (sliding window).
	// El token del cliente no cambia (sigue siendo de larga vida), pero la sesión en BD caduca
	// tras SessionSlidingWindow de inactividad y se renueva en cada request autenticado.
	// Esto mantiene el dispositivo "vinculado" mientras se use, y acota un token filtrado.
	sessionExpiresAt := time.Now().Add(commonMiddleware.SessionSlidingWindow)
	if err := s.userRepo.CreateSession(ctx, user.ID, tokenString, sessionExpiresAt); err != nil {
		slog.Error("Fallo al registrar la sesión activa tras login", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al iniciar sesión (fallo de registro de sesión)")
	}

	slog.Info("Sesión iniciada con éxito. Emisión de token JWT autorizada", "user_id", user.ID)

	res.Token = tokenString
	return res, nil
}

// LoginAdmin autentica un usuario para el BackOffice y genera un token JWT con el claim 'user_type'.
// SEGURIDAD CRÍTICA: Si el usuario es CUSTOMER, retorna exactamente el mismo error genérico
// que una contraseña incorrecta para mitigar ataques de enumeración de usuarios.
func (s *authUseCase) LoginAdmin(ctx context.Context, req domain.LoginAdminRequest) (domain.AuthResponse, error) {
	slog.Debug("Ejecutando caso de uso de Login Administrativo (BackOffice)", "email", req.Email)

	var res domain.AuthResponse

	if req.Email == "" || req.Password == "" {
		return res, errors.New("El correo electrónico y la contraseña son requeridos")
	}

	// 1. Buscar el usuario registrado por email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		slog.Warn("Login BackOffice denegado: usuario no encontrado en base de datos", "email", req.Email)
		// SEGURIDAD: Mismo mensaje genérico que contraseña incorrecta
		return res, errors.New("Credenciales incorrectas")
	}

	// 2. Comparar el hash bcrypt con la contraseña recibida
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		slog.Warn("Login BackOffice denegado: contraseña incorrecta", "email", req.Email, "user_id", user.ID)
		return res, errors.New("Credenciales incorrectas")
	}

	// 3. VALIDACIÓN DE ROL: Si RequiresAdmin, verificar que el usuario sea ADMIN.
	//    Si es CUSTOMER u otro rol, abortar con el MISMO error genérico para evitar enumeración.
	if req.RequiresAdmin {
		userTypeCode, err := s.userRepo.GetUserTypeCode(ctx, user.UserTypeID)
		if err != nil {
			slog.Warn("Login BackOffice denegado: no se pudo verificar el tipo de usuario",
				"email", req.Email, "user_id", user.ID, "user_type_id", user.UserTypeID, "error", err,
			)
			// SEGURIDAD: Mismo mensaje genérico — no revelamos detalles del error interno
			return res, errors.New("Credenciales incorrectas")
		}

		if userTypeCode != "ADMIN" {
			slog.Warn("Login BackOffice denegado: rol insuficiente (usuario no es ADMIN)",
				"email", req.Email, "user_id", user.ID, "user_type_code", userTypeCode,
			)
			// SEGURIDAD CRÍTICA: NO generamos token ni sesión. Retornamos el MISMO error
			// genérico para impedir que un atacante descubra que el usuario existe con otro rol.
			return res, errors.New("Credenciales incorrectas")
		}
	}

	// 4. Crear token JWT con claims extendidos incluyendo user_type para validación posterior
	claims := jwt.MapClaims{
		"user_id":   user.ID,
		"user_type": "ADMIN",
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		slog.Error("Fallo crítico al firmar el token JWT administrativo", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al emitir el token de sesión")
	}

	// 5. Registrar la sesión activa en la base de datos (duración inicial de 10 minutos)
	sessionExpiresAt := time.Now().Add(10 * time.Minute)
	if err := s.userRepo.CreateSession(ctx, user.ID, tokenString, sessionExpiresAt); err != nil {
		slog.Error("Fallo al registrar la sesión activa tras login BackOffice", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al iniciar sesión (fallo de registro de sesión)")
	}

	slog.Info("Sesión administrativa iniciada con éxito. Emisión de token JWT con user_type=ADMIN autorizada", "user_id", user.ID)

	res.Token = tokenString
	return res, nil
}

// Logout maneja el cierre de sesión de un usuario y elimina la sesión activa de la base de datos.
func (s *authUseCase) Logout(ctx context.Context, userID int64, token string) error {
	slog.Info("Cierre de sesión solicitado y procesado en el caso de uso", "user_id", userID)
	if token != "" {
		return s.userRepo.DeleteSession(ctx, token)
	}
	return nil
}


// ResetPassword valida el OTP para "RECOVER", hashea la nueva contraseña con bcrypt y actualiza la tabla users.
func (s *authUseCase) ResetPassword(ctx context.Context, email, newPassword, otpCode string) error {
	slog.Info("Ejecutando caso de uso de Restablecimiento de Contraseña", "email", email)

	if email == "" || newPassword == "" || otpCode == "" {
		return errors.New("Los campos email, new_password y otp_code son requeridos")
	}

	// Validar fortaleza y complejidad de la contraseña
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	// 1. Buscar al usuario activo por email para garantizar su existencia
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		slog.Warn("Usuario no encontrado para restablecimiento de contraseña", "email", email)
		// SEGURIDAD: mismo mensaje genérico que un OTP inválido para no revelar si el correo existe (anti-enumeración).
		return errors.New("Código OTP inválido, expirado o ya consumido")
	}

	// 2. Obtener los últimos 5 hashes del historial del usuario
	history, err := s.userRepo.GetPasswordHistory(ctx, user.ID)
	if err != nil {
		slog.Error("Fallo al obtener historial de contraseñas", "error", err, "user_id", user.ID)
		return fmt.Errorf("Error al verificar el historial de contraseñas")
	}

	// 3. Comparar la nueva contraseña con la contraseña actual
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)); err == nil {
		return errors.New("La nueva contraseña no puede coincidir con ninguna de las últimas 5 contraseñas")
	}

	// 4. Comparar la nueva contraseña con las contraseñas previas del historial
	for _, histHash := range history {
		if err := bcrypt.CompareHashAndPassword([]byte(histHash), []byte(newPassword)); err == nil {
			return errors.New("La nueva contraseña no puede coincidir con ninguna de las últimas 5 contraseñas")
		}
	}

	// 5. Validar y consumir OTP para "RECOVER" (se consume después de validar el historial)
	if err := s.otpRepo.ValidateAndConsumeOTP(ctx, email, otpCode, "RECOVER"); err != nil {
		return err
	}

	// 6. Hashear la nueva contraseña con bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		slog.Error("Fallo al hashear la nueva contraseña", "error", err)
		return fmt.Errorf("Error al procesar credenciales")
	}

	// 7. Actualizar en la base de datos
	if err := s.userRepo.UpdatePassword(ctx, user.ID, string(hashedPassword)); err != nil {
		return err
	}

	// Registrar el nuevo hash en el historial de contraseñas del usuario
	if err := s.userRepo.AddPasswordHistory(ctx, user.ID, string(hashedPassword)); err != nil {
		slog.Error("Fallo al agregar la nueva contraseña al historial", "error", err, "user_id", user.ID)
		return fmt.Errorf("Error al registrar en el historial de contraseñas")
	}

	slog.Info("Contraseña restablecida de manera exitosa", "user_id", user.ID, "email", email)
	return nil
}

// DeleteAccount inicia la eliminación de cuenta tras verificar el OTP de DELETE.
//
// Modelo de borrado con periodo de gracia (defendible y alineado con Play Store/App Store):
//  1. Se marca la cuenta como eliminada (deleted_at = NOW()), lo que la desactiva al instante:
//     el login y todas las consultas filtran por `deleted_at IS NULL`.
//  2. Se revoca el acceso de inmediato: se cierran todas las sesiones activas y se eliminan
//     los tokens de notificaciones push (FCM), de modo que el uso de datos cesa enseguida.
//  3. Transcurridos 15 días de gracia (ventana de recuperación ante error o robo de cuenta),
//     un cron purga físicamente la cuenta con DELETE FROM users, disparando los ON DELETE
//     CASCADE que eliminan los datos personales de forma irreversible; los registros con
//     ON DELETE SET NULL (comentarios, mensajes, compromisos) quedan anonimizados.
func (s *authUseCase) DeleteAccount(ctx context.Context, userID int64, otpCode string) error {
	slog.Info("Ejecutando caso de uso de Borrado de Cuenta con OTP", "user_id", userID)

	if userID <= 0 {
		return errors.New("ID de usuario inválido")
	}
	if otpCode == "" {
		return errors.New("El código OTP es requerido para confirmar la eliminación de la cuenta")
	}

	// 1. Obtener los datos del usuario mediante FindByID para recuperar su correo electrónico
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		slog.Warn("Usuario no encontrado para eliminación", "user_id", userID)
		return errors.New("Usuario no encontrado o ya inactivo")
	}

	// 2. Validar y consumir OTP para "DELETE" atado a su email
	if err := s.otpRepo.ValidateAndConsumeOTP(ctx, user.Email, otpCode, "DELETE"); err != nil {
		return err
	}

	// 3. Marcar la cuenta como eliminada (desactivación inmediata + inicio del periodo de gracia de 15 días).
	if err := s.userRepo.SoftDelete(ctx, userID); err != nil {
		return err
	}

	// 4. Revocar el acceso de inmediato. Es "best-effort": si falla, la cuenta ya quedó desactivada
	//    y la purga a 15 días eliminará igualmente sesiones y tokens por cascada; solo lo registramos.
	if err := s.userRepo.DeleteUserSessions(ctx, userID); err != nil {
		slog.Error("No se pudieron cerrar las sesiones activas al eliminar la cuenta", "error", err, "user_id", userID)
	}
	if err := s.notifRepo.DeleteAllUserDeviceTokens(ctx, userID); err != nil {
		slog.Error("No se pudieron revocar los tokens push al eliminar la cuenta", "error", err, "user_id", userID)
	}

	slog.Info("Cuenta marcada para eliminación y acceso revocado con éxito", "user_id", userID, "email", user.Email)
	return nil
}

// GetProfile obtiene el perfil completo de un usuario activo por su ID.
func (s *authUseCase) GetProfile(ctx context.Context, userID int64) (*domain.User, error) {
	slog.Debug("Obteniendo perfil completo del usuario", "user_id", userID)
	return s.userRepo.FindByID(ctx, userID)
}

// ValidateOTP valida que un OTP sea correcto y vigente sin consumirlo.
func (s *authUseCase) ValidateOTP(ctx context.Context, email, code, action string) error {
	slog.Info("Procesando caso de uso de validación de OTP (sin consumo)", "email", email, "action", action)

	if email == "" || code == "" || action == "" {
		return errors.New("El correo electrónico, el código OTP y la acción son campos requeridos")
	}

	if action != "REGISTER" && action != "RECOVER" && action != "DELETE" {
		return fmt.Errorf("Acción de OTP inválida: %s", action)
	}

	return s.otpRepo.ValidateOTPOnly(ctx, email, code, action)
}

// SearchUsers busca usuarios aplicando validaciones de MVP (mínimo 3 caracteres, límite 10).
func (s *authUseCase) SearchUsers(ctx context.Context, query string) ([]domain.UserSearchResult, error) {
	// Normalizar el query (quitar espacios sobrantes)
	q := strings.TrimSpace(query)

	// Si el parámetro tiene menos de 3 caracteres, retornar vacío inmediatamente sin consultar la BD
	if len(q) < 3 {
		slog.Debug("Búsqueda de usuarios omitida: el query tiene menos de 3 caracteres", "query", q)
		return []domain.UserSearchResult{}, nil
	}

	// Consultar base de datos con un límite estricto de 10
	return s.userRepo.SearchUsers(ctx, q, 10)
}

// validatePasswordStrength comprueba si una contraseña cumple con las políticas de complejidad requeridas:
// - Mínimo 8 caracteres
// - Al menos una letra mayúscula
// - Al menos una letra minúscula
// - Al menos un dígito numérico
// - Al menos un carácter especial del conjunto !@#$%&*
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("la contraseña debe tener al menos 8 caracteres de longitud")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%&*", r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("la contraseña debe contener al menos una letra mayúscula")
	}
	if !hasLower {
		return errors.New("la contraseña debe contener al menos una letra minúscula")
	}
	if !hasDigit {
		return errors.New("la contraseña debe contener al menos un número")
	}
	if !hasSpecial {
		return errors.New("la contraseña debe contener al menos un carácter especial (!@#$%&*)")
	}

	return nil
}

