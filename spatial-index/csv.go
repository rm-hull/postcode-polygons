package spatialindex

import (
	"encoding/csv"
	"fmt"
	"io"
	"iter"
)

type Result[T any] struct {
	Value   T
	LineNum int
	Error   error
}

func parseCSV[T any](reader io.Reader, includesHeader bool, fromFunc func(data []string, headers []string) (T, error)) iter.Seq[Result[T]] {

	return func(yield func(Result[T]) bool) {
		lineNum := 0
		csvReader := csv.NewReader(reader)
		var headers []string
		var err error

		if includesHeader {
			headers, err = csvReader.Read()
			if err != nil {
				yield(Result[T]{
					LineNum: lineNum,
					Error:   fmt.Errorf("failed to read CSV headers: %w", err),
				})
				return
			}
		}

		for {
			lineNum++
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				yield(Result[T]{
					LineNum: lineNum,
					Error:   fmt.Errorf("failed to read CSV line %d: %w", lineNum, err),
				})
				return
			}

			data, err := fromFunc(record, headers)
			if err != nil {
				yield(Result[T]{
					LineNum: lineNum,
					Error:   fmt.Errorf("failed to parse CSV line %d: %w", lineNum, err),
				})
				return
			}

			if !yield(Result[T]{
				Value:   data,
				LineNum: lineNum,
				Error:   nil,
			}) {
				return
			}
		}
	}
}
