package pg

import (
	"context"
	"em_golang_rest_service_example/internal/model"
	"em_golang_rest_service_example/internal/storage"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func newTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// 1.Prepare req
	req := prepareTestDbReq(t)

	// 2.Create DB container
	ctx := context.Background()
	container := createTestDbContainer(t, ctx, &req)

	// 3.Prepare it params
	host, port := getTestDbHostPort(t, ctx, container)

	connStr := fmt.Sprintf(
		"postgres://test:test@%s:%s/testdb?sslmode=disable",
		host, port.Port(),
	)

	// 4.Get PG configuration due to params and create pool object
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// 5.Initialization migrations
	runTestDbInitMigrations(t, ctx, pool)

	// 6.Close it when test is over
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func prepareTestDbReq(t *testing.T) testcontainers.ContainerRequest {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		),
	}
	return req
}

func createTestDbContainer(t *testing.T, ctx context.Context, req *testcontainers.ContainerRequest) testcontainers.Container {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	})

	return container
}

func getTestDbHostPort(t *testing.T, ctx context.Context, cont testcontainers.Container) (string, nat.Port) {
	t.Helper()

	host, err := cont.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	port, err := cont.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}

	return host, port
}

func runTestDbInitMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS subscription(
			id SERIAL PRIMARY KEY,
			service_name TEXT NOT NULL,
			price INTEGER NOT NULL,
			user_id TEXT NOT NULL,
			start_date TEXT NOT NULL CHECK (
				start_date ~ '^[0-9]{2}-[0-9]{4}$' AND
				CAST(SUBSTRING(start_date FROM 1 FOR 2) AS INTEGER) BETWEEN 1 AND 12
			),
			end_date TEXT NOT NULL CHECK (
				end_date ~ '^[0-9]{2}-[0-9]{4}$' AND
				CAST(SUBSTRING(end_date FROM 1 FOR 2) AS INTEGER) BETWEEN 1 AND 12
			),
			CONSTRAINT unique_subscription UNIQUE (service_name, user_id),
			CONSTRAINT check_end_after_start 
				CHECK (
					-- sneaky trick (convert 'MM-YYYY' to 'YYYYMM' and compare integers)
					(
						CAST(SUBSTRING(end_date FROM 4) AS INTEGER) * 100 + 
						CAST(SUBSTRING(end_date FROM 1 FOR 2) AS INTEGER)
					) >
					(
						CAST(SUBSTRING(start_date FROM 4) AS INTEGER) * 100 + 
						CAST(SUBSTRING(start_date FROM 1 FOR 2) AS INTEGER)
					)
				)
		);
	`

	// 1.Get connection from pool
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %s", err.Error())
	}
	defer conn.Release()

	// 2.Try to begin transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction for applying initial migration: %s", err.Error())
	}

	// 3.Try to apply migration
	_, err = tx.Exec(ctx, createTableSQL)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("failed to apply initial migration: %s", err.Error())
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("failed to commit migration: %s", err.Error())
	}
}

func TestCreateSubscription(t *testing.T) {
	pool := newTestDB(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pgStorage := newStorage(logger, pool)

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
			name:           "End date constraint",
			serviceName:    "Yandex",
			price:          400,
			userId:         user2,
			startDate:      model.Date{Month: 1, Year: 2026},
			endDate:        model.Date{Month: 12, Year: 2025},
			expectedId:     int64(0),
			expectedErrMsg: "constraint",
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

			id, err := pgStorage.CreateSubscription(spec)

			assert.Equal(t, tc.expectedId, id)

			if tc.expectedErrMsg != "" {
				assert.ErrorContains(t, err, tc.expectedErrMsg)
			}
		})
	}
}

func TestGetSubscription(t *testing.T) {
	// 1.Init
	pool := newTestDB(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pgStorage := newStorage(logger, pool)

	// 2.Insert something
	spec := model.SubscriptionSpec{
		ServiceName: "Yandex",
		Price:       400,
		UserID:      uuid.New(),
		StartDate:   model.Date{Month: 1, Year: 2026},
		EndDate:     model.Date{Month: 2, Year: 2026},
	}

	id, err := pgStorage.CreateSubscription(spec)
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
			subscription, err := pgStorage.GetSubscription(tc.id)
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
	pool := newTestDB(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pgStorage := newStorage(logger, pool)

	// 2.Update some non-existen values
	t.Run("Update non-existen", func(t *testing.T) {
		err := pgStorage.UpdateSubscription(532, 350, model.Date{Month: 1, Year: 1991})
		assert.ErrorIs(t, err, storage.ErrSubscribtionNotFound)
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
		id, _ := pgStorage.CreateSubscription(spec)

		err := pgStorage.UpdateSubscription(id, 350, model.Date{Month: 1, Year: 2027})
		assert.NoError(t, err)

		subscription, _ := pgStorage.GetSubscription(id)
		assert.Equal(t, subscription.ID, id)
		assert.Equal(t, subscription.ServiceName, spec.ServiceName)
		assert.Equal(t, subscription.Price, 350)
		assert.Equal(t, subscription.UserID, spec.UserID)
		assert.Equal(t, subscription.StartDate, spec.StartDate)
		assert.Equal(t, subscription.EndDate, model.Date{Month: 1, Year: 2027})
	})

	// 3.2.Update existen OK
	t.Run("Update existen OK only price", func(t *testing.T) {
		spec := model.SubscriptionSpec{
			ServiceName: "Yandex",
			Price:       400,
			UserID:      uuid.New(),
			StartDate:   model.Date{Month: 1, Year: 2026},
			EndDate:     model.Date{Month: 2, Year: 2026},
		}
		id, _ := pgStorage.CreateSubscription(spec)

		err := pgStorage.UpdateSubscription(id, 300, model.Date{})
		assert.NoError(t, err)

		subscription, _ := pgStorage.GetSubscription(id)
		assert.Equal(t, subscription.ID, id)
		assert.Equal(t, subscription.ServiceName, spec.ServiceName)
		assert.Equal(t, subscription.Price, 300)
		assert.Equal(t, subscription.UserID, spec.UserID)
		assert.Equal(t, subscription.StartDate, spec.StartDate)
		assert.Equal(t, subscription.EndDate, spec.EndDate)
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
		id, _ := pgStorage.CreateSubscription(spec)

		err := pgStorage.UpdateSubscription(id, 500, model.Date{Month: 12, Year: 2025})
		assert.ErrorContains(t, err, "constraint")
	})
}

func TestDeleteSubscription(t *testing.T) {
	// 1.Init
	pool := newTestDB(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pgStorage := newStorage(logger, pool)

	spec := model.SubscriptionSpec{
		ServiceName: "Wink",
		Price:       300,
		UserID:      uuid.New(),
		StartDate:   model.Date{Month: 3, Year: 2026},
		EndDate:     model.Date{Month: 4, Year: 2027},
	}
	id, _ := pgStorage.CreateSubscription(spec)

	// 2.Case non-existen id
	err := pgStorage.DeleteSubscription(-532)
	assert.ErrorIs(t, err, storage.ErrSubscribtionNotFound)

	// 3.Case OK
	err = pgStorage.DeleteSubscription(id)
	assert.Nil(t, err)

	_, err = pgStorage.GetSubscription(id)
	assert.ErrorIs(t, err, storage.ErrSubscribtionNotFound)
}

func TestGetSubscriptions(t *testing.T) {
	// 1.Init
	pool := newTestDB(t)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pgStorage := newStorage(logger, pool)

	// 2.Tests
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
		_, err := pgStorage.CreateSubscription(spec)
		assert.Nil(t, err)
	}

	subs, err := pgStorage.GetSubscriptions()
	assert.Nil(t, err)

	for i := 0; i < len(subs); i++ {
		assert.Equal(t, subs[i].ID, int64(i+1))
		assert.Equal(t, subs[i].ServiceName, services[i])
		assert.Equal(t, subs[i].Price, prices[i])
	}
}
