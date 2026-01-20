package sqlite

import (
	"database/sql"
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Helper function to create and set up a new in-memory SQLite database
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// 1.Open an in-memory database using a URI filename with mode=memory and cache=shared
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// 2.Migrations
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS subscription(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			service_name TEXT NOT NULL,
			price INTEGER NOT NULL,
			user_id TEXT NOT NULL,
			start_date TEXT NOT NULL CHECK (
				start_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND
				CAST(substr(start_date, 1, 4) AS INTEGER) BETWEEN 2000 AND 2100 AND
				CAST(substr(start_date, 6, 2) AS INTEGER) BETWEEN 1 AND 12 AND
				CAST(substr(start_date, 9, 2) AS INTEGER) BETWEEN 1 AND 31
			),
			end_date TEXT NOT NULL CHECK (
				-- Check ISO date format YYYY-MM-DD
				end_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND
				CAST(substr(end_date, 1, 4) AS INTEGER) BETWEEN 2000 AND 2100 AND
				CAST(substr(end_date, 6, 2) AS INTEGER) BETWEEN 1 AND 12 AND
				CAST(substr(end_date, 9, 2) AS INTEGER) BETWEEN 1 AND 31
			),
			CONSTRAINT unique_subscription UNIQUE (service_name, user_id),
			CONSTRAINT check_end_after_start CHECK (end_date > start_date)
		);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create table: %v", err)
	}

	return db
}

func TestCreateSubscription(t *testing.T) {
	// 1.Init
	db := newTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	sqliteStorage := newStorage(db, logger)

	user1, user2 := uuid.New(), uuid.New()

	// 2.Tests
	cases := []struct {
		name           string
		serviceName    string
		price          int
		userId         uuid.UUID
		startDate      model.Date
		endDate        model.Date
		expectedId     int64
		expectedErrMsg string
	}{
		{
			name:        "Success",
			serviceName: "Yandex",
			price:       400,
			userId:      user1,
			startDate:   model.Date{Month: 1, Year: 2026},
			endDate:     model.Date{Month: 2, Year: 2026},
			expectedId:  int64(1),
		},
		{
			name:           "Already exist",
			serviceName:    "Yandex",
			price:          400,
			userId:         user1,
			startDate:      model.Date{Month: 1, Year: 2026},
			endDate:        model.Date{Month: 2, Year: 2026},
			expectedId:     int64(0),
			expectedErrMsg: storage.ErrSubscriptionExists.Error(),
		},
		{
			name:           "End date constraint",
			serviceName:    "Yandex",
			price:          400,
			userId:         user2,
			startDate:      model.Date{Month: 1, Year: 2026},
			endDate:        model.Date{Month: 12, Year: 2025},
			expectedId:     int64(0),
			expectedErrMsg: "constraint",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := model.SubscriptionSpec{
				ServiceName: tc.serviceName,
				Price:       tc.price,
				UserID:      tc.userId,
				StartDate:   tc.startDate,
				EndDate:     tc.endDate,
			}

			id, err := sqliteStorage.CreateSubscription(spec)

			assert.Equal(t, tc.expectedId, id)

			if tc.expectedErrMsg != "" {
				assert.ErrorContains(t, err, tc.expectedErrMsg)
			}
		})
	}
}

func TestGetSubscription(t *testing.T) {
	// 1.Init
	db := newTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	sqliteStorage := newStorage(db, logger)

	// 2.Insert something
	spec := model.SubscriptionSpec{
		ServiceName: "Yandex",
		Price:       400,
		UserID:      uuid.New(),
		StartDate:   model.Date{Month: 1, Year: 2026},
		EndDate:     model.Date{Month: 2, Year: 2026},
	}

	id, err := sqliteStorage.CreateSubscription(spec)
	assert.Nil(t, err)

	// 3.Tests
	cases := []struct {
		name           string
		id             int64
		expectedErrMsg string
	}{
		{
			name: "Success",
			id:   id,
		},
		{
			name:           "Not exist",
			id:             -8,
			expectedErrMsg: storage.ErrSubscribtionNotFound.Error(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subscription, err := sqliteStorage.GetSubscription(tc.id)
			if err == nil {
				assert.Equal(t, subscription.ID, tc.id)
				assert.Equal(t, subscription.ServiceName, spec.ServiceName)
				assert.Equal(t, subscription.Price, spec.Price)
				assert.Equal(t, subscription.UserID, spec.UserID)
				assert.Equal(t, subscription.StartDate, spec.StartDate)
				assert.Equal(t, subscription.EndDate, spec.EndDate)
			} else {
				assert.ErrorContains(t, err, tc.expectedErrMsg)
			}
		})
	}
}

