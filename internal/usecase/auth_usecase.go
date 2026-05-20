package usecase

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/aaron/sakoo-backend/internal/infrastructure/email"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authUseCase struct {
	userRepo  domain.UserRepository
	otpRepo   domain.OTPRepository
	emailSrv  email.EmailService
	jwtSecret string
}

// NewAuthUseCase crea una instancia concreta del caso de uso de autenticación con todas sus dependencias.
func NewAuthUseCase(
	userRepo domain.UserRepository,
	otpRepo domain.OTPRepository,
	emailSrv email.EmailService,
	jwtSecret string,
) domain.AuthUseCase {
	return &authUseCase{
		userRepo:  userRepo,
		otpRepo:   otpRepo,
		emailSrv:  emailSrv,
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

// RequestOTP genera un OTP, lo persiste con 15 minutos de vigencia, y lo envía por email.
func (s *authUseCase) RequestOTP(ctx context.Context, email string, action string) error {
	slog.Info("Procesando solicitud de OTP", "email", email, "action", action)

	if email == "" || action == "" {
		return errors.New("el correo electrónico y la acción son requeridos")
	}

	if action != "REGISTER" && action != "RECOVER" && action != "DELETE" {
		return fmt.Errorf("acción de OTP inválida: %s", action)
	}

	// 1. Generar código OTP
	code, err := generateNumericOTP()
	if err != nil {
		slog.Error("Error al generar código OTP seguro", "error", err, "email", email)
		return fmt.Errorf("error al generar código de seguridad: %w", err)
	}

	// 2. Persistir en la base de datos
	otp := &domain.UserOTP{
		Email:     email,
		OTPCode:   code,
		Action:    action,
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
		Used:      false,
	}

	if err := s.otpRepo.CreateOTP(ctx, otp); err != nil {
		return err
	}

	// 3. Invocar al servicio de email
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
		return "Venezuela"
	}

	client := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}

	resp, err := client.Get("http://ip-api.com/json/" + ip)
	if err != nil {
		slog.Warn("GeoIP lookup falló (usando fallback Venezuela)", "ip", ip, "error", err)
		return "Venezuela"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("GeoIP lookup retornó status no exitoso (usando fallback Venezuela)", "ip", ip, "status", resp.Status)
		return "Venezuela"
	}

	var result struct {
		Status  string `json:"status"`
		Country string `json:"country"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Warn("Error al decodificar respuesta de GeoIP (usando fallback Venezuela)", "ip", ip, "error", err)
		return "Venezuela"
	}

	if result.Status == "success" && result.Country != "" {
		return result.Country
	}

	return "Venezuela"
}

// Register refactorizado: ahora consume y valida un OTP de REGISTER antes de la creación del usuario.
func (s *authUseCase) Register(ctx context.Context, req domain.RegisterRequest) error {
	slog.Debug("Ejecutando caso de uso de Registro", "email", req.Email, "username", req.Username)

	// Validaciones básicas de negocio
	if req.Email == "" || req.Username == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" || req.OTPCode == "" {
		return errors.New("los campos email, username, password, first_name, last_name y otp_code son requeridos")
	}

	// 1. Validar y consumir OTP para el registro
	if err := s.otpRepo.ValidateAndConsumeOTP(ctx, req.Email, req.OTPCode, "REGISTER"); err != nil {
		return err
	}

	// 2. Generar hash seguro de la contraseña con bcrypt (costo recomendado 12)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		slog.Error("Error al encriptar contraseña con bcrypt", "error", err)
		return fmt.Errorf("error al procesar credenciales: %w", err)
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
		return err
	}

	// Registrar la contraseña inicial en el historial de contraseñas
	if err := s.userRepo.AddPasswordHistory(ctx, user.ID, user.PasswordHash); err != nil {
		slog.Error("Fallo al registrar contraseña inicial en historial", "error", err, "user_id", user.ID)
		return fmt.Errorf("error al registrar en historial de contraseñas: %w", err)
	}

	return nil
}

// Login autentica un usuario y genera un token de sesión firmado en formato JWT.
func (s *authUseCase) Login(ctx context.Context, req domain.LoginRequest) (domain.AuthResponse, error) {
	slog.Debug("Ejecutando caso de uso de Inicio de Sesión", "email", req.Email)

	var res domain.AuthResponse

	if req.Email == "" || req.Password == "" {
		return res, errors.New("correo y contraseña requeridos")
	}

	// 1. Buscar el usuario registrado por email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		slog.Warn("Login denegado: usuario no encontrado en base de datos", "email", req.Email)
		return res, errors.New("credenciales incorrectas")
	}

	// 2. Comparar el hash bcrypt con la contraseña recibida
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		slog.Warn("Login denegado: contraseña incorrecta", "email", req.Email, "user_id", user.ID)
		return res, errors.New("credenciales incorrectas")
	}

	// 3. Crear token JWT con claims estándar inyectando user_id como int64
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // Duración estándar de 24 horas
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Firmar el token usando el secreto inyectado
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		slog.Error("Fallo crítico al firmar el token JWT de usuario", "error", err, "user_id", user.ID)
		return res, fmt.Errorf("error al emitir el token de sesión: %w", err)
	}

	slog.Info("Sesión iniciada con éxito. Emisión de token JWT autorizada", "user_id", user.ID)

	res.Token = tokenString
	return res, nil
}

// DeleteMyAccount realiza el borrado lógico de la cuenta del usuario previa validación de sus credenciales (versión legacy).
func (s *authUseCase) DeleteMyAccount(ctx context.Context, userID int64, email, password string) error {
	slog.Info("Ejecutando caso de uso de borrado lógico de cuenta (Legacy)", "user_id", userID, "email", email)
	
	if userID <= 0 {
		return errors.New("ID de usuario inválido para borrado de cuenta")
	}
	if email == "" || password == "" {
		return errors.New("correo y contraseña requeridos para confirmar el borrado de cuenta")
	}

	// 1. Buscar el usuario registrado por email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		slog.Warn("Intento de borrado de cuenta denegado: usuario no encontrado o inactivo", "email", email)
		return errors.New("credenciales incorrectas")
	}

	// 2. Verificar que el token JWT corresponde al usuario que se desea eliminar
	if user.ID != userID {
		slog.Warn("Intento de violación de seguridad: ID de token no coincide con el email provisto", 
			"token_user_id", userID, 
			"target_user_id", user.ID, 
			"email", email,
		)
		return errors.New("no estás autorizado para eliminar esta cuenta")
	}

	// 3. Comparar el hash bcrypt con la contraseña recibida
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		slog.Warn("Intento de borrado de cuenta denegado: contraseña incorrecta", "email", email, "user_id", user.ID)
		return errors.New("credenciales incorrectas")
	}

	// 4. Proceder al borrado lógico
	return s.userRepo.SoftDelete(ctx, userID)
}

// ResetPassword valida el OTP para "RECOVER", hashea la nueva contraseña con bcrypt y actualiza la tabla users.
func (s *authUseCase) ResetPassword(ctx context.Context, email, newPassword, otpCode string) error {
	slog.Info("Ejecutando caso de uso de Restablecimiento de Contraseña", "email", email)

	if email == "" || newPassword == "" || otpCode == "" {
		return errors.New("los campos email, new_password y otp_code son requeridos")
	}

	// 1. Validar y consumir OTP para "RECOVER"
	if err := s.otpRepo.ValidateAndConsumeOTP(ctx, email, otpCode, "RECOVER"); err != nil {
		return err
	}

	// 2. Buscar al usuario activo por email para garantizar su existencia
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		slog.Warn("Usuario no encontrado para restablecimiento de contraseña", "email", email)
		return errors.New("el correo electrónico ingresado no corresponde a ningún usuario activo")
	}

	// Obtener los últimos 5 hashes del historial del usuario
	history, err := s.userRepo.GetPasswordHistory(ctx, user.ID)
	if err != nil {
		slog.Error("Fallo al obtener historial de contraseñas", "error", err, "user_id", user.ID)
		return fmt.Errorf("error al verificar historial de contraseñas: %w", err)
	}

	// Comparar la nueva contraseña con la contraseña actual
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)); err == nil {
		return errors.New("la nueva contraseña no puede coincidir con ninguna de las últimas 5 contraseñas")
	}

	// Comparar la nueva contraseña con las contraseñas previas del historial
	for _, histHash := range history {
		if err := bcrypt.CompareHashAndPassword([]byte(histHash), []byte(newPassword)); err == nil {
			return errors.New("la nueva contraseña no puede coincidir con ninguna de las últimas 5 contraseñas")
		}
	}

	// 3. Hashear la nueva contraseña con bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		slog.Error("Fallo al hashear la nueva contraseña", "error", err)
		return fmt.Errorf("error al procesar credenciales: %w", err)
	}

	// 4. Actualizar en la base de datos
	if err := s.userRepo.UpdatePassword(ctx, user.ID, string(hashedPassword)); err != nil {
		return err
	}

	// Registrar el nuevo hash en el historial de contraseñas del usuario
	if err := s.userRepo.AddPasswordHistory(ctx, user.ID, string(hashedPassword)); err != nil {
		slog.Error("Fallo al agregar la nueva contraseña al historial", "error", err, "user_id", user.ID)
		return fmt.Errorf("error al registrar en historial de contraseñas: %w", err)
	}

	slog.Info("Contraseña restablecida de manera exitosa", "user_id", user.ID, "email", email)
	return nil
}

// DeleteAccount realiza la eliminación lógica del usuario tras verificar su OTP de DELETE.
func (s *authUseCase) DeleteAccount(ctx context.Context, userID int64, otpCode string) error {
	slog.Info("Ejecutando caso de uso de Borrado Lógico de Cuenta con OTP", "user_id", userID)

	if userID <= 0 {
		return errors.New("ID de usuario inválido")
	}
	if otpCode == "" {
		return errors.New("el código OTP es requerido para confirmar la eliminación de la cuenta")
	}

	// 1. Obtener los datos del usuario mediante FindByID para recuperar su correo electrónico
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		slog.Warn("Usuario no encontrado para eliminación", "user_id", userID)
		return errors.New("usuario no encontrado o ya inactivo")
	}

	// 2. Validar y consumir OTP para "DELETE" atado a su email
	if err := s.otpRepo.ValidateAndConsumeOTP(ctx, user.Email, otpCode, "DELETE"); err != nil {
		return err
	}

	// 3. Ejecutar el soft delete
	if err := s.userRepo.SoftDelete(ctx, userID); err != nil {
		return err
	}

	slog.Info("Cuenta eliminada lógicamente de manera exitosa con OTP", "user_id", userID, "email", user.Email)
	return nil
}

// GetProfile obtiene el perfil completo de un usuario activo por su ID.
func (s *authUseCase) GetProfile(ctx context.Context, userID int64) (*domain.User, error) {
	slog.Debug("Obteniendo perfil completo del usuario", "user_id", userID)
	return s.userRepo.FindByID(ctx, userID)
}
