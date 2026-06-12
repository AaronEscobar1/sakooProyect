package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// hashSessionToken deriva un hash SHA-256 (hex) del token de sesión.
// Solo se almacena el hash en BD: si la base de datos se filtra, los JWT no son reutilizables.
func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// userRepository implementa la interfaz domain.UserRepository para PostgreSQL.
type userRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository crea una nueva instancia del repositorio de usuarios.
func NewUserRepository(db *pgxpool.Pool) domain.UserRepository {
	return &userRepository{
		db: db,
	}
}

// Create inserta un nuevo usuario en la base de datos y almacena el ID numérico generado.
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Creando registro de usuario en base de datos", "email", user.Email)

	query := `
		INSERT INTO users (
			email, 
			username,
			first_name, 
			last_name, 
			middle_name,
			second_last_name,
			avatar_index, 
			user_type_id, 
			document_type_id, 
			document_number, 
			password_hash, 
			registration_ip,
			country,
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		RETURNING id;
	`

	// Ejecutar consulta y capturar el ID numérico autoincremental (BIGSERIAL)
	err := r.db.QueryRow(dbCtx, query,
		user.Email,
		user.Username,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.SecondLastName,
		user.AvatarIndex,
		user.UserTypeID,
		user.DocumentTypeID,
		user.DocumentNumber,
		user.PasswordHash,
		user.RegistrationIP,
		user.Country,
	).Scan(&user.ID)

	if err != nil {
		slog.Error("Fallo al insertar usuario en PostgreSQL", "error", err, "email", user.Email)

		// Detectar violación de unicidad (23505) y devolver un error de dominio tipado,
		// sin propagar el texto interno de PostgreSQL hacia la capa API.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "username") {
				return domain.ErrUsernameTaken
			}
			return domain.ErrEmailTaken
		}

		return fmt.Errorf("error al guardar usuario en base de datos")
	}

	slog.Info("Usuario registrado exitosamente en base de datos", "id", user.ID, "email", user.Email)
	return nil
}

// FindByEmail busca un usuario activo en base de datos por su correo electrónico.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Buscando usuario por correo electrónico", "email", email)

	query := `
		SELECT 
			id, 
			email, 
			username,
			first_name, 
			last_name, 
			middle_name,
			second_last_name,
			avatar_index, 
			user_type_id, 
			document_type_id, 
			document_number, 
			password_hash, 
			registration_ip,
			country,
			deleted_at,
			created_at, 
			updated_at 
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL;
	`

	var u domain.User
	err := r.db.QueryRow(dbCtx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.FirstName,
		&u.LastName,
		&u.MiddleName,
		&u.SecondLastName,
		&u.AvatarIndex,
		&u.UserTypeID,
		&u.DocumentTypeID,
		&u.DocumentNumber,
		&u.PasswordHash,
		&u.RegistrationIP,
		&u.Country,
		&u.DeletedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("Usuario no encontrado o inactivo en base de datos", "email", email)
			return nil, fmt.Errorf("usuario no encontrado: %w", pgx.ErrNoRows)
		}
		slog.Error("Error al consultar usuario por email en PostgreSQL", "error", err, "email", email)
		return nil, fmt.Errorf("error de consulta en base de datos")
	}

	return &u, nil
}

// FindByID busca un usuario activo en base de datos por su ID de base de datos.
func (r *userRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Buscando usuario por ID", "id", id)

	query := `
		SELECT 
			id, 
			email, 
			username,
			first_name, 
			last_name, 
			middle_name,
			second_last_name,
			avatar_index, 
			user_type_id, 
			document_type_id, 
			document_number, 
			password_hash, 
			registration_ip,
			country,
			deleted_at,
			created_at, 
			updated_at 
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL;
	`

	var u domain.User
	err := r.db.QueryRow(dbCtx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.Username,
		&u.FirstName,
		&u.LastName,
		&u.MiddleName,
		&u.SecondLastName,
		&u.AvatarIndex,
		&u.UserTypeID,
		&u.DocumentTypeID,
		&u.DocumentNumber,
		&u.PasswordHash,
		&u.RegistrationIP,
		&u.Country,
		&u.DeletedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("Usuario no encontrado o inactivo en base de datos", "id", id)
			return nil, fmt.Errorf("usuario no encontrado: %w", pgx.ErrNoRows)
		}
		slog.Error("Error al consultar usuario por ID en PostgreSQL", "error", err, "id", id)
		return nil, fmt.Errorf("error de consulta en base de datos")
	}

	return &u, nil
}

