package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type mockEmailService struct{}

// NewMockEmailService crea un servicio de email simulado para desarrollo.
func NewMockEmailService() domain.EmailService {
	return &mockEmailService{}
}

func (s *mockEmailService) SendOTP(ctx context.Context, email, code string) error {
	slog.Info("----------------------------------------------------------------------")
	slog.Info("MOCK EMAIL SERVICE: SE HA GENERADO Y ENVIADO UN CÓDIGO OTP",
		"email", email,
		"otp_code", code,
	)
	slog.Info("----------------------------------------------------------------------")
	return nil
}

type smtpEmailService struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

// NewEmailService selecciona la implementación de envío de correo según las variables
// de entorno disponibles, con el siguiente orden de prioridad:
//  1. Resend  — si RESEND_API_KEY está definida (proveedor HTTP recomendado para el inicio).
//  2. SMTP    — si SMTP_HOST/SMTP_USER/SMTP_PASSWORD están definidas.
//  3. Mock    — fallback resiliente que solo escribe el OTP en consola (desarrollo).
func NewEmailService() domain.EmailService {
	// Prioridad 1: Resend (HTTP API). Es el proveedor objetivo para el arranque del producto.
	if srv := newResendEmailService(); srv != nil {
		return srv
	}

	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	if host == "" || user == "" || password == "" {
		slog.Warn("⚠️ SMTP no está configurado (SMTP_HOST, SMTP_USER o SMTP_PASSWORD ausentes). El servicio de Email operará en modo MOCK (Consola).")
		return NewMockEmailService()
	}

	if port == "" {
		port = "587" // Puerto SMTP estándar para TLS/STARTTLS
	}
	if from == "" {
		from = user
	}

	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "production"
	}

	var envSpanish string
	switch env {
	case "production":
		envSpanish = "producción"
	case "qa":
		envSpanish = "qa"
	case "local":
		envSpanish = "local"
	default:
		envSpanish = env
	}

	slog.Info("🚀 Servicio de correo SMTP inicializado correctamente en " + envSpanish + ".", "ambiente", env)
	return &smtpEmailService{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
	}
}

func (s *smtpEmailService) SendOTP(ctx context.Context, toEmail, code string) error {
	// Construir cabeceras y cuerpo en HTML con excelente diseño estético
	subject := "Subject: " + otpEmailSubject + "\r\n"
	contentType := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	body := buildOTPEmailHTML(code)

	msg := []byte(subject + contentType + "\r\n" + body)

	// Configurar autenticación PlainAuth
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	// Enlace SMTP completo (host:puerto)
	addr := s.host + ":" + s.port

	// Enviar el correo electrónico
	err := smtp.SendMail(addr, auth, s.from, []string{toEmail}, msg)
	if err != nil {
		slog.Error("Fallo al enviar correo de verificación OTP vía SMTP", "error", err, "destinatario", toEmail)
		return err
	}

	slog.Info("Correo electrónico de verificación OTP enviado con éxito vía SMTP", "destinatario", toEmail)
	return nil
}

// otpEmailSubject es el asunto compartido por todos los proveedores de correo para el OTP.
const otpEmailSubject = "Código de Seguridad Sakoo"

// buildOTPEmailHTML construye el cuerpo HTML del correo de verificación OTP.
// Lo comparten el proveedor SMTP y el proveedor Resend para mantener una sola plantilla.
func buildOTPEmailHTML(code string) string {
	return fmt.Sprintf(`
		<div style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; max-width: 600px; margin: auto; padding: 30px; border: 1px solid #e1e8ed; border-radius: 12px; background-color: #ffffff; box-shadow: 0 4px 12px rgba(0,0,0,0.05);">
			<div style="text-align: center; margin-bottom: 25px;">
				<h1 style="color: #0f172a; margin: 0; font-size: 28px; font-weight: 700; letter-spacing: -0.5px;">Sakoo</h1>
				<span style="color: #64748b; font-size: 14px;">Tus divisas en tiempo real</span>
			</div>

			<div style="border-top: 1px solid #f1f5f9; padding-top: 25px;">
				<h2 style="color: #3b82f6; margin-top: 0; font-size: 20px; font-weight: 600;">Verificación de Seguridad</h2>
				<p style="color: #334155; line-height: 1.6; font-size: 15px;">Has solicitado una operación crítica dentro del ecosistema Sakoo (Registro, Recuperación de Contraseña o Baja de Cuenta).</p>
				<p style="color: #334155; line-height: 1.6; font-size: 15px;">Para continuar de forma segura, introduce el siguiente código de verificación de un solo uso (OTP) en la aplicación móvil:</p>

				<div style="background-color: #f8fafc; border: 1px dashed #cbd5e1; padding: 20px; text-align: center; font-size: 32px; font-weight: 700; letter-spacing: 8px; color: #1e293b; border-radius: 8px; margin: 25px 0;">
					%s
				</div>

				<p style="color: #64748b; font-size: 13px; line-height: 1.5; margin-top: 20px;">
					⚠️ <strong>Importante:</strong> Este código es personal, de un solo uso y expirará en <strong>5 minutos</strong>. Nunca compartas este código con nadie. Si no has solicitado este código, puedes ignorar este correo con total tranquilidad.
				</p>
			</div>

			<div style="border-top: 1px solid #f1f5f9; margin-top: 35px; padding-top: 20px; text-align: center; color: #94a3b8; font-size: 12px;">
				<p style="margin: 0 0 5px 0;">&copy; 2026 Sakoo. Todos los derechos reservados.</p>
				<p style="margin: 0;">Diseñado para el ecosistema financiero moderno en Venezuela.</p>
			</div>
		</div>
	`, code)
}
