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

	"github.com/google/uuid"
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
		spec.StartDate.ToStringISO(),
		spec.EndDate.ToStringISO(),
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
	query := `
	    SELECT
	        id,
		    service_name,
		    price,
		    user_id,
		    start_date::text,
		    end_date::text
		FROM subscription
		WHERE id = $1
	`

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
	start, err := model.DateFromStringISO(startDate)
	if err != nil {
		s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting start date: %w", err))
		return model.Subscription{}, fmt.Errorf("%s: getting start date: %w", op, err)
	}
	subscription.StartDate = start

	// 2.2.End date
	end, err := model.DateFromStringISO(endDate)
	if err != nil {
		s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting end date: %w", err))
		return model.Subscription{}, fmt.Errorf("%s: getting end date: %w", op, err)
	}
	subscription.EndDate = end

	return subscription, nil
}

func (s *PostgresStorage) UpdateSubscription(id int64, newServiceName string, newPrice int, newStart, newEnd model.Date) error {
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

	// 2.Prepare query in according with optional end_date value
	query := "UPDATE subscription SET service_name = $1, price = $2, start_date = $3"
	args := []interface{}{newServiceName, newPrice, newStart.ToStringISO()}

	if !(newEnd.Month == 0 && newEnd.Year == 0) {
		query += ", end_date = $4 WHERE id = $5"
		args = append(args, newEnd.ToStringISO())
	} else {
		query += " WHERE id = $4"
	}
	args = append(args, id)

	// 3.Run
	res, err = tx.Exec(ctx, query, args...)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return err
	}

	// 4.Check if was updated and commit in case of success
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

func (s *PostgresStorage) GetSubscriptions(limit, offset *int) ([]model.Subscription, error) {
	const op = "storage.postgres.GetSubscriptions"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	ctx := context.Background()

	// 1.Validation
	if limit != nil && offset == nil {
		s.logger.Error(loggerMsg, "details", "no offset value while limit is set")
		return []model.Subscription{}, errors.New("no offset value while limit is set")
	} else if limit == nil && offset != nil {
		s.logger.Error(loggerMsg, "details", "no limit value while offset is set")
		return []model.Subscription{}, errors.New("no limit value while offset is set")
	}

	// 2.Prepare and exec
	query := "SELECT id, service_name, price, user_id, start_date::text, end_date::text FROM subscription"
	args := []interface{}{}

	if limit != nil {
		query += " LIMIT $1 OFFSET $2"
		args = append(args, *limit, *offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return []model.Subscription{}, fmt.Errorf("%s: exec statement: %w", op, err)
	}

	// 3.Parse and get data
	subscriptions, err := s.getSubscriptionsFromPgRows(&loggerMsg, op, rows)
	if err != nil {
		return []model.Subscription{}, err
	}

	return subscriptions, nil
}

func (s *PostgresStorage) FilterSubscriptions(startDate, endDate model.Date, userId uuid.UUID, serviceName *string) ([]model.Subscription, error) {
	const op = "storage.postgres.FilterSubscriptions"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	ctx := context.Background()

	// 2.Prepare and exec
	query := `
		SELECT
			id,
			service_name,
			price,
			user_id,
			start_date::text,
			end_date::text
		FROM subscription
		WHERE start_date > $1 AND end_date < $2
	`
	args := []interface{}{startDate.ToStringISO(), endDate.ToStringISO()}

	if userId != uuid.Nil {
		args = append(args, userId.String())
		query += fmt.Sprintf(" AND user_id = $%d", len(args))
	}

	if serviceName != nil {
		args = append(args, *serviceName)
		query += fmt.Sprintf(" AND user_id = $%d", len(args))
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return []model.Subscription{}, fmt.Errorf("%s: exec statement: %w", op, err)
	}

	// 3.Parse and get data
	subscriptions, err := s.getSubscriptionsFromPgRows(&loggerMsg, op, rows)
	if err != nil {
		return []model.Subscription{}, err
	}

	return subscriptions, nil
}

func (s *PostgresStorage) getSubscriptionsFromPgRows(loggerMsg *string, op string, rows pgx.Rows) ([]model.Subscription, error) {
	var subscriptions []model.Subscription

	for rows.Next() {
		var sub model.Subscription

		var startDate string
		var endDate string

		err := rows.Scan(
			&sub.ID,
			&sub.ServiceName,
			&sub.Price,
			&sub.UserID,
			&startDate,
			&endDate,
		)
		if err != nil {
			s.logger.Error(*loggerMsg, "details", fmt.Errorf("error while parsing db data: %w", err))
			return nil, fmt.Errorf("%s: scan row: %w", op, err)
		}

		// Start date
		start, err := model.DateFromStringISO(startDate)
		if err != nil {
			s.logger.Error(*loggerMsg, "details", fmt.Errorf("error while getting start date: %w", err))
			return []model.Subscription{}, fmt.Errorf("%s: getting start date: %w", op, err)
		}
		sub.StartDate = start

		// End date
		end, err := model.DateFromStringISO(endDate)
		if err != nil {
			s.logger.Error(*loggerMsg, "details", fmt.Errorf("error while getting end date: %w", err))
			return []model.Subscription{}, fmt.Errorf("%s: getting end date: %w", op, err)
		}
		sub.EndDate = end

		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}
