package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/require"
)

// Helper to create a compressed .bz2 file with a known FeatureCollection
func createTestBZ2File(t *testing.T, fc *geojson.FeatureCollection) string {
	tmpfile, err := os.CreateTemp("", "testfc-*.geojson.bz2")
	require.NoError(t, err)
	_ = tmpfile.Close()
	_, err = CompressFeatureCollection(tmpfile.Name(), fc)
	require.NoError(t, err)
	return tmpfile.Name()
}

func TestDecompressFeatureCollection_Success(t *testing.T) {
	fc := geojson.NewFeatureCollection()
	fc.Append(geojson.NewFeature(orb.Point{1, 2}))
	fc.Append(geojson.NewFeature(orb.Point{3, 4}))

	bz2file := createTestBZ2File(t, fc)
	defer func() { _ = os.Remove(bz2file) }()

	result, err := DecompressFeatureCollection(bz2file)
	require.NoError(t, err)
	require.Equal(t, len(fc.Features), len(result.Features))
	for i := range fc.Features {
		require.Equal(t, fc.Features[i].Geometry, result.Features[i].Geometry)
	}
}

func TestDecompressFeatureCollection_FileNotFound(t *testing.T) {
	_, err := DecompressFeatureCollection("nonexistent_file.bz2")
	require.Error(t, err)
}

func TestDecompressFeatureCollection_InvalidBZ2(t *testing.T) {
	tmpfile := filepath.Join(os.TempDir(), "invalid.bz2")
	err := os.WriteFile(tmpfile, []byte("not a bzip2 file"), 0644)
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile) }()

	_, err = DecompressFeatureCollection(tmpfile)
	require.Error(t, err)
}

func TestCompressFeatureCollection_Success(t *testing.T) {
	fc := geojson.NewFeatureCollection()
	fc.Append(geojson.NewFeature(orb.Point{10, 20}))
	fc.Append(geojson.NewFeature(orb.Point{30, 40}))

	tmpfile, err := os.CreateTemp("", "compressfc-*.geojson.bz2")
	require.NoError(t, err)
	_ = tmpfile.Close()
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	n, err := CompressFeatureCollection(tmpfile.Name(), fc)
	require.NoError(t, err)
	require.Greater(t, n, 0)

	// Check file exists and is not empty
	info, err := os.Stat(tmpfile.Name())
	require.NoError(t, err)
	require.Equal(t, info.Size(), int64(n))
}

func TestCompressFeatureCollection_ErrorOnCreate(t *testing.T) {
	// Try to write to a directory (should fail)
	_, err := CompressFeatureCollection("/this/does/not/exist/test.bz2", geojson.NewFeatureCollection())
	require.Error(t, err)
}

func TestCompressFeatureCollection_And_DecompressFeatureCollection_RoundTrip(t *testing.T) {
	// Create a FeatureCollection with properties
	fc := geojson.NewFeatureCollection()
	feat := geojson.NewFeature(orb.Point{5, 6})
	feat.Properties = map[string]any{"foo": "bar"}
	fc.Append(feat)

	tmpfile, err := os.CreateTemp("", "roundtripfc-*.geojson.bz2")
	require.NoError(t, err)
	_ = tmpfile.Close()
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = CompressFeatureCollection(tmpfile.Name(), fc)
	require.NoError(t, err)

	result, err := DecompressFeatureCollection(tmpfile.Name())
	require.NoError(t, err)
	require.Equal(t, len(fc.Features), len(result.Features))
	require.Equal(t, fc.Features[0].Geometry, result.Features[0].Geometry)
	require.Equal(t, fc.Features[0].Properties["foo"], result.Features[0].Properties["foo"])
}