func TestUpdateSubscription(t *testing.T) {
	// 1.Init
	db := newTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	sqliteStorage := newStorage(db, logger)

	// 2.Update some non-existen values
	t.Run("Update non-existen", func(t *testing.T) {
		err := sqliteStorage.UpdateSubscription(532, "Any non-existen", 350, model.Date{Month: 1, Year: 1990}, model.Date{Month: 1, Year: 1991})
		assert.ErrorContains(t, err, storage.ErrSubscribtionNotFound.Error())
	})

	// 3.1.Update existen OK
	t.Run("Update existen OK with end date", func(t *testing.T) {
		spec := model.SubscriptionSpec{
			ServiceName: "Yandex",
			Price:       400,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 1, Year: 2026},
			EndDate:     model.Date{Month: 2, Year: 2026},
		}
		id, _ := sqliteStorage.CreateSubscription(spec)

		err := sqliteStorage.UpdateSubscription(id, "Яндекс", 350, spec.StartDate, model.Date{Month: 1, Year: 2027})
		assert.NoError(t, err)

		subscription, _ := sqliteStorage.GetSubscription(id)
		assert.Equal(t, id, subscription.ID)
		assert.Equal(t, "Яндекс", subscription.ServiceName)
		assert.Equal(t, 350, subscription.Price)
		assert.Equal(t, spec.UserID, subscription.UserID)
		assert.Equal(t, spec.StartDate, subscription.StartDate)
		assert.Equal(t, subscription.EndDate, model.Date{Month: 1, Year: 2027})
	})

	// 3.2.Update existen OK
	t.Run("Update existen OK no end date", func(t *testing.T) {
		spec := model.SubscriptionSpec{
			ServiceName: "Yandex",
			Price:       400,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 1, Year: 2026},
			EndDate:     model.Date{Month: 2, Year: 2026},
		}
		id, _ := sqliteStorage.CreateSubscription(spec)

		err := sqliteStorage.UpdateSubscription(id, spec.ServiceName, 300, spec.StartDate, model.Date{})
		assert.NoError(t, err)

		subscription, _ := sqliteStorage.GetSubscription(id)
		assert.Equal(t, id, subscription.ID)
		assert.Equal(t, spec.ServiceName, subscription.ServiceName)
		assert.Equal(t, 300, subscription.Price)
		assert.Equal(t, spec.UserID, subscription.UserID)
		assert.Equal(t, spec.StartDate, subscription.StartDate)
		assert.Equal(t, spec.EndDate, subscription.EndDate)
	})

	// 4.Update existen FAIL
	t.Run("Update existen end date FAIL", func(t *testing.T) {
		spec := model.SubscriptionSpec{
			ServiceName: "Google",
			Price:       400,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 1, Year: 2026},
			EndDate:     model.Date{Month: 12, Year: 2026},
		}
		id, _ := sqliteStorage.CreateSubscription(spec)

		err := sqliteStorage.UpdateSubscription(id, spec.ServiceName, 500, spec.StartDate, model.Date{Month: 12, Year: 2025})
		assert.ErrorContains(t, err, "constraint")
	})
}

func TestDeleteSubscription(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	sqliteStorage := newStorage(db, logger)

	spec := model.SubscriptionSpec{
		ServiceName: "Wink",
		Price:       300,
		UserID:      uuid.New(),
		StartDate:   model.Date{Month: 3, Year: 2026},
		EndDate:     model.Date{Month: 4, Year: 2027},
	}
	id, _ := sqliteStorage.CreateSubscription(spec)

	// 1.Case non-existen id
	err := sqliteStorage.DeleteSubscription(-532)
	assert.ErrorIs(t, err, storage.ErrSubscribtionNotFound)

	// 2.Case OK
	err = sqliteStorage.DeleteSubscription(id)
	assert.Nil(t, err)

	_, err = sqliteStorage.GetSubscription(id)
	assert.ErrorIs(t, err, storage.ErrSubscribtionNotFound)
}

func TestGetSubscriptions(t *testing.T) {
	// 1.Init
	db := newTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	sqliteStorage := newStorage(db, logger)

	// 2.Prepare
	services := []string{
		"Yandex", "Google", "Netflix", "Wink",
	}
	prices := []int{
		400, 800, 700, 300,
	}

	for i := 0; i < len(services); i++ {
		spec := model.SubscriptionSpec{
			ServiceName: services[i],
			Price:       prices[i],
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 3, Year: 2026},
			EndDate:     model.Date{Month: 5, Year: 2026},
		}
		_, err := sqliteStorage.CreateSubscription(spec)
		assert.Nil(t, err)
	}

	// 2.Tests
	cases := []struct {
		name   string
		limit  *int
		offset *int
		errMsg string
	}{
		{
			name: "Success no limit and offset",
		},
		{
			name:   "Success with limit and offset",
			limit:  intPointerHelper(2),
			offset: intPointerHelper(0),
		},
		{
			name:   "Fail got limit but no offset",
			limit:  intPointerHelper(2),
			errMsg: "no offset value while limit is set",
		},
		{
			name:   "Fail got offset but no limit",
			offset: intPointerHelper(2),
			errMsg: "no limit value while offset is set",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subs, err := sqliteStorage.GetSubscriptions(tc.limit, tc.offset)

			if tc.errMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}

			for i := 0; i < len(subs); i++ {
				assert.Equal(t, subs[i].ID, int64(i+1))
				assert.Equal(t, subs[i].ServiceName, services[i])
				assert.Equal(t, subs[i].Price, prices[i])
			}
		})
	}

}

func intPointerHelper(value int) *int {
	p := new(int)
	*p = value
	return p
}
