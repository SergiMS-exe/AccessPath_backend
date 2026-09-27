package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // driver pgx
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrateUp aplica las migraciones pendientes desde ./db/migrations contra
// la base de datos. Es idempotente: si no hay migraciones nuevas sale sin
// hacer nada. Si la URL es invalida o el esquema/migrations no existen,
// devuelve error y el caller decide si abortar el arranque.
//
// sourceURL acepta una ruta absoluta o relativa al working directory.
// Recomendado: "./db/migrations".
func MigrateUp(databaseURL, sourceURL string) error {
	src, err := iofs.New(os.DirFS(sourceURL), ".")
	if err != nil {
		return fmt.Errorf("migrate source: %w", err)
	}

	// golang-migrate espera el esquema pgx5:// para Postgres con pgx v5.
	migrateURL := "pgx5://" + stripPostgresScheme(databaseURL)

	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("migrate version: %w", err)
	}
	slog.Info("migrate up",
		slog.String("source", sourceURL),
		slog.Uint64("version", uint64(version)),
		slog.Bool("dirty", dirty),
	)
	return nil
}

func stripPostgresScheme(url string) string {
	const prefix = "postgres://"
	const prefix2 = "postgresql://"
	if len(url) > len(prefix) && url[:len(prefix)] == prefix {
		return url[len(prefix):]
	}
	if len(url) > len(prefix2) && url[:len(prefix2)] == prefix2 {
		return url[len(prefix2):]
	}
	return url
}

// ConfirmPoolHealthy es un healthcheck ligero antes de migrar. Si el pool
// no puede hacer un Ping, no tiene sentido intentar migrar.
func ConfirmPoolHealthy(ctx context.Context, db *pgxpool.Pool) error {
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	return nil
}
