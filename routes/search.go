package routes

import (
	"fmt"
	"log"
	"math"
	"net/http"
	spatialindex "postcode-polygons/spatial-index"
	"strconv"
	"strings"

	"encoding/json"
	"os"

	"github.com/dsnet/compress/bzip2"
	"github.com/gin-gonic/gin"
	"github.com/paulmach/orb/geojson"
)

type SearchResponse struct {
	Results     []spatialindex.CodePoint `json:"results"`
	Attribution []string                 `json:"attribution"`
}

const MAX_BOUNDS = 5000 // Maximum bounds in meters (5 KM)

func CodePointSearch(spatialIndex *spatialindex.SpatialIndex) func(c *gin.Context) {
	return func(c *gin.Context) {
		bbox, err := parseBBox(c.Query("bbox"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		results, err := spatialIndex.Search(bbox)
		if err != nil {
			log.Printf("error while fetching postcode data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
			return
		}

		c.JSON(http.StatusOK, SearchResponse{
			Results:     *results,
			Attribution: ATTRIBUTION,
		})
	}
}

func PolygonSearch(spatialIndex *spatialindex.SpatialIndex) func(c *gin.Context) {
	return func(c *gin.Context) {
		bbox, err := parseBBox(c.Query("bbox"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		requestedPostcodes := make(map[string]struct{})
		districts := make(map[string]struct{})

		err = spatialIndex.SearchIter(bbox, func(min, max [2]uint32, postcode string) bool {
			district := strings.Split(postcode, " ")[0] // Take the first part of the postcode
			districts[district] = struct{}{}
			requestedPostcodes[postcode] = struct{}{}
			return true
		})
		if err != nil {
			log.Printf("error while fetching postcode data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
			return
		}

		var fc geojson.FeatureCollection
		for district := range districts {
			featureCollection, err := loadFromFile(fmt.Sprintf("./data/postcodes/%s.geojson.bz2", district))
			if err != nil {
				log.Printf("error loading polygon for district %s: %v", district, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
				return
			}
			for _, feature := range featureCollection.Features {
				if feature.Properties != nil {
					if v, ok := feature.Properties["postcode"]; ok {
						if postCode, ok := v.(string); ok {
							if _, exists := requestedPostcodes[postCode]; exists {
								fc.Append(feature)
							}
						}
					}
				}
			}
		}

		c.Header("Content-Type", "application/geo+json")
		c.JSON(http.StatusOK, &fc)
	}
}

func loadFromFile(bz2filename string) (*geojson.FeatureCollection, error) {

	file, err := os.Open(bz2filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file %s: %v", bz2filename, err)
		}
	}()

	bz2Reader, err := bzip2.NewReader(file, &bzip2.ReaderConfig{})
	if err != nil {
		return nil, fmt.Errorf("error creating bzip2 reader: %w", err)
	}
	var fc geojson.FeatureCollection
	decoder := json.NewDecoder(bz2Reader)
	if err := decoder.Decode(&fc); err != nil {
		return nil, err
	}
	return &fc, nil
}

func parseBBox(bboxStr string) ([]uint32, error) {
	bboxParts := strings.Split(bboxStr, ",")
	if len(bboxParts) != 4 {
		return nil, fmt.Errorf("bbox must have 4 comma-separated values")
	}

	bbox := make([]uint32, 4)
	for i, part := range bboxParts {
		val, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid bbox value '%s': not a valid float", part)
		}
		bbox[i] = uint32(val)
	}

	if bbox[0] > bbox[2] || bbox[1] > bbox[3] {
		return nil, fmt.Errorf("invalid bbox: min values must be less than or equal to max values")
	}

	if math.Abs(float64(bbox[2]-bbox[0])) > MAX_BOUNDS || math.Abs(float64(bbox[3]-bbox[1])) > MAX_BOUNDS {
		return nil, fmt.Errorf("bbox must define a valid area (no more than %d KM in either dimension)", MAX_BOUNDS/1000)
	}

	return bbox, nil
}
