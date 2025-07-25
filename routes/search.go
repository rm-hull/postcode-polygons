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

const UNITS_BOUNDS_EXPANSION = 100

type SearchResponse struct {
	Results     []spatialindex.CodePoint `json:"results"`
	Attribution []string                 `json:"attribution"`
}

const MAX_BOUNDS = 5000 // Maximum bounds in meters (5 KM)

func CodePointSearch(idx spatialindex.SpatialIndex) func(c *gin.Context) {
	return func(c *gin.Context) {
		bbox, err := parseBBox(c.Query("bbox"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if isTooBig(bbox) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bbox is too large, must be less than 5km in width and height"})
			return
		}

		results, err := idx.Search(bbox)
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

func PolygonSearch(idx spatialindex.SpatialIndex, repo internal.PolygonsRepo) func(c *gin.Context) {
	return func(c *gin.Context) {
		bbox, err := parseBBox(c.Query("bbox"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tooBig := isTooBig(bbox)
		target := map[bool]string{true: "districts", false: "units"}[tooBig]

		if target == "units" {
			expandBounds(&bbox, UNITS_BOUNDS_EXPANSION)
		}

		requested := make(map[string]struct{}, 100)
		districts := make(map[string]struct{}, 20)

		err = idx.SearchIter(bbox, func(min, max [2]uint32, postcode string) bool {
			district := strings.Split(postcode, " ")[0] // Take the first part of the postcode
			districts[district] = struct{}{}
			if tooBig {
				requested[district] = struct{}{}
			} else {
				requested[postcode] = struct{}{}
			}
			return true
		})
		if err != nil {
			log.Printf("error while fetching postcode data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
			return
		}

		fc := geojson.NewFeatureCollection()
		fc.Features = make([]*geojson.Feature, 0, len(requested))

		for district := range districts {

			featureCollection, err := repo.RetrieveFeatureCollection(target, district)
			if err != nil && os.IsNotExist(err) {
				log.Printf("polygon file for district %s does not exist, skipping", district)
				continue
			}
			if err != nil {
				log.Printf("error loading feature collection for district %s: %v", district, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
				return
			}
			for _, feature := range featureCollection.Features {
				if _, exists := requested[feature.ID.(string)]; exists {
					fc.Append(feature)
				}
			}
		}

		c.Header("Content-Type", "application/geo+json")
		c.JSON(http.StatusOK, &fc)
	}
}

func expandBounds(bbox *[]uint32, extendBy uint32) {
	b := *bbox
	b[0] -= extendBy // min_easting
	b[1] -= extendBy // min_northing
	b[2] += extendBy // max_easting
	b[3] += extendBy // max_northing
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

	return bbox, nil
}

func isTooBig(bbox []uint32) bool {
	return math.Abs(float64(bbox[2]-bbox[0])) > MAX_BOUNDS || math.Abs(float64(bbox[3]-bbox[1])) > MAX_BOUNDS
}
