package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/user"
	"github.com/aaron/sakoo-backend/ent/userpasswordhistory"
	"github.com/aaron/sakoo-backend/ent/usersession"
	"github.com/aaron/sakoo-backend/ent/usertype"
	"github.com/aaron/sakoo-backend/internal/domain"
)

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type userRepository struct {
	client *ent.Client
}

// NewUserRepository crea una nueva instancia del repositorio de usuarios utilizando Ent.
func NewUserRepository(client *ent.Client) domain.UserRepository {
	return &userRepository{
		client: client,
	}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Creando registro de usuario en Ent", "email", u.Email)

	builder := r.client.User.Create().
		SetEmail(u.Email).
		SetFirstName(u.FirstName).
		SetLastName(u.LastName).
		SetAvatarIndex(u.AvatarIndex).
		SetUserTypeID(u.UserTypeID).
		SetPasswordHash(u.PasswordHash)

	if u.Username != "" {
		builder.SetUsername(u.Username)
	}
	if u.MiddleName != nil {
		builder.SetMiddleName(*u.MiddleName)
	}
	if u.SecondLastName != nil {
		builder.SetSecondLastName(*u.SecondLastName)
	}
	if u.DocumentTypeID != nil {
		builder.SetDocumentTypeID(*u.DocumentTypeID)
	}
	if u.DocumentNumber != nil {
		builder.SetDocumentNumber(*u.DocumentNumber)
	}
	if u.RegistrationIP != nil {
		builder.SetRegistrationIP(*u.RegistrationIP)
	}
	if u.Country != nil {
		builder.SetCountry(*u.Country)
	}

	created, err := builder.Save(dbCtx)
	if err != nil {
		slog.Error("Fallo al insertar usuario en Ent", "error", err, "email", u.Email)
		if ent.IsConstraintError(err) {
			if strings.Contains(err.Error(), "username") {
				return domain.ErrUsernameTaken
			}
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("error al guardar usuario en base de datos")
	}

	u.ID = int64(created.ID)
	u.CreatedAt = created.CreatedAt
	u.UpdatedAt = created.UpdatedAt

	slog.Info("Usuario registrado exitosamente en base de datos", "id", u.ID, "email", u.Email)
	return nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return r.client.User.Query().Where(user.EmailEQ(email)).Exist(dbCtx)
}

func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return r.client.User.Query().Where(user.UsernameEQ(username)).Exist(dbCtx)
}

func (r *userRepository) ExistsByDocument(ctx context.Context, documentNumber string) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return r.client.User.Query().Where(user.DocumentNumberEQ(documentNumber)).Exist(dbCtx)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Buscando usuario por correo electrónico", "email", email)

	u, err := r.client.User.Query().
		Where(user.EmailEQ(email), user.DeletedAtIsNil()).
		Only(dbCtx)

	if err != nil {
		if ent.IsNotFound(err) {
			slog.Debug("Usuario no encontrado o inactivo en Ent", "email", email)
			return nil, fmt.Errorf("usuario no encontrado: %w", err)
		}
		slog.Error("Error al consultar usuario por email en Ent", "error", err, "email", email)
		return nil, fmt.Errorf("error de consulta en base de datos")
	}

	return toDomainUser(u), nil
}

func (r *userRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Buscando usuario por ID", "id", id)

	u, err := r.client.User.Query().
		Where(user.IDEQ(int(id)), user.DeletedAtIsNil()).
		Only(dbCtx)

	if err != nil {
		if ent.IsNotFound(err) {
			slog.Debug("Usuario no encontrado o inactivo en Ent", "id", id)
			return nil, fmt.Errorf("usuario no encontrado: %w", err)
		}
		slog.Error("Error al consultar usuario por ID en Ent", "error", err, "id", id)
		return nil, fmt.Errorf("error de consulta en base de datos")
	}

	return toDomainUser(u), nil
}

