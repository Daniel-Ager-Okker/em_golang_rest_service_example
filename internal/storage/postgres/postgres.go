package pg

import (
	"context"
	"em_golang_rest_service_example/internal/config"
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	pgUserEnv  = "PG_USER"
	pgUserPass = "PG_PASS"

	pgErrConstraintUnique = "23505"
)

type PostgresStorage struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

// Construct Postgres storage for test purposes
func newStorage(logger *slog.Logger, pool *pgxpool.Pool) PostgresStorage {
	return PostgresStorage{logger: logger, pool: pool}
}

// Construct Postgres storage
func NewStorage(cfg *config.StorageCfg, logger *slog.Logger) (PostgresStorage, error) {
	const op = "storage.postgres.NewStorage"

	// 1.Construct pg URL due to two parts of data: open (from yaml) and confidential (from env)
	user, ok := os.LookupEnv(pgUserEnv)
	if !ok {
		return PostgresStorage{}, fmt.Errorf("%s: no value for %s env", op, pgUserEnv)
	}

	pass, ok := os.LookupEnv(pgUserPass)
	if !ok {
		return PostgresStorage{}, fmt.Errorf("%s: no value for %s env", op, pgUserPass)
	}

	pgUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, pass, cfg.PgHost, cfg.PgPort, cfg.PgDbName)

	// 2.Create driver objects

	poolCfg, err := pgxpool.ParseConfig(pgUrl)
	if err != nil {
		return PostgresStorage{}, fmt.Errorf("%s: %w", op, err)
	}

	poolCfg.MaxConns = int32(cfg.PgMaxPoolSize)

	var pool *pgxpool.Pool
	for i := cfg.PgConnectionAttempts; i > 0; i-- {
		pool, err = pgxpool.NewWithConfig(context.Background(), poolCfg)
		if err == nil {
			return PostgresStorage{logger: logger, pool: pool}, nil
		}

		fmt.Printf("%s: trying reconnect", op)

		time.Sleep(cfg.PgConnectionTimeout)
	}

	return PostgresStorage{}, errors.New("connection attempts timed out")
}

// Close DB connection
func (s *PostgresStorage) Close() {
	s.pool.Close()
}

func (s *PostgresStorage) CreateSubscription(spec model.SubscriptionSpec) (int64, error) {
	const op = "storage.postgres.CreateSubscription"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	// 1.Prepare transaction
	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return 0, fmt.Errorf("%s: prepare transaction: %w", op, err)
	}

	defer tx.Rollback(ctx)

	// 2.Run transaction
	query := `
	    INSERT INTO subscription (service_name,price,user_id,start_date,end_date)
		values ($1,$2,$3,$4,$5)
		RETURNING id
	`

	var idStr string
	err = tx.QueryRow(
		ctx, query,
		spec.ServiceName,
		spec.Price,
		spec.UserID.String(),
		spec.StartDate.ToString(),
		spec.EndDate.ToString(),
	).Scan(&idStr)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgErrConstraintUnique {
			s.logger.Error(loggerMsg, "details", storage.ErrSubscriptionExists)
			return 0, fmt.Errorf("%s: %w", op, storage.ErrSubscriptionExists)
		}

		s.logger.Error(loggerMsg, "details", err)
		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return 0, fmt.Errorf("%s: failed to get id as integer: %w", op, err)
	}

	// 3.Commit changes
	err = tx.Commit(ctx)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return 0, fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return id, nil
}

