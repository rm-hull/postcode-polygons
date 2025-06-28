package spatialindex

import (
	"archive/zip"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/tidwall/rtree"
)

type CodePoint struct {
	PostCode string `json:"post_code"`
	Easting  uint32 `json:"easting"`
	Northing uint32 `json:"northing"`
}

type SpatialIndex interface {
	Search(bounds []uint32) (*[]CodePoint, error)
	SearchIter(bounds []uint32, iter func(min, max [2]uint32, data string) bool) error
	Len() int
}

type RtreeSpatialIndex struct {
	tree *rtree.RTreeGN[uint32, string]
}

func NewCodePointSpatialIndex(zipFile string) (SpatialIndex, error) {
	idx := RtreeSpatialIndex{
		tree: &rtree.RTreeGN[uint32, string]{},
	}

	err := idx.importCodePoint(zipFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load codepoint index from zip file: %w", err)
	}

	return &idx, nil
}

func (idx *RtreeSpatialIndex) Search(bounds []uint32) (*[]CodePoint, error) {

	results := make([]CodePoint, 0, 100)
	err := idx.SearchIter(bounds, func(min, max [2]uint32, data string) bool {
		results = append(results, CodePoint{
			PostCode: data,
			Easting:  min[0],
			Northing: min[1],
		})
		return true
	})

	if err != nil {
		return nil, fmt.Errorf("error during spatial search: %w", err)
	}

	return &results, nil
}

func (idx *RtreeSpatialIndex) SearchIter(bounds []uint32, iter func(min, max [2]uint32, data string) bool) error {
	if len(bounds) != 4 {
		return fmt.Errorf("search bounds must contain exactly 4 values: min_easting, min_northing, max_easting, max_northing")
	}

	sw := [2]uint32{bounds[0], bounds[1]} // South-West corner
	ne := [2]uint32{bounds[2], bounds[3]} // North-East corner

	idx.tree.Search(sw, ne, iter)
	return nil
}

func (idx *RtreeSpatialIndex) Len() int {
	return idx.tree.Len()
}

func (idx *RtreeSpatialIndex) importCodePoint(zipPath string) error {

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("error closing zip file: %v", err)
		}
	}()

	for _, f := range r.File {
		if f.FileInfo().IsDir() || !strings.HasPrefix(f.Name, "Data/CSV/") {
			continue
		}

		if err := idx.processCSV(f); err != nil {
			return fmt.Errorf("failed to process CSV data: %w", err)
		}
	}
	return nil
}

func (idx *RtreeSpatialIndex) processCSV(f *zip.File) error {
	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open embedded file %s in zip: %w", f.Name, err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("error closing embedded zip file: %v", err)
		}
	}()

	for result := range parseCSV(r, false, fromCodePointCSV) {

		if result.Error != nil {
			return fmt.Errorf("error parsing line %d: %w", result.LineNum, result.Error)
		}

		point := [2]uint32{uint32(result.Value.Easting), uint32(result.Value.Northing)}
		idx.tree.Insert(point, point, result.Value.PostCode)
	}

	return nil
}

func fromCodePointCSV(record []string, headers []string) (*CodePoint, error) {

	easting, err := strconv.ParseUint(record[2], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid easting value: %w", err)
	}
	northing, err := strconv.ParseUint(record[3], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid northing value: %w", err)
	}

	return &CodePoint{
		PostCode: record[0],
		Easting:  uint32(easting),
		Northing: uint32(northing),
	}, nil
}
