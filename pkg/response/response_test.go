package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTest() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	return c, w
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

func TestSuccess(t *testing.T) {
	c, w := setupTest()
	Success(c, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "ok", resp["message"])
	assert.Equal(t, "value", resp["data"].(map[string]interface{})["key"])
}

func TestSuccess_NilData(t *testing.T) {
	c, w := setupTest()
	Success(c, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, float64(0), resp["code"])
}

func TestCreated(t *testing.T) {
	c, w := setupTest()
	Created(c, map[string]string{"id": "HR001"})

	assert.Equal(t, http.StatusCreated, w.Code)

	resp := parseResponse(w)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "created", resp["message"])
}

func TestPaginated(t *testing.T) {
	t.Run("1 page with 20 items", func(t *testing.T) {
		c, w := setupTest()
		items := []map[string]interface{}{
			{"id": "1", "name": "test"},
		}
		Paginated(c, items, int64(100), 1, 20)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := parseResponse(w)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(100), data["total"])
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(20), data["page_size"])
		assert.Equal(t, float64(5), data["total_pages"])
	})

	t.Run("last page partial", func(t *testing.T) {
		c, w := setupTest()
		Paginated(c, []interface{}{}, int64(95), 5, 20)

		resp := parseResponse(w)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(5), data["total_pages"], "95 items / 20 page_size = 5 pages")
	})

	t.Run("zero page size", func(t *testing.T) {
		c, w := setupTest()
		Paginated(c, []interface{}{}, int64(0), 1, 0)

		resp := parseResponse(w)
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(0), data["total_pages"])
	})
}

func TestErrorFunctions(t *testing.T) {
	tests := []struct {
		name       string
		call       func(c *gin.Context, msg string)
		statusCode int
		code       int
		message    string
	}{
		{"BadRequest", BadRequest, 400, 40000, "bad request"},
		{"Unauthorized default", func(c *gin.Context, msg string) { Unauthorized(c, "") }, 401, 40100, "unauthorized"},
		{"Unauthorized custom", func(c *gin.Context, msg string) { Unauthorized(c, "custom msg") }, 401, 40100, "custom msg"},
		{"Forbidden default", func(c *gin.Context, msg string) { Forbidden(c, "") }, 403, 40300, "forbidden"},
		{"NotFound", NotFound, 404, 40400, "resource not found"},
		{"Conflict", Conflict, 409, 40900, "conflict"},
		{"InternalError default", func(c *gin.Context, msg string) { InternalError(c, "") }, 500, 50000, "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTest()
			tt.call(c, tt.message)

			assert.Equal(t, tt.statusCode, w.Code)

			resp := parseResponse(w)
			assert.Equal(t, float64(tt.code), resp["code"])
			assert.Equal(t, tt.message, resp["message"])
		})
	}
}