func (s *PostgresStorage) GetSubscription(id int64) (model.Subscription, error) {
	const op = "storage.postgres.GetSubscription"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	ctx := context.Background()

	// 1.Run query
	query := "SELECT * FROM subscription WHERE id = $1"

	row := s.pool.QueryRow(ctx, query, id)

	var subscription model.Subscription

	var startDate string
	var endDate string

	err := row.Scan(
		&subscription.ID,
		&subscription.ServiceName,
		&subscription.Price,
		&subscription.UserID,
		&startDate,
		&endDate,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error(loggerMsg, "details", storage.ErrSubscribtionNotFound)
		return model.Subscription{}, storage.ErrSubscribtionNotFound
	}
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return model.Subscription{}, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	// 2.Return subscription model data

	// 2.1.Start date
	start, err := model.DateFromString(startDate)
	if err != nil {
		s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting start date: %w", err))
		return model.Subscription{}, fmt.Errorf("%s: getting start date: %w", op, err)
	}
	subscription.StartDate = start

	// 2.2.End date
	end, err := model.DateFromString(endDate)
	if err != nil {
		s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting end date: %w", err))
		return model.Subscription{}, fmt.Errorf("%s: getting end date: %w", op, err)
	}
	subscription.EndDate = end

	return subscription, nil
}

func (s *PostgresStorage) UpdateSubscription(id int64, newPrice int, newEnd model.Date) error {
	const op = "storage.postgres.UpdateSubscription"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	var res pgconn.CommandTag

	// 1.Prepare transaction
	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return fmt.Errorf("%s: prepare transaction: %w", op, err)
	}

	defer tx.Rollback(ctx)

	// 2.Run needed query in according with optional end_date value
	if newEnd.Month == 0 && newEnd.Year == 0 {
		// Prepare
		query := "UPDATE subscription SET price = $1 WHERE id = $2"

		// Run
		res, err = tx.Exec(ctx, query, newPrice, id)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return err
		}

	} else {
		// Prepare
		query := "UPDATE subscription SET price = $1, end_date = $2 WHERE id = $3"

		// Run
		res, err = tx.Exec(ctx, query, newPrice, newEnd.ToString(), id)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return err
		}
	}

	// 3.Check if was updated and commit in case of success
	if res.RowsAffected() == 0 {
		s.logger.Error(loggerMsg, "details", storage.ErrSubscribtionNotFound)
		return storage.ErrSubscribtionNotFound
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}

func (s *PostgresStorage) DeleteSubscription(id int64) error {
	const op = "storage.postgres.DeleteSubscription"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	ctx := context.Background()

	// 1.Prepare transaction
	query := "DELETE FROM subscription WHERE id = $1"

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return fmt.Errorf("%s: prepare transaction: %w", op, err)
	}

	defer tx.Rollback(ctx)

	// 2.Run
	res, err := tx.Exec(ctx, query, id)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return err
	}

	// 3.Check if was deleted and commit in case of success
	if res.RowsAffected() == 0 {
		s.logger.Error(loggerMsg, "details", storage.ErrSubscribtionNotFound)
		return storage.ErrSubscribtionNotFound
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}

func (s *PostgresStorage) GetSubscriptions() ([]model.Subscription, error) {
	const op = "storage.postgres.GetSubscriptions"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	ctx := context.Background()

	// 1.Run query
	query := "SELECT * FROM subscription"

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return []model.Subscription{}, fmt.Errorf("%s: exec statement: %w", op, err)
	}

	// 3.Parse and get data
	var subscriptions []model.Subscription

	for rows.Next() {
		var sub model.Subscription

		var startDate string
		var endDate string

		err = rows.Scan(
			&sub.ID,
			&sub.ServiceName,
			&sub.Price,
			&sub.UserID,
			&startDate,
			&endDate,
		)
		if err != nil {
			s.logger.Error(loggerMsg, "details", fmt.Errorf("error while parsing db data: %w", err))
			return nil, fmt.Errorf("%s: scan row: %w", op, err)
		}

		// 3.1.Start date
		start, err := model.DateFromString(startDate)
		if err != nil {
			s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting start date: %w", err))
			return []model.Subscription{}, fmt.Errorf("%s: getting start date: %w", op, err)
		}
		sub.StartDate = start

		// 3.2.End date
		end, err := model.DateFromString(endDate)
		if err != nil {
			s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting end date: %w", err))
			return []model.Subscription{}, fmt.Errorf("%s: getting end date: %w", op, err)
		}
		sub.EndDate = end

		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}
