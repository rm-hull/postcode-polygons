package spatialindex

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type testRow struct {
	A string
	B string
}

func fromFuncSimple(data []string, headers []string) (testRow, error) {
	return testRow{A: data[0], B: data[1]}, nil
}

func fromFuncWithError(data []string, headers []string) (testRow, error) {
	return testRow{}, errors.New("parse error")
}

func TestParseCSV_WithHeader_Success(t *testing.T) {
	csvData := "a,b\nfoo,bar\nbaz,qux\n"
	reader := strings.NewReader(csvData)

	var results []Result[testRow]
	iter := parseCSV(reader, true, fromFuncSimple)
	for r := range iter {
		results = append(results, r)
	}

	require.Len(t, results, 2)
	require.Equal(t, 1, results[0].LineNum)
	require.Equal(t, "foo", results[0].Value.A)
	require.Equal(t, "bar", results[0].Value.B)
	require.NoError(t, results[0].Error)
	require.Equal(t, 2, results[1].LineNum)
	require.Equal(t, "baz", results[1].Value.A)
	require.Equal(t, "qux", results[1].Value.B)
	require.NoError(t, results[1].Error)
}

func TestParseCSV_WithoutHeader_Success(t *testing.T) {
	csvData := "foo,bar\nbaz,qux\n"
	reader := strings.NewReader(csvData)

	var results []Result[testRow]
	iter := parseCSV(reader, false, fromFuncSimple)
	for r := range iter {
		results = append(results, r)
	}

	require.Len(t, results, 2)
	require.Equal(t, 1, results[0].LineNum)
	require.Equal(t, "foo", results[0].Value.A)
	require.Equal(t, "bar", results[0].Value.B)
	require.NoError(t, results[0].Error)
	require.Equal(t, 2, results[1].LineNum)
	require.Equal(t, "baz", results[1].Value.A)
	require.Equal(t, "qux", results[1].Value.B)
	require.NoError(t, results[1].Error)
}

func TestParseCSV_ParseError(t *testing.T) {
	csvData := "a,b\nfoo,bar\nbaz,qux\n"
	reader := strings.NewReader(csvData)

	var results []Result[testRow]
	iter := parseCSV(reader, true, fromFuncWithError)
	for r := range iter {
		results = append(results, r)
	}

	require.Len(t, results, 1)
	require.Equal(t, 1, results[0].LineNum)
	require.Error(t, results[0].Error)
	require.Contains(t, results[0].Error.Error(), "failed to parse CSV line 1")
}

func TestParseCSV_MalformedLine(t *testing.T) {
	csvData := "a,b\nfoo\n"
	reader := strings.NewReader(csvData)

	var results []Result[testRow]
	iter := parseCSV(reader, true, fromFuncSimple)
	for r := range iter {
		results = append(results, r)
	}

	require.Len(t, results, 1)
	require.Equal(t, 1, results[0].LineNum)
	require.Error(t, results[0].Error)
	// Error message is from underlying csv.Reader
	require.Contains(t, results[0].Error.Error(), "wrong number of fields")
}

func TestParseCSV_HeaderReadError(t *testing.T) {
	// An empty reader will cause an io.EOF when reading headers.
	reader := strings.NewReader("")
	iter := parseCSV(reader, true, fromFuncSimple)
	var results []Result[testRow]
	for r := range iter {
		results = append(results, r)
	}
	require.Len(t, results, 1)
	require.Equal(t, 0, results[0].LineNum)
	require.Error(t, results[0].Error)
	require.Contains(t, results[0].Error.Error(), "failed to read CSV headers")
}

func TestParseCSV_YieldStopsEarly(t *testing.T) {
	csvData := "a,b\nfoo,bar\nbaz,qux\n"
	reader := strings.NewReader(csvData)

	count := 0
	iter := parseCSV(reader, true, fromFuncSimple)
	iter(func(r Result[testRow]) bool {
		count++
		return false // stop after first
	})
	require.Equal(t, 1, count)
}
