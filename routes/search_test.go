package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	spatialindex "postcode-polygons/spatial-index"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kofalt/go-memoize"
	"github.com/stretchr/testify/require"
)

type mockSpatialIndex struct {
	SearchFunc     func(bounds []uint32) (*[]spatialindex.CodePoint, error)
	SearchIterFunc func(bounds []uint32, iter func([2]uint32, [2]uint32, string) bool) error
	LenFunc        func() int
}

func (m *mockSpatialIndex) Search(bounds []uint32) (*[]spatialindex.CodePoint, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(bounds)
	}
	return nil, nil
}
func (m *mockSpatialIndex) SearchIter(bounds []uint32, iter func([2]uint32, [2]uint32, string) bool) error {
	if m.SearchIterFunc != nil {
		return m.SearchIterFunc(bounds, iter)
	}
	return nil
}
func (m *mockSpatialIndex) Len() int {
	if m.LenFunc != nil {
		return m.LenFunc()
	}
	return 0
}

// --- CodePointSearch tests ---
func TestCodePointSearch_BadBBox(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/search?bbox=bad,bbox,values", nil)
	c.Request.URL.RawQuery = "bbox=bad,bbox,values"

	spatialIdx := &mockSpatialIndex{}
	handler := CodePointSearch(spatialIdx)
	handler(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "bbox must have 4 comma-separated values")
}

func TestCodePointSearch_TooBig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/search?bbox=0,0,10000,10000", nil)
	c.Request.URL.RawQuery = "bbox=0,0,10000,10000"

	spatialIdx := &mockSpatialIndex{}
	handler := CodePointSearch(spatialIdx)
	handler(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "bbox is too large")
}

func TestCodePointSearch_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/search?bbox=0,0,1,1", nil)
	c.Request.URL.RawQuery = "bbox=0,0,1,1"

	spatialIdx := &mockSpatialIndex{
		SearchFunc: func(bounds []uint32) (*[]spatialindex.CodePoint, error) {
			return nil, errors.New("fail")
		},
	}
	handler := CodePointSearch(spatialIdx)
	handler(c)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "An internal server error occurred")
}

func TestCodePointSearch_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/search?bbox=0,0,1,1", nil)
	c.Request.URL.RawQuery = "bbox=0,0,1,1"

	spatialIdx := &mockSpatialIndex{
		SearchFunc: func(bounds []uint32) (*[]spatialindex.CodePoint, error) {
			results := []spatialindex.CodePoint{{PostCode: "AB1 2CD", Easting: 1, Northing: 2}}
			return &results, nil
		},
	}
	handler := CodePointSearch(spatialIdx)
	handler(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "AB1 2CD")
}

// --- PolygonSearch tests ---
func TestPolygonSearch_BadBBox(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/polygon?bbox=bad,bbox,values", nil)
	c.Request.URL.RawQuery = "bbox=bad,bbox,values"

	spatialIdx := &mockSpatialIndex{}
	cache := memoize.NewMemoizer(0, 0)
	handler := PolygonSearch(spatialIdx, cache)
	handler(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "bbox must have 4 comma-separated values")
}

func TestPolygonSearch_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/polygon?bbox=0,0,1,1", nil)
	c.Request.URL.RawQuery = "bbox=0,0,1,1"

	spatialIdx := &mockSpatialIndex{
		SearchIterFunc: func(bounds []uint32, iter func([2]uint32, [2]uint32, string) bool) error {
			return errors.New("fail")
		},
	}
	cache := memoize.NewMemoizer(0, 0)
	handler := PolygonSearch(spatialIdx, cache)
	handler(c)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "An internal server error occurred")
}

// --- parseBBox and isTooBig tests ---
func TestParseBBox(t *testing.T) {
	_, err := parseBBox("1,2,3,4")
	require.NoError(t, err)

	_, err = parseBBox("1,2,3")
	require.Error(t, err)

	_, err = parseBBox("a,b,c,d")
	require.Error(t, err)

	_, err = parseBBox("4,3,2,1")
	require.Error(t, err)
}

func TestIsTooBig(t *testing.T) {
	require.False(t, isTooBig([]uint32{0, 0, 100, 100}))
	require.True(t, isTooBig([]uint32{0, 0, 6000, 100}))
	require.True(t, isTooBig([]uint32{0, 0, 100, 6000}))
}
