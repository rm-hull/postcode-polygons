package cmd

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"postcode-polygons/internal"
	"strings"

	"github.com/dsnet/compress/bzip2"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func ExtractData(tarBz2File string) {

	f, err := os.Open(tarBz2File)
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

	os.MkdirAll("./data/postcodes/units", os.ModePerm)
	os.MkdirAll("./data/postcodes/districts", os.ModePerm)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from tar archive: %v", err)
			}
			break
		}

		fileType, propName := extractFileType(header)
		if fileType != "" {

			outputFile := fmt.Sprintf("./data/postcodes/%ss/%s.bz2", fileType, filepath.Base(header.Name))
			if exists, err := os.Stat(outputFile); err == nil && !exists.IsDir() {
				log.Printf("Skipping file %s (already exists)", skipped(outputFile))
				continue
			}

			content := make([]byte, header.Size)
			_, err := io.ReadFull(tarReader, content)
			if err != nil {
				log.Fatalf("Error reading file %s: %v", header.Name, err)
			}

			fc, err := geojson.UnmarshalFeatureCollection(content)
			if err != nil {
				log.Fatalf("Error unmarshalling GeoJSON: %v", err)
			}

			err = reprocessFeatureCollection(fileType, propName, fc)
			if err != nil {
				log.Fatalf("Error reprocessing feature collection for file %s: %v", header.Name, err)
			}

			newSize, err := internal.CompressFeatureCollection(outputFile, fc)
			if err != nil {
				log.Fatalf("Error compressing file %s: %v", outputFile, err)
			}

			log.Printf("Processed file %s: original size %s -> %s (%0.2f%% reduction)\n",
				successful(outputFile),
				humanize.Bytes(uint64(header.Size)),
				humanize.Bytes(uint64(newSize)),
				100-float64(newSize)/float64(header.Size)*100)

		} else {
			log.Printf("Skipping: %v\n", skipped(header.Name))
		}
	}
}

func extractFileType(header *tar.Header) (string, string) {
	if header.Typeflag != tar.TypeReg {
		return "", ""
	} else if strings.HasPrefix(header.Name, "gb-postcodes-v5/units/") {
		return "unit", "postcodes"
	} else if strings.HasPrefix(header.Name, "gb-postcodes-v5/districts/") {
		return "district", "district"
	} else {
		return "", ""
	}
}

func reprocessFeatureCollection(fileType string, propName string, fc *geojson.FeatureCollection) error {

	for _, feature := range fc.Features {
		id, ok := feature.Properties[propName].(string)
		if !ok {
			return fmt.Errorf("missing or invalid '%s' property in feature", propName)
		}

		feature.ID = id
		feature.Properties["type"] = fileType
		truncateCoordinates(feature)
		delete(feature.Properties, "mapit_code")
		delete(feature.Properties, propName)
	}

	return nil
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
