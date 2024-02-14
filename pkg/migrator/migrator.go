package migrator

import (
	"database/sql"
	"github.com/dlomanov/go-diploma-tpl/migrations"
	"github.com/go-errors/errors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lopezator/migrator"
	"go.uber.org/zap"
)

const migrationTable = "__migrations"

func Migrate(databaseURI string, sugar *zap.SugaredLogger) error {
	db, err := sql.Open("pgx", databaseURI)
	if err != nil {
		return errors.New(err)
	}
	defer func(db *sql.DB) { _ = db.Close() }(db)
	if err = db.Ping(); err != nil {
		return errors.New(err)
	}

	ms, err := getMigrations()
	if err != nil {
		return errors.New(err)
	}

	logger := migrator.LoggerFunc(func(msg string, args ...interface{}) { sugar.Info(msg, args) })
	m, err := migrator.New(
		ms,
		migrator.WithLogger(logger),
		migrator.TableName(migrationTable),
	)
	if err != nil {
		return errors.New(err)
	}
	return m.Migrate(db)
}

func getMigrations() (migrator.Option, error) {
	ms, err := migrations.GetMigrations()
	if err != nil {
		return nil, err
	}

	result := make([]any, len(ms))
	for i, m := range ms {
		if m.NoTx {
			result[i] = newMigrationNoTx(m.Name, m.Query)
			continue
		}
		result[i] = newMigration(m.Name, m.Query)
	}

	return migrator.Migrations(result...), nil
}

func newMigration(name, query string) *migrator.Migration {
	return &migrator.Migration{
		Name: name,
		Func: func(tx *sql.Tx) error {
			if _, err := tx.Exec(query); err != nil {
				return err
			}
			return nil
		},
	}
}

func newMigrationNoTx(name, query string) *migrator.MigrationNoTx {
	return &migrator.MigrationNoTx{
		Name: name,
		Func: func(tx *sql.DB) error {
			if _, err := tx.Exec(query); err != nil {
				return err
			}
			return nil
		},
	}
}
