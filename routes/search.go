package routes

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"postcode-polygons/internal"
	spatialindex "postcode-polygons/spatial-index"
	"strconv"
	"strings"

	"os"

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

		requestedPostcodes := make(map[string]struct{}, 100)
		districts := make(map[string]struct{}, 20)

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

		isRequestedPostcode := func(feature *geojson.Feature) bool {
			postcode, ok := feature.Properties["postcode"].(string)
			if !ok {
				return false
			}
			_, exists := requestedPostcodes[postcode]
			return exists
		}

		fc := geojson.NewFeatureCollection()
		fc.Features = make([]*geojson.Feature, 0, len(requestedPostcodes))

		for district := range districts {
			filename := fmt.Sprintf("./data/postcodes/units/%s.geojson.bz2", district)
			featureCollection, err := internal.DecompressFeatureCollection(filename)
			if err != nil && os.IsNotExist(err) {
				log.Printf("polygon file for district %s does not exist, skipping", district)
				continue
			}
			if err != nil {
				log.Printf("error loading polygon for district %s: %v", district, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
				return
			}
			for _, feature := range featureCollection.Features {
				if isRequestedPostcode(feature) {
					fc.Append(feature)
				}
			}
		}

		c.Header("Content-Type", "application/geo+json")
		c.JSON(http.StatusOK, &fc)
	}
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