func (r *userRepository) SoftDelete(ctx context.Context, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info("Ejecutando borrado lógico de usuario", "user_id", userID)

	now := time.Now()
	count, err := r.client.User.Update().
		Where(user.IDEQ(int(userID)), user.DeletedAtIsNil()).
		SetDeletedAt(now).
		SetUpdatedAt(now).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al ejecutar soft delete en Ent", "error", err, "user_id", userID)
		return fmt.Errorf("error al eliminar lógicamente al usuario")
	}

	if count == 0 {
		slog.Warn("El usuario no existe o ya ha sido eliminado lógicamente", "user_id", userID)
		return fmt.Errorf("usuario no encontrado o ya eliminado")
	}

	slog.Info("Usuario eliminado lógicamente de forma exitosa", "user_id", userID)
	return nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info("Actualizando contraseña de usuario en Ent", "user_id", userID)

	count, err := r.client.User.Update().
		Where(user.IDEQ(int(userID)), user.DeletedAtIsNil()).
		SetPasswordHash(passwordHash).
		SetUpdatedAt(time.Now()).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al actualizar contraseña en Ent", "error", err, "user_id", userID)
		return fmt.Errorf("error al actualizar la contraseña del usuario")
	}

	if count == 0 {
		slog.Warn("El usuario no existe o ha sido eliminado lógicamente", "user_id", userID)
		return fmt.Errorf("usuario no encontrado o ya eliminado")
	}

	slog.Info("Contraseña del usuario actualizada exitosamente", "user_id", userID)
	return nil
}

func (r *userRepository) GetPasswordHistory(ctx context.Context, userID int64) ([]string, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando historial de contraseñas de usuario", "user_id", userID)

	entries, err := r.client.UserPasswordHistory.Query().
		Where(userpasswordhistory.UserID(userID)).
		Order(ent.Desc(userpasswordhistory.FieldCreatedAt), ent.Desc(userpasswordhistory.FieldID)).
		Limit(5).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al obtener historial de contraseñas en Ent", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al obtener historial de contraseñas")
	}

	var history []string
	for _, entry := range entries {
		history = append(history, entry.PasswordHash)
	}

	return history, nil
}

func (r *userRepository) AddPasswordHistory(ctx context.Context, userID int64, passwordHash string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando hash de contraseña en historial", "user_id", userID)

	_, err := r.client.UserPasswordHistory.Create().
		SetUserID(userID).
		SetPasswordHash(passwordHash).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al insertar en UserPasswordHistory", "error", err, "user_id", userID)
		return fmt.Errorf("error al registrar en historial de contraseñas")
	}

	// Mantener solo los últimos 5 registros
	oldEntries, err := r.client.UserPasswordHistory.Query().
		Where(userpasswordhistory.UserID(userID)).
		Order(ent.Desc(userpasswordhistory.FieldCreatedAt), ent.Desc(userpasswordhistory.FieldID)).
		Offset(5).
		All(dbCtx)

	if err == nil && len(oldEntries) > 0 {
		var idsToDelete []int
		for _, e := range oldEntries {
			idsToDelete = append(idsToDelete, e.ID)
		}
		_, _ = r.client.UserPasswordHistory.Delete().
			Where(userpasswordhistory.IDIn(idsToDelete...)).
			Exec(dbCtx)
	}

	return nil
}

func (r *userRepository) SearchUsers(ctx context.Context, query string, limit int) ([]domain.UserSearchResult, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Buscando usuarios en Ent", "query", query, "limit", limit)

	users, err := r.client.User.Query().
		Where(
			user.UsernameHasPrefix(query),
			user.DeletedAtIsNil(),
		).
		Order(ent.Asc(user.FieldUsername)).
		Limit(limit).
		All(dbCtx)

	if err != nil {
		slog.Error("Fallo al buscar usuarios en Ent", "error", err, "query", query)
		return nil, fmt.Errorf("error al buscar usuarios")
	}

	var results []domain.UserSearchResult
	for _, u := range users {
		displayName := fmt.Sprintf("%s %s", u.FirstName, u.LastName)
		avatarURL := fmt.Sprintf("https://sakoo-public-assets.s3.amazonaws.com/avatars/avatar_%d.png", u.AvatarIndex)

		results = append(results, domain.UserSearchResult{
			ID:          int64(u.ID),
			Username:    u.Username,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
		})
	}

	if results == nil {
		results = []domain.UserSearchResult{}
	}

	return results, nil
}

func (r *userRepository) CreateSession(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando nueva sesión de usuario en Ent", "user_id", userID)

	_, err := r.client.UserSession.Create().
		SetUserID(userID).
		SetToken(hashSessionToken(token)).
		SetExpiresAt(expiresAt).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al crear sesión de usuario en Ent", "error", err, "user_id", userID)
		return fmt.Errorf("error al crear sesión")
	}

	return nil
}

