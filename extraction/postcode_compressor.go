package extraction

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsnet/compress/bzip2"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func Extract(tarFile string) {

	f, err := os.Open(tarFile)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}()

	bz2Reader, err := bzip2.NewReader(f, &bzip2.ReaderConfig{})
	if err != nil {
		log.Fatalf("Error creating bzip2 reader: %v", err)
	}
	tarReader := tar.NewReader(bz2Reader)

	skipped := color.New(color.FgBlue).SprintFunc()
	successful := color.New(color.FgGreen).SprintFunc()

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from tar archive: %v", err)
			}
			break
		}

		if header.Typeflag == tar.TypeReg && strings.HasPrefix(header.Name, "gb-postcodes-v5/units/") {

			filename := filepath.Base(header.Name)
			if exists, err := os.Stat("./data/postcodes/" + filename + ".bz2"); err == nil && !exists.IsDir() {
				log.Printf("Skipping file %s (already exists)", skipped(filename))
				continue
			}

			content := make([]byte, header.Size)
			_, err := io.ReadFull(tarReader, content)
			if err != nil {
				log.Fatalf("Error reading file %s: %v", header.Name, err)
			}

			processed, err := reprocessFile(content)
			if err != nil {
				log.Fatalf("Error processing file %s: %v", header.Name, err)
			}

			outputFile := "./data/postcodes/" + filename + ".bz2"
			newSize, err := compressFile(outputFile, processed)
			if err != nil {
				log.Fatalf("Error compressing file %s: %v", outputFile, err)
			}

			log.Printf("Processed file %s: original size %s -> %s (%0.2f%% reduction)\n",
				successful(filename),
				humanize.Bytes(uint64(header.Size)),
				humanize.Bytes(uint64(newSize)),
				100-float64(newSize)/float64(header.Size)*100)

		} else {
			log.Printf("Skipping: %v\n", skipped(header.Name))
		}
	}
}

func compressFile(filename string, content []byte) (int, error) {
	f, err := os.Create(filename)
	if err != nil {
		return 0, fmt.Errorf("error creating output file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Error closing file %s: %v", filename, err)
		}
	}()

	w, err := bzip2.NewWriter(f, &bzip2.WriterConfig{Level: bzip2.BestCompression})
	if err != nil {
		return 0, fmt.Errorf("error creating bzip2 writer: %w", err)
	}

	_, err = w.Write(content)
	if err != nil {
		return 0, fmt.Errorf("error writing bzip2 file: %w", err)
	}

	err = w.Close() // Ensure to close the writer to flush the data, so that the output offset is correct
	if err != nil {
		return 0, fmt.Errorf("error closing bzip2 writer: %w", err)
	}

	return int(w.OutputOffset), nil
}

func reprocessFile(content []byte) ([]byte, error) {
	fc, err := geojson.UnmarshalFeatureCollection(content)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling GeoJSON: %w", err)
	}

	for _, feature := range fc.Features {
		postcode, ok := feature.Properties["postcodes"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'postcodes' property in feature %s", feature.ID)
		}

		truncateCoordinates(feature)
		delete(feature.Properties, "mapit_code")
		delete(feature.Properties, "postcodes")
		feature.Properties["postcode"] = postcode
	}

	return fc.MarshalJSON()
}

func truncateCoordinates(feature *geojson.Feature) {
	if polygon, ok := feature.Geometry.(orb.Polygon); ok {
		truncatePolygon(&polygon)
	} else if multiPolygon, ok := feature.Geometry.(orb.MultiPolygon); ok {
		for i := range multiPolygon {
			truncatePolygon(&multiPolygon[i])
		}
	} else {
		log.Fatalf("Geometry type %T not supported for truncation", feature.Geometry.GeoJSONType())
	}
}

func truncatePolygon(polygon *orb.Polygon) {
	for i := range *polygon {
		ring := (*polygon)[i]
		for j := range ring {
			point := &ring[j]
			point[0] = truncate(point.X())
			point[1] = truncate(point.Y())
		}
	}
}

func truncate(value float64) float64 {
	return float64(int(value*1e6)) / 1e6
}
