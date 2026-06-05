package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectAndMigrate inicializa el pool de conexiones pgxpool, establece el search path opcional y ejecuta las migraciones.
func ConnectAndMigrate(dbURL string, searchPaths []string, migrationPath string) (*pgxpool.Pool, error) {
	redactedURL := redactDBURL(dbURL)
	slog.Info("Intentando conectar a PostgreSQL...", "url", redactedURL)

	if err := ensureDatabaseExists(dbURL); err != nil {
		slog.Warn("No se pudo verificar/crear la base de datos automáticamente (se intentará continuar)", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("error al analizar la configuración de base de datos: %w", err)
	}

	// Ajustes recomendados para producción
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 15 * time.Minute

	// Configurar search_path si se proveen
	if len(searchPaths) > 0 {
		searchPathQuery := fmt.Sprintf("SET search_path TO %s;", strings.Join(searchPaths, ", "))
		config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, searchPathQuery)
			if err != nil {
				slog.Error("Fallo al establecer el search_path en la conexión de PostgreSQL", "error", err)
				return err
			}
			return nil
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error al instanciar el pool de conexiones pgx: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("la base de datos no responde al Ping: %w", err)
	}

	slog.Info("Conexión con PostgreSQL establecida exitosamente", "max_conns", config.MaxConns)

	// Ejecutar migraciones si se provee ruta
	if migrationPath != "" {
		slog.Info("Buscando y aplicando migraciones en la base de datos...", "ruta", migrationPath)
		
		migrationURL := dbURL
		if len(migrationURL) >= 10 && migrationURL[:10] == "postgresql" {
			migrationURL = "postgres" + migrationURL[10:]
		}

		m, err := migrate.New(migrationPath, migrationURL)
		if err != nil {
			return nil, fmt.Errorf("error al inicializar el motor de migraciones: %w", err)
		}
		defer m.Close()

		version, _, errVersion := m.Version()
		if errVersion == nil && version > 1 {
			slog.Warn("Se detectó versión superior en base de datos. Sincronizando de forma preventiva a versión 1...")
			if errForce := m.Force(1); errForce != nil {
				return nil, fmt.Errorf("error al forzar sincronización de versión: %w", errForce)
			}
		}

		if err := m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				slog.Info("La base de datos ya está actualizada, no se aplicaron cambios")
			} else if strings.Contains(err.Error(), "dirty") || strings.Contains(err.Error(), "Dirty") {
				slog.Warn("Se detectó un estado 'dirty' (sucia) en la base de datos de migraciones. Intentando autorecuperación...")
				version, _, errVersion := m.Version()
				if errVersion == nil {
					slog.Info("Forzando versión de base de datos a limpia", "version", version)
					if errForce := m.Force(int(version)); errForce == nil {
						slog.Info("Estado dirty limpiado automáticamente. Reintentando aplicar las migraciones...")
						if errRetry := m.Up(); errRetry == nil || errors.Is(errRetry, migrate.ErrNoChange) {
							slog.Info("Migraciones aplicadas con éxito tras limpiar el estado dirty!")
						} else {
							return nil, fmt.Errorf("error al reintentar migraciones tras forzar limpieza: %w", errRetry)
						}
					} else {
						return nil, fmt.Errorf("no se pudo forzar la limpieza del estado dirty: %w", errForce)
					}
				} else {
					return nil, fmt.Errorf("error al obtener versión actual de la base de datos sucia: %w", errVersion)
				}
			} else {
				return nil, fmt.Errorf("error en la ejecución de la migración: %w", err)
			}
		} else {
			slog.Info("Migraciones aplicadas de manera exitosa en el esquema")
		}
	}

	return pool, nil
}

// ensureDatabaseExists se conecta a la base de datos 'postgres' y crea la base de datos objetivo si no existe.
func ensureDatabaseExists(dbURL string) error {
	u, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("error al analizar la URL de conexión: %w", err)
	}

	targetDB := u.Path
	if len(targetDB) > 0 && targetDB[0] == '/' {
		targetDB = targetDB[1:]
	}

	if targetDB == "" || targetDB == "postgres" {
		return nil
	}

	u.Path = "/postgres"
	postgresURL := u.String()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, postgresURL)
	if err != nil {
		return fmt.Errorf("error al conectar a la base de datos de administración 'postgres': %w", err)
	}
	defer conn.Close(ctx)

	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	err = conn.QueryRow(ctx, query, targetDB).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error al verificar existencia de base de datos en catálogo: %w", err)
	}

	if !exists {
		slog.Info("La base de datos objetivo no existe. Creándola automáticamente...", "database", targetDB)
		
		safeDBName := pgx.Identifier{targetDB}.Sanitize()
		createStmt := fmt.Sprintf("CREATE DATABASE %s", safeDBName)

		_, err = conn.Exec(ctx, createStmt)
		if err != nil {
			return fmt.Errorf("error al ejecutar la creación de base de datos: %w", err)
		}
		slog.Info("Base de datos creada exitosamente en PostgreSQL", "database", targetDB)
	}

	return nil
}

// redactDBURL oculta la contraseña de la base de datos en la cadena de conexión para proteger las credenciales en logs.
func redactDBURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[URL de base de datos inválida]"
	}
	if u.User != nil {
		if _, hasPass := u.User.Password(); hasPass {
			u.User = url.UserPassword(u.User.Username(), "xxxxx")
		}
	}
	return u.String()
}
