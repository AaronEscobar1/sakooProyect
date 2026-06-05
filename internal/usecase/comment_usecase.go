package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"

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

	cleanedContent := censorText(content)
	if cleanedContent != content {
		slog.Warn("Se detectó y censuró contenido ofensivo en el comentario...", "user_id", userID, "rate_id", rateID)
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

	slog.Info("Procesando listado de opiniones para tasa", "rate_id", rateID)

	return uc.repo.ListByRateID(ctx, rateID)
}

var badWords = []string{
	"mierda", "puta", "puto", "putas", "putos", "marico", "marica", "maricos", "maricas",
	"cabron", "cabrón", "cabrones", "coño", "coñazo", "joder", "pendejo", "pendejos", "pendeja", "pendejas",
	"verga", "mamaguevo", "mamagüevo", "mamaguebos", "mamaguevazo", "guevon", "güevón", "guevón", "guevones",
	"güevones", "guevo", "güevo", "malparido", "malparida", "malparidos", "hijo de puta", "hijo de perra",
	"hijodeputa", "chupalo", "chúpalo", "chupala", "chúpala", "singar", "maldito", "maldita", "malditos", "malditas",
	"mmg", "mmvg", "mrico", "mariko", "csm", "cdsm", "mgvo", "mgv", "mmgbo", "hijueputa",
	"mmgvo", "mmgvos", "mmgva", "mmgvas", "mamaguevos", "mamagüevos",
	"pajuo", "pajuos", "pajuato", "pajuatos", "idiota", "idiotas", "chupa culo", "chupaculo", "chupaculos",
	"marginal", "marginales", "mamaguebaso", "mamagüebaso", "teta", "tetas",
	"gafo", "gafos", "gafa", "gafas", "totona", "totonas", "becerro", "becerros", "sapo", "sapos",
	"diablo", "diablos", "pupu", "pupus", "pipi", "pipis", "homosexual", "homosexuales",
	"enchufado", "enchufados", "enchufao", "enchufaos", "gay", "gays",
	"gilipollas", "subnormal", "subnormales", "imbecil", "imbécil", "imbeciles", "imbéciles",
	"jala bola", "jala bolas", "jalabola", "jalabolas", "cono e madre", "coño e madre", "cono de madre", "coño de madre",
	"conoemadre", "coñoemadre", "conodemadre", "coñodemadre", "cono de la madre", "coño de la madre",
	"conodelamadre", "coñodelamadre", "cabeza de huevo", "cabezadehuevo", "ladilla", "ladillas", "niche", "niches",
	"webon", "webón", "webones", "huevon", "huevón", "huevones", "boludo", "boludos", "boluda", "boludas",
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

func censorText(content string) string {
	originalRunes := []rune(content)
	cleaned := cleanText(content)
	cleanedRunes := []rune(cleaned)

	toCensor := make([]bool, len(originalRunes))

	for _, badWord := range badWords {
		cleanedBad := cleanText(badWord)
		badRunes := []rune(cleanedBad)
		lenBad := len(badRunes)
		if lenBad == 0 {
			continue
		}

		for i := 0; i <= len(cleanedRunes)-lenBad; i++ {
			match := true
			for j := 0; j < lenBad; j++ {
				if cleanedRunes[i+j] != badRunes[j] {
					match = false
					break
				}
			}

			if match {
				// Validar límites de palabra
				if i > 0 {
					prevRune := cleanedRunes[i-1]
					if (prevRune >= 'a' && prevRune <= 'z') || (prevRune >= '0' && prevRune <= '9') {
						continue
					}
				}
				if i+lenBad < len(cleanedRunes) {
					nextRune := cleanedRunes[i+lenBad]
					if (nextRune >= 'a' && nextRune <= 'z') || (nextRune >= '0' && nextRune <= '9') {
						continue
					}
				}

				for j := 0; j < lenBad; j++ {
					toCensor[i+j] = true
				}
			}
		}
	}

	var result strings.Builder
	inBadWord := false
	for i, r := range originalRunes {
		if toCensor[i] {
			if !inBadWord {
				result.WriteString("****")
				inBadWord = true
			}
		} else {
			result.WriteRune(r)
			inBadWord = false
		}
	}

	return result.String()
}
