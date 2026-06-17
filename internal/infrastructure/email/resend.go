package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
)

// resendAPIURL es el endpoint de envío de correos de la API HTTP de Resend.
const resendAPIURL = "https://api.resend.com/emails"

// defaultResendFrom es el remitente por defecto cuando no se ha definido RESEND_FROM.
// onboarding@resend.dev funciona sin verificar un dominio propio, pero en el plan gratuito
// SOLO permite enviar al correo de la cuenta dueña de la API Key. Para enviar a cualquier
// destinatario hay que verificar un dominio en Resend y ajustar RESEND_FROM.
const defaultResendFrom = "Sakoo <onboarding@resend.dev>"

type resendEmailService struct {
	apiKey string
	from   string
	client *http.Client
}

// newResendEmailService construye el proveedor de correo basado en Resend si RESEND_API_KEY
// está definida. Retorna nil cuando no está configurado, permitiendo a la factory caer al
// siguiente proveedor (SMTP o Mock).
func newResendEmailService() domain.EmailService {
	apiKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	if apiKey == "" {
		return nil
	}

	from := strings.TrimSpace(os.Getenv("RESEND_FROM"))
	if from == "" {
		from = defaultResendFrom
	}

	slog.Info("🚀 Servicio de correo Resend inicializado correctamente.", "from", from)
	return &resendEmailService{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// resendRequest representa el cuerpo JSON aceptado por POST https://api.resend.com/emails.
type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (s *resendEmailService) SendOTP(ctx context.Context, toEmail, code string) error {
	payload := resendRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: otpEmailSubject,
		HTML:    buildOTPEmailHTML(code),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Fallo al serializar el cuerpo de la petición a Resend", "error", err, "destinatario", toEmail)
		return fmt.Errorf("error al preparar el correo de verificación")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendAPIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Error("Fallo al construir la petición HTTP a Resend", "error", err, "destinatario", toEmail)
		return fmt.Errorf("error al preparar el correo de verificación")
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Error("Fallo de red al enviar correo OTP vía Resend", "error", err, "destinatario", toEmail)
		return fmt.Errorf("error al enviar el correo de verificación")
	}
	defer resp.Body.Close()

	// Resend responde 200 OK con el id del correo en caso de éxito.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// SEGURIDAD: el cuerpo del error de Resend se registra en el log, no se expone al cliente.
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		slog.Error("Resend retornó un estado no exitoso al enviar el OTP",
			"status", resp.StatusCode, "destinatario", toEmail, "respuesta", string(respBody),
		)
		return fmt.Errorf("error al enviar el correo de verificación")
	}

	slog.Info("Correo electrónico de verificación OTP enviado con éxito vía Resend", "destinatario", toEmail)
	return nil
}