// SoftDelete realiza un borrado lógico estableciendo deleted_at a la fecha actual.
func (r *userRepository) SoftDelete(ctx context.Context, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info("Ejecutando borrado lógico de usuario", "user_id", userID)

	query := `
		UPDATE users 
		SET deleted_at = NOW(), updated_at = NOW() 
		WHERE id = $1 AND deleted_at IS NULL;
	`
	res, err := r.db.Exec(dbCtx, query, userID)
	if err != nil {
		slog.Error("Fallo al ejecutar soft delete del usuario en PostgreSQL", "error", err, "user_id", userID)
		return fmt.Errorf("error al eliminar lógicamente al usuario")
	}

	if res.RowsAffected() == 0 {
		slog.Warn("El usuario no existe o ya ha sido eliminado lógicamente", "user_id", userID)
		return fmt.Errorf("usuario no encontrado o ya eliminado")
	}

	slog.Info("Usuario eliminado lógicamente de forma exitosa", "user_id", userID)
	return nil
}

// UpdatePassword actualiza la contraseña de un usuario en base de datos.
func (r *userRepository) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info("Actualizando contraseña de usuario en base de datos", "user_id", userID)

	query := `
		UPDATE users 
		SET password_hash = $1, updated_at = NOW() 
		WHERE id = $2 AND deleted_at IS NULL;
	`
	res, err := r.db.Exec(dbCtx, query, passwordHash, userID)
	if err != nil {
		slog.Error("Fallo al actualizar la contraseña del usuario en PostgreSQL", "error", err, "user_id", userID)
		return fmt.Errorf("error al actualizar la contraseña del usuario")
	}

	if res.RowsAffected() == 0 {
		slog.Warn("El usuario no existe o ha sido eliminado lógicamente", "user_id", userID)
		return fmt.Errorf("usuario no encontrado o ya eliminado")
	}

	slog.Info("Contraseña del usuario actualizada exitosamente", "user_id", userID)
	return nil
}

// GetPasswordHistory obtiene los últimos 5 hashes de contraseña del historial del usuario.
func (r *userRepository) GetPasswordHistory(ctx context.Context, userID int64) ([]string, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando historial de contraseñas de usuario", "user_id", userID)

	query := `
		SELECT password_hash 
		FROM user_passwords_history 
		WHERE user_id = $1 
		ORDER BY created_at DESC, id DESC 
		LIMIT 5;
	`
	rows, err := r.db.Query(dbCtx, query, userID)
	if err != nil {
		slog.Error("Fallo al obtener historial de contraseñas", "error", err, "user_id", userID)
		return nil, fmt.Errorf("error al obtener historial de contraseñas")
	}
	defer rows.Close()

	var history []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			slog.Error("Fallo al escanear fila de historial de contraseñas", "error", err)
			return nil, fmt.Errorf("error al escanear historial de contraseñas")
		}
		history = append(history, hash)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de historial de contraseñas")
	}

	return history, nil
}

