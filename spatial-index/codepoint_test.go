package spatialindex

import (
	"archive/zip"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/rtree"
)

func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "test-*.zip")
	require.NoError(t, err, "failed to create temp file")
	defer func() { _ = tmpfile.Close() }()

	w := zip.NewWriter(tmpfile)
	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err, "failed to create file in zip")
		_, err = f.Write([]byte(content))
		require.NoError(t, err, "failed to write file in zip")
	}
	_ = w.Close()
	return tmpfile.Name()
}

func TestNewCodePointSpatialIndex_Success(t *testing.T) {
	csv := "PC1,PC2,123,456\nPC3,PC4,789,1011\n"
	zipPath := createTestZip(t, map[string]string{"Data/CSV/test.csv": csv})
	defer func() { _ = os.Remove(zipPath) }()

	idx, err := NewCodePointSpatialIndex(zipPath)
	require.NoError(t, err)
	require.Equal(t, 2, idx.Len())
}

func TestNewCodePointSpatialIndex_BadZip(t *testing.T) {
	_, err := NewCodePointSpatialIndex("/no/such/file.zip")
	require.Error(t, err, "failed to open zip file: file does not exist")
}

func TestSearch(t *testing.T) {
	csv := "PC1,PC2,100,200\nPC2,PC3,300,400\n"
	zipPath := createTestZip(t, map[string]string{"Data/CSV/test.csv": csv})
	defer func() { _ = os.Remove(zipPath) }()
	idx, err := NewCodePointSpatialIndex(zipPath)
	require.NoError(t, err)

	// Search for both
	res, err := idx.Search([]uint32{0, 0, 500, 500})
	require.NoError(t, err)
	require.Equal(t, 2, len(*res))

	// Search for one
	res, err = idx.Search([]uint32{90, 190, 110, 210})
	require.NoError(t, err)
	require.Equal(t, 1, len(*res))
	require.Equal(t, "PC1", (*res)[0].PostCode)

	// Search for none
	res, err = idx.Search([]uint32{1000, 1000, 2000, 2000})
	require.NoError(t, err)
	require.Equal(t, 0, len(*res))
}

func TestSearchIter_InvalidBounds(t *testing.T) {
	idx := &RtreeSpatialIndex{tree: &rtree.RTreeGN[uint32, string]{}}
	err := idx.SearchIter([]uint32{1, 2, 3}, func([2]uint32, [2]uint32, string) bool { return true })
	require.Error(t, err)
	require.Contains(t, err.Error(), "bounds must contain exactly 4 values")
}

func TestLen(t *testing.T) {
	csv := "PC1,PC2,1,2\nPC2,PC3,3,4\nPC3,PC4,5,6\n"
	zipPath := createTestZip(t, map[string]string{"Data/CSV/test.csv": csv})
	defer func() { _ = os.Remove(zipPath) }()
	idx, err := NewCodePointSpatialIndex(zipPath)
	require.NoError(t, err)
	require.Equal(t, 3, idx.Len())
}

func Test_fromCodePointCSV(t *testing.T) {
	rec := []string{"PC1", "PC2", "123", "456"}
	cp, err := fromCodePointCSV(rec, nil)
	require.NoError(t, err)
	require.Equal(t, "PC1", cp.PostCode)
	require.Equal(t, uint32(123), cp.Easting)
	require.Equal(t, uint32(456), cp.Northing)

	// Bad easting
	rec = []string{"PC1", "PC2", "bad", "456"}
	_, err = fromCodePointCSV(rec, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "easting")

	// Bad northing
	rec = []string{"PC1", "PC2", "123", "bad"}
	_, err = fromCodePointCSV(rec, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "northing")
}