func (r *userRepository) ValidateSession(ctx context.Context, token string) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Validando sesión de usuario en Ent")

	return r.client.UserSession.Query().
		Where(
			usersession.TokenEQ(hashSessionToken(token)),
			usersession.ExpiresAtGT(time.Now()),
		).
		Exist(dbCtx)
}

func (r *userRepository) DeleteSession(ctx context.Context, token string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Eliminando sesión de usuario en Ent")

	_, err := r.client.UserSession.Delete().
		Where(usersession.TokenEQ(hashSessionToken(token))).
		Exec(dbCtx)

	if err != nil {
		slog.Error("Fallo al eliminar sesión en Ent", "error", err)
		return fmt.Errorf("error al eliminar sesión")
	}

	return nil
}

func (r *userRepository) DeleteExpiredSessions(ctx context.Context) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Purgando sesiones expiradas en Ent")

	_, err := r.client.UserSession.Delete().
		Where(usersession.ExpiresAtLT(time.Now())).
		Exec(dbCtx)

	if err != nil {
		slog.Error("Fallo al purgar sesiones expiradas en Ent", "error", err)
		return fmt.Errorf("error al purgar sesiones expiradas")
	}

	return nil
}

func (r *userRepository) GetUserTypeCode(ctx context.Context, userTypeID int64) (string, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando tipo de usuario por ID en Ent", "user_type_id", userTypeID)

	ut, err := r.client.UserType.Query().
		Where(usertype.IDEQ(int(userTypeID))).
		Only(dbCtx)

	if err != nil {
		if ent.IsNotFound(err) {
			slog.Warn("Tipo de usuario no encontrado en Ent", "user_type_id", userTypeID)
			return "", fmt.Errorf("tipo de usuario no encontrado: %w", err)
		}
		slog.Error("Error al consultar tipo de usuario en Ent", "error", err, "user_type_id", userTypeID)
		return "", fmt.Errorf("error al consultar tipo de usuario")
	}

	return ut.Code, nil
}

func (r *userRepository) DeleteUserSessions(ctx context.Context, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info("Eliminando todas las sesiones del usuario en Ent", "user_id", userID)

	_, err := r.client.UserSession.Delete().
		Where(usersession.UserID(userID)).
		Exec(dbCtx)

	if err != nil {
		slog.Error("Fallo al eliminar sesiones de usuario en Ent", "error", err, "user_id", userID)
		return fmt.Errorf("error al eliminar sesiones del usuario")
	}

	return nil
}

func (r *userRepository) ExtendSession(ctx context.Context, token string, newExpiresAt time.Time) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Extendiendo expiración de sesión activa en Ent")

	_, err := r.client.UserSession.Update().
		Where(usersession.TokenEQ(hashSessionToken(token))).
		SetExpiresAt(newExpiresAt).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al extender expiración de sesión en Ent", "error", err)
		return fmt.Errorf("error al extender sesión")
	}

	return nil
}

// Helper to convert Ent User to Domain User
func toDomainUser(u *ent.User) *domain.User {
	if u == nil {
		return nil
	}
	var middleName, secondLastName, docNum, ip, country *string
	if u.MiddleName != "" {
		middleName = &u.MiddleName
	}
	if u.SecondLastName != "" {
		secondLastName = &u.SecondLastName
	}
	if u.DocumentNumber != "" {
		docNum = &u.DocumentNumber
	}
	if u.RegistrationIP != "" {
		ip = &u.RegistrationIP
	}
	if u.Country != "" {
		country = &u.Country
	}
	var docTypeID *int64
	if u.DocumentTypeID != 0 {
		docTypeID = &u.DocumentTypeID
	}
	var deletedAt *time.Time
	if !u.DeletedAt.IsZero() {
		deletedAt = &u.DeletedAt
	}

	return &domain.User{
		ID:             int64(u.ID),
		Email:          u.Email,
		Username:       u.Username,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		MiddleName:     middleName,
		SecondLastName: secondLastName,
		AvatarIndex:    u.AvatarIndex,
		UserTypeID:     u.UserTypeID,
		DocumentTypeID: docTypeID,
		DocumentNumber: docNum,
		PasswordHash:   u.PasswordHash,
		RegistrationIP: ip,
		Country:        country,
		DeletedAt:      deletedAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}
