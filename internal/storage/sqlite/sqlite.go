package sqlite

import (
	"database/sql"
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mattn/go-sqlite3"
)

type SqliteStorage struct {
	db     *sql.DB
	logger *slog.Logger
}

// Construct SQLite storage for test purposes
func newStorage(db *sql.DB, logger *slog.Logger) SqliteStorage {
	return SqliteStorage{db: db, logger: logger}
}

// Construct SQLite storage
func NewStorage(storagePath *string, logger *slog.Logger) (SqliteStorage, error) {
	const op = "storage.sqlite.NewStorage"

	db, err := sql.Open("sqlite3", *storagePath)
	if err != nil {
		return SqliteStorage{}, fmt.Errorf("%s: %w", op, err)
	}

	return SqliteStorage{db: db, logger: logger}, nil
}

// Close db connection
func (s *SqliteStorage) Close() {
	s.logger.Info("closing database")
	s.db.Close()
}

func (s *SqliteStorage) CreateSubscription(spec model.SubscriptionSpec) (int64, error) {
	const op = "storage.sqlite.CreateSubscription"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	// 1.Prepare query
	query := `
	    INSERT INTO subscription (service_name,price,user_id,start_date,end_date)
		values (?,?,?,?,?)
	`
	stmt, err := s.db.Prepare(query)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return 0, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	// 2.Run it
	startDate := spec.StartDate.ToStringISO()
	endDate := spec.EndDate.ToStringISO()

	res, err := stmt.Exec(spec.ServiceName, spec.Price, spec.UserID, startDate, endDate)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			s.logger.Error(loggerMsg, "details", storage.ErrSubscriptionExists)
			return 0, fmt.Errorf("%s: %w", op, storage.ErrSubscriptionExists)
		}

		s.logger.Error(loggerMsg, "details", err)
		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	// 3.Get created item ID and return it
	id, err := res.LastInsertId()
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *SqliteStorage) GetSubscription(id int64) (model.Subscription, error) {
	const op = "storage.sqlite.GetSubscription"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	// 1.Prepare query
	query := "SELECT * FROM subscription WHERE id = ?"
	stmt, err := s.db.Prepare(query)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return model.Subscription{}, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	// 2.Run it
	var startDate string
	var endDate string

	var subscription model.Subscription

	err = stmt.QueryRow(id).Scan(
		&subscription.ID,
		&subscription.ServiceName,
		&subscription.Price,
		&subscription.UserID,
		&startDate,
		&endDate,
	)
	if errors.Is(err, sql.ErrNoRows) {
		s.logger.Error(loggerMsg, "details", storage.ErrSubscribtionNotFound)
		return model.Subscription{}, storage.ErrSubscribtionNotFound
	}
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return model.Subscription{}, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	// 3.Return subscription model data

	// 3.1.Start date
	start, err := model.DateFromStringISO(startDate)
	if err != nil {
		s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting start date: %w", err))
		return model.Subscription{}, fmt.Errorf("%s: getting start date: %w", op, err)
	}
	subscription.StartDate = start

	// 3.2.End date
	end, err := model.DateFromStringISO(endDate)
	if err != nil {
		s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting end date: %w", err))
		return model.Subscription{}, fmt.Errorf("%s: getting end date: %w", op, err)
	}
	subscription.EndDate = end

	return subscription, nil
}

func (s *SqliteStorage) UpdateSubscription(id int64, newPrice int, newEnd model.Date) error {
	const op = "storage.sqlite.UpdateSubscription"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	var res sql.Result

	// 1.Run needed query in according with optional end_date value
	if newEnd.Month == 0 && newEnd.Year == 0 {
		// Prepare
		query := "UPDATE subscription SET price = ? WHERE id = ?"

		stmt, err := s.db.Prepare(query)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return fmt.Errorf("%s: prepare statement: %w", op, err)
		}

		// Run
		res, err = stmt.Exec(newPrice, id)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return err
		}

	} else {
		query := "UPDATE subscription SET price = ?, end_date = ? WHERE id = ?"

		stmt, err := s.db.Prepare(query)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return fmt.Errorf("%s: prepare statement: %w", op, err)
		}

		// Run
		res, err = stmt.Exec(newPrice, newEnd.ToStringISO(), id)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return err
		}
	}

	// 2.Check if was updated and return corresponding status
	changedRows, err := res.RowsAffected()
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return err
	}
	if changedRows == 0 {
		s.logger.Error(loggerMsg, "details", storage.ErrSubscribtionNotFound)
		return storage.ErrSubscribtionNotFound
	}

	return nil
}

func (s *SqliteStorage) DeleteSubscription(id int64) error {
	const op = "storage.sqlite.DeleteSubscription"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	// 1.Prepare query
	query := `
	    DELETE FROM subscription
		WHERE id = ?
	`

	stmt, err := s.db.Prepare(query)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	// 2.Run it
	res, err := stmt.Exec(id)
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return err
	}

	// 3.Check if was deleted and return corresponding status
	deletedRows, err := res.RowsAffected()
	if err != nil {
		s.logger.Error(loggerMsg, "details", err)
		return err
	}
	if deletedRows == 0 {
		s.logger.Error(loggerMsg, "details", storage.ErrSubscribtionNotFound)
		return storage.ErrSubscribtionNotFound
	}

	return nil
}

func (s *SqliteStorage) GetSubscriptions(limit, offset *int) ([]model.Subscription, error) {
	const op = "storage.sqlite.GetSubscriptions"
	var loggerMsg string = fmt.Sprintf("operation is %s", op)

	// 1.Validation
	if limit != nil && offset == nil {
		s.logger.Error(loggerMsg, "details", "no offset value while limit is set")
		return []model.Subscription{}, errors.New("no offset value while limit is set")
	} else if limit == nil && offset != nil {
		s.logger.Error(loggerMsg, "details", "no limit value while offset is set")
		return []model.Subscription{}, errors.New("no limit value while offset is set")
	}

	// 2.Prepare query and exec needed
	var rows *sql.Rows

	if limit == nil {
		query := "SELECT * FROM subscription"

		stmt, err := s.db.Prepare(query)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return []model.Subscription{}, fmt.Errorf("%s: prepare statement: %w", op, err)
		}

		rows, err = stmt.Query()
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return []model.Subscription{}, fmt.Errorf("%s: exec statement: %w", op, err)
		}
	} else {
		query := "SELECT * FROM subscription LIMIT ? OFFSET ?"

		stmt, err := s.db.Prepare(query)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return []model.Subscription{}, fmt.Errorf("%s: prepare statement: %w", op, err)
		}

		rows, err = stmt.Query(*limit, *offset)
		if err != nil {
			s.logger.Error(loggerMsg, "details", err)
			return []model.Subscription{}, fmt.Errorf("%s: exec statement: %w", op, err)
		}
	}

	// 3.Parse and get data
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
			s.logger.Error(loggerMsg, "details", fmt.Errorf("error while parsing db data: %w", err))
			return nil, fmt.Errorf("%s: scan row: %w", op, err)
		}

		// 3.1.Start date
		start, err := model.DateFromStringISO(startDate)
		if err != nil {
			s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting start date: %w", err))
			return []model.Subscription{}, fmt.Errorf("%s: getting start date: %w", op, err)
		}
		sub.StartDate = start

		// 3.2.End date
		end, err := model.DateFromStringISO(endDate)
		if err != nil {
			s.logger.Error(loggerMsg, "details", fmt.Errorf("error while getting end date: %w", err))
			return []model.Subscription{}, fmt.Errorf("%s: getting end date: %w", op, err)
		}
		sub.EndDate = end

		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}
