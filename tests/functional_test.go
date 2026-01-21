package tests

import (
	"em_golang_rest_service_example/internal/http-server/handlers"
	"strconv"

	"net/http"
	"net/url"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	host = "0.0.0.0:8082"
)

var (
	u = url.URL{
		Scheme: "http",
		Host:   host,
	}
)

func TestCreate(t *testing.T) {
	e := httpexpect.Default(t, u.String())

	// 1.Send request on create
	req := handlers.CreateRequest{
		ServiceName: "Yandex",
		Price:       400,
		UserID:      uuid.NewString(),
		StartDate:   "07-2025",
	}

	e.POST("/subscription").
		WithJSON(req).
		Expect().
		Status(http.StatusCreated).
		JSON().Object().Value("id").IsNumber()

	// 2.Try to create it once more time
	e.POST("/subscription").
		WithJSON(req).
		Expect().
		Status(http.StatusConflict).
		JSON().Object().Value("error").IsEqual("subscription already exists")
}

func TestRead(t *testing.T) {
	e := httpexpect.Default(t, u.String())

	// 1.Send request on create
	req := handlers.CreateRequest{
		ServiceName: "Google",
		Price:       800,
		UserID:      uuid.NewString(),
		StartDate:   "07-2025",
		EndDate:     "09-2025",
	}

	id := e.POST("/subscription").
		WithJSON(req).
		Expect().
		Status(http.StatusCreated).
		JSON().Object().Value("id").Number().Raw()

	// 2.Try to get it
	expectedResp := handlers.ReadResponse{
		Id:          int64(id),
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Response:    handlers.RespOK(),
	}

	e.GET("/subscription/" + strconv.FormatInt(int64(id), 10)).
		Expect().
		Status(http.StatusOK).
		JSON().Object().IsEqual(expectedResp)

	// 3.Try to get non-existen data
	expectedResp = handlers.ReadResponse{
		Response: handlers.RespError("subscription not found"),
	}

	e.GET("/subscription/-532").
		Expect().
		Status(http.StatusNotFound).
		JSON().Object().IsEqual(expectedResp)
}

func TestUpdate(t *testing.T) {
	e := httpexpect.Default(t, u.String())

	// 1.Send request on create
	req := handlers.CreateRequest{
		ServiceName: "Netflix",
		Price:       900,
		UserID:      uuid.NewString(),
		StartDate:   "01-2026",
		EndDate:     "02-2026",
	}

	id := e.POST("/subscription").
		WithJSON(req).
		Expect().
		Status(http.StatusCreated).
		JSON().Object().Value("id").Number().Raw()

	// 2.Update it
	updateReq := handlers.UpdateRequest{
		ServiceName: "Нетфликс",
		Price:       750,
		StartDate:   "02-2026",
		EndDate:     "03-2026",
	}

	e.PATCH("/subscription/" + strconv.FormatInt(int64(id), 10)).
		WithJSON(updateReq).
		Expect().
		Status(http.StatusOK).
		JSON().Object().IsEqual(handlers.RespOK())

	// 3.Get it updated
	expectedResp := handlers.ReadResponse{
		Id:          int64(id),
		ServiceName: updateReq.ServiceName,
		Price:       updateReq.Price,
		UserID:      req.UserID,
		StartDate:   updateReq.StartDate,
		EndDate:     updateReq.EndDate,
		Response:    handlers.RespOK(),
	}

	e.GET("/subscription/" + strconv.FormatInt(int64(id), 10)).
		Expect().
		Status(http.StatusOK).
		JSON().Object().IsEqual(expectedResp)

	// 4.Try to update non-existen data
	e.PATCH("/subscription/-532").
		WithJSON(updateReq).
		Expect().
		Status(http.StatusNotFound).
		JSON().Object().IsEqual(handlers.RespError("subscription not found"))
}

func TestDelete(t *testing.T) {
	e := httpexpect.Default(t, u.String())

	// 1.Send request on create
	req := handlers.CreateRequest{
		ServiceName: "Wink",
		Price:       200,
		UserID:      uuid.NewString(),
		StartDate:   "01-2027",
	}

	id := e.POST("/subscription").
		WithJSON(req).
		Expect().
		Status(http.StatusCreated).
		JSON().Object().Value("id").Number().Raw()

	// 2.Try to delete it once more time
	e.DELETE("/subscription/" + strconv.FormatInt(int64(id), 10)).
		Expect().
		Status(http.StatusOK).
		JSON().Object().IsEqual(handlers.RespOK())

	// 3.Try to delete non-existen data
	e.DELETE("/subscription/-532").
		Expect().
		Status(http.StatusNotFound).
		JSON().Object().IsEqual(handlers.RespError("subscription not found"))
}

func TestList(t *testing.T) {
	e := httpexpect.Default(t, u.String())

	// 1.Create some data
	services := []string{"Yandex", "VKMusic", "Google", "Netflix", "Wink"}
	prices := []int{400, 75, 800, 900, 200}

	createdIDs := make([]int64, 0, len(prices))

	for i := 0; i < len(prices); i++ {
		req := handlers.CreateRequest{
			ServiceName: services[i],
			Price:       prices[i],
			UserID:      uuid.NewString(),
			StartDate:   "01-2027",
		}

		id := e.POST("/subscription").
			WithJSON(req).
			Expect().
			Status(http.StatusCreated).
			JSON().Object().Value("id").Number().Raw()

		createdIDs = append(createdIDs, int64(id))
	}

	// 2.Get all subscriptions
	var resp handlers.ListResponse
	e.GET("/subscriptions").
		Expect().
		Status(http.StatusOK).
		JSON().
		Decode(&resp)

	assert.True(t, len(resp.Items) > 0)
}

func TestTotalCost(t *testing.T) {
	e := httpexpect.Default(t, u.String())

	userId := uuid.NewString()

	// 1.Create some data for one user
	services := []string{"Yandex", "VKMusic", "Google", "Netflix", "Wink"}
	prices := []int{400, 75, 800, 900, 200}

	createdIDs := make([]int64, 0, len(prices))

	for i := 0; i < len(prices); i++ {
		req := handlers.CreateRequest{
			ServiceName: services[i],
			Price:       prices[i],
			UserID:      userId,
			StartDate:   "05-2027",
			EndDate:     "06-2027",
		}

		id := e.POST("/subscription").
			WithJSON(req).
			Expect().
			Status(http.StatusCreated).
			JSON().Object().Value("id").Number().Raw()

		createdIDs = append(createdIDs, int64(id))
	}

	// 2.Get total cost
	var resp handlers.TotalCostResponse

	e.GET("/subscriptions/total-cost").
		WithQuery("start_date", "01-2025").
		WithQuery("end_date", "01-2028").
		WithQuery("user_id", userId).
		Expect().
		Status(http.StatusOK).
		JSON().
		Decode(&resp)

	// 3.Check it
	assert.Equal(t, handlers.RespOK(), resp.Response)
	assert.Equal(t, 2375, resp.TotalCost)
}
