package internal

import (
	"fmt"
	"log"
	"os"

	"github.com/dsnet/compress/bzip2"
	jsoniter "github.com/json-iterator/go"
	"github.com/paulmach/orb/geojson"
)

var c = jsoniter.Config{EscapeHTML: true, SortMapKeys: true, MarshalFloatWith6Digits: true}.Froze()

func CompressFeatureCollection(bz2filename string, fc *geojson.FeatureCollection) (int, error) {
	f, err := os.Create(bz2filename)
	if err != nil {
		return 0, fmt.Errorf("error creating output file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Error closing file %s: %v", bz2filename, err)
		}
	}()

	w, err := bzip2.NewWriter(f, &bzip2.WriterConfig{Level: bzip2.BestCompression})
	if err != nil {
		return 0, fmt.Errorf("error creating bzip2 writer: %w", err)
	}

	geojson.CustomJSONMarshaler = c
	geojson.CustomJSONUnmarshaler = c
	if err := c.NewEncoder(w).Encode(fc); err != nil {
		return 0, fmt.Errorf("error writing bzip2 file: %w", err)
	}

	err = w.Close() // Ensure to close the writer to flush the data, so that the output offset is correct
	if err != nil {
		return 0, fmt.Errorf("error closing bzip2 writer: %w", err)
	}

	return int(w.OutputOffset), nil
}

func DecompressFeatureCollection(bz2filename string) (*geojson.FeatureCollection, error) {

	file, err := os.Open(bz2filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file %s: %v", bz2filename, err)
		}
	}()

	r, err := bzip2.NewReader(file, &bzip2.ReaderConfig{})
	if err != nil {
		return nil, fmt.Errorf("error creating bzip2 reader: %w", err)
	}
	geojson.CustomJSONMarshaler = c
	geojson.CustomJSONUnmarshaler = c
	fc := geojson.NewFeatureCollection()
	if err := c.NewDecoder(r).Decode(fc); err != nil {
		return nil, err
	}
	return fc, nil
}
