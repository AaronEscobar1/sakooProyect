package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
)

type commentUseCase struct {
	repo domain.CommentRepository
}

// NewCommentUseCase crea un caso de uso para comentarios en tasas de cambio.
func NewCommentUseCase(repo domain.CommentRepository) domain.CommentUseCase {
	return &commentUseCase{
		repo: repo,
	}
}

func (uc *commentUseCase) AddComment(ctx context.Context, userID, rateID int64, content string) (*domain.Comment, error) {
	if content == "" {
		return nil, errors.New("El contenido del comentario no puede estar vacío")
	}
	if rateID <= 0 {
		return nil, errors.New("El ID de tasa 'rate_id' no es válido")
	}

	slog.Info("Procesando creación de comentario para tasa", "user_id", userID, "rate_id", rateID)

	// Validar la regla de negocio: el usuario solo puede comentar 1 vez por tasa de cambio
	alreadyCommented, err := uc.repo.HasCommentedOnRate(ctx, userID, rateID)
	if err != nil {
		return nil, err
	}
	if alreadyCommented {
		return nil, errors.New("El usuario ya ha comentado en esta tasa de cambio, solo se permite un comentario por tasa")
	}

	cleanedContent := content
	if containsProfanity(content) {
		slog.Warn("Se detectó contenido ofensivo en el comentario, censurando...", "user_id", userID, "rate_id", rateID)
		cleanedContent = "****"
	}

	comment := &domain.Comment{
		UserID:  userID,
		RateID:  rateID,
		Content: cleanedContent,
	}

	err = uc.repo.Create(ctx, comment)
	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (uc *commentUseCase) GetCommentsByRate(ctx context.Context, rateID int64) ([]domain.Comment, error) {
	if rateID <= 0 {
		return nil, errors.New("El ID de tasa 'rate_id' no es válido")
	}

	slog.Info("Procesando listado de opiniones del día para tasa", "rate_id", rateID)

	// Obtener opiniones del día de hoy
	today := time.Now()
	return uc.repo.ListByRateIDAndDate(ctx, rateID, today)
}

var badWords = []string{
	"mierda", "puta", "puto", "putas", "putos", "marico", "marica", "maricos", "maricas",
	"cabron", "cabrón", "cabrones", "coño", "coñazo", "joder", "pendejo", "pendejos", "pendeja", "pendejas",
	"verga", "mamaguevo", "mamagüevo", "mamaguebos", "mamaguevazo", "guevon", "güevón", "guevón", "guevones",
	"güevones", "guevo", "güevo", "malparido", "malparida", "malparidos", "hijo de puta", "hijo de perra",
	"hijodeputa", "chupalo", "chúpalo", "chupala", "chúpala", "singar", "maldito", "maldita", "malditos", "malditas",
	"mmg", "mmvg", "mrico", "mariko", "csm", "cdsm", "mgvo", "mgv", "mmgbo", "hijueputa",
}

func cleanText(text string) string {
	text = strings.ToLower(text)
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ü", "u",
	)
	return replacer.Replace(text)
}

func containsProfanity(content string) bool {
	cleanedContent := cleanText(content)
	for _, badWord := range badWords {
		cleanedBadWord := cleanText(badWord)
		if strings.Contains(cleanedContent, cleanedBadWord) {
			if strings.Contains(cleanedBadWord, " ") {
				return true
			}
			words := strings.FieldsFunc(cleanedContent, func(r rune) bool {
				return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
			})
			for _, w := range words {
				if w == cleanedBadWord {
					return true
				}
			}
		}
	}
	return false
}