// AddPasswordHistory registra un nuevo hash de contraseña en el historial del usuario y mantiene únicamente los últimos 5 registros de forma atómica.
func (r *userRepository) AddPasswordHistory(ctx context.Context, userID int64, passwordHash string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando hash de contraseña en historial con límite de 5 registros", "user_id", userID)

	// Ejecutar en una transacción para garantizar consistencia atómica
	tx, err := r.db.Begin(dbCtx)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción de historial de contraseñas")
	}
	defer tx.Rollback(dbCtx)

	insertQuery := `
		INSERT INTO user_passwords_history (user_id, password_hash, created_at)
		VALUES ($1, $2, NOW());
	`
	_, err = tx.Exec(dbCtx, insertQuery, userID, passwordHash)
	if err != nil {
		slog.Error("Fallo al insertar en user_passwords_history", "error", err, "user_id", userID)
		return fmt.Errorf("error al registrar en historial de contraseñas")
	}

	// Borrar registros de historial más antiguos de los últimos 5
	deleteQuery := `
		DELETE FROM user_passwords_history 
		WHERE id NOT IN (
			SELECT id FROM user_passwords_history 
			WHERE user_id = $1 
			ORDER BY created_at DESC, id DESC 
			LIMIT 5
		) AND user_id = $1;
	`
	_, err = tx.Exec(dbCtx, deleteQuery, userID)
	if err != nil {
		slog.Error("Fallo al limpiar historial de contraseñas excedente", "error", err, "user_id", userID)
		return fmt.Errorf("error al limpiar historial de contraseñas")
	}

	if err := tx.Commit(dbCtx); err != nil {
		return fmt.Errorf("error al confirmar transacción de historial de contraseñas")
	}

	slog.Info("Hash de contraseña insertado y limpiado en historial con éxito", "user_id", userID)
	return nil
}

// SearchUsers busca usuarios cuyo username comience con el patrón indicado (case-insensitive), limitado y ordenado.
func (r *userRepository) SearchUsers(ctx context.Context, query string, limit int) ([]domain.UserSearchResult, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Buscando usuarios en la base de datos", "query", query, "limit", limit)

	// Búsqueda de tipo "Empieza con" (Ej: 'jos%') ordenada alfabéticamente
	sqlQuery := `
		SELECT id, username, first_name, last_name, avatar_index
		FROM users
		WHERE username ILIKE $1 AND deleted_at IS NULL
		ORDER BY username ASC
		LIMIT $2
	`

	// El patrón de búsqueda debe ser query + "%"
	pattern := query + "%"

	rows, err := r.db.Query(dbCtx, sqlQuery, pattern, limit)
	if err != nil {
		slog.Error("Fallo al buscar usuarios en PostgreSQL", "error", err, "query", query)
		return nil, fmt.Errorf("error al buscar usuarios")
	}
	defer rows.Close()

	var results []domain.UserSearchResult
	for rows.Next() {
		var id int64
		var username, firstName, lastName string
		var avatarIndex int

		err := rows.Scan(&id, &username, &firstName, &lastName, &avatarIndex)
		if err != nil {
			slog.Error("Fallo al escanear resultado de búsqueda de usuario", "error", err)
			return nil, fmt.Errorf("error al escanear resultado de búsqueda")
		}

		// Construir displayName y avatarURL de forma limpia y profesional
		displayName := fmt.Sprintf("%s %s", firstName, lastName)
		avatarURL := fmt.Sprintf("https://sakoo-public-assets.s3.amazonaws.com/avatars/avatar_%d.png", avatarIndex)

		results = append(results, domain.UserSearchResult{
			ID:          id,
			Username:    username,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
		})
	}

	if results == nil {
		results = []domain.UserSearchResult{}
	}

	return results, nil
}

// CreateSession inserta una nueva sesión en security.user_sessions.
func (r *userRepository) CreateSession(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Insertando nueva sesión de usuario en base de datos", "user_id", userID)

	query := `
		INSERT INTO user_sessions (user_id, token, expires_at)
		VALUES ($1, $2, $3);
	`
	_, err := r.db.Exec(dbCtx, query, userID, hashSessionToken(token), expiresAt)
	if err != nil {
		slog.Error("Fallo al crear sesión de usuario en PostgreSQL", "error", err, "user_id", userID)
		return fmt.Errorf("error al crear sesión")
	}

	return nil
}

// ValidateSession comprueba si existe una sesión válida y vigente para el token dado.
func (r *userRepository) ValidateSession(ctx context.Context, token string) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Validando sesión de usuario en base de datos")

	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM user_sessions 
			WHERE token = $1 AND expires_at > NOW()
		);
	`
	var valid bool
	err := r.db.QueryRow(dbCtx, query, hashSessionToken(token)).Scan(&valid)
	if err != nil {
		slog.Error("Fallo al validar sesión en PostgreSQL", "error", err)
		return false, fmt.Errorf("error al validar sesión")
	}

	return valid, nil
}

// DeleteSession elimina una sesión de la base de datos (logout).
func (r *userRepository) DeleteSession(ctx context.Context, token string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Eliminando sesión de usuario de la base de datos (logout)")

	query := `
		DELETE FROM user_sessions
		WHERE token = $1;
	`
	_, err := r.db.Exec(dbCtx, query, hashSessionToken(token))
	if err != nil {
		slog.Error("Fallo al eliminar sesión en PostgreSQL", "error", err)
		return fmt.Errorf("error al eliminar sesión")
	}

	return nil
}

// DeleteExpiredSessions elimina todas las sesiones expiradas en base de datos.
func (r *userRepository) DeleteExpiredSessions(ctx context.Context) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Purgando sesiones expiradas de la base de datos")

	query := `
		DELETE FROM user_sessions 
		WHERE expires_at < NOW();
	`
	_, err := r.db.Exec(dbCtx, query)
	if err != nil {
		slog.Error("Fallo al purgar sesiones expiradas en PostgreSQL", "error", err)
		return fmt.Errorf("error al purgar sesiones expiradas")
	}

	return nil
}

// GetUserTypeCode obtiene el código del tipo de usuario (ej: 'ADMIN', 'CUSTOMER') desde catalogs.user_type.
func (r *userRepository) GetUserTypeCode(ctx context.Context, userTypeID int64) (string, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando tipo de usuario por ID", "user_type_id", userTypeID)

	query := `SELECT code FROM catalogs.user_type WHERE id = $1;`

	var code string
	err := r.db.QueryRow(dbCtx, query, userTypeID).Scan(&code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("Tipo de usuario no encontrado en catálogo", "user_type_id", userTypeID)
			return "", fmt.Errorf("tipo de usuario no encontrado: %w", pgx.ErrNoRows)
		}
		slog.Error("Error al consultar tipo de usuario en PostgreSQL", "error", err, "user_type_id", userTypeID)
		return "", fmt.Errorf("error al consultar tipo de usuario")
	}

	return code, nil
}

// DeleteUserSessions elimina todas las sesiones activas asociadas a un ID de usuario.
func (r *userRepository) DeleteUserSessions(ctx context.Context, userID int64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info("Eliminando todas las sesiones del usuario", "user_id", userID)

	query := `
		DELETE FROM user_sessions 
		WHERE user_id = $1;
	`
	_, err := r.db.Exec(dbCtx, query, userID)
	if err != nil {
		slog.Error("Fallo al eliminar sesiones de usuario en PostgreSQL", "error", err, "user_id", userID)
		return fmt.Errorf("error al eliminar sesiones del usuario")
	}

	return nil
}

// ExtendSession actualiza la fecha de expiración de una sesión específica (expiración deslizante).
func (r *userRepository) ExtendSession(ctx context.Context, token string, newExpiresAt time.Time) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Extendiendo expiración de sesión activa")

	query := `
		UPDATE user_sessions
		SET expires_at = $1
		WHERE token = $2;
	`
	_, err := r.db.Exec(dbCtx, query, newExpiresAt, hashSessionToken(token))
	if err != nil {
		slog.Error("Fallo al extender expiración de sesión en PostgreSQL", "error", err)
		return fmt.Errorf("error al extender sesión")
	}

	return nil
}


