package cmd

import (
	"fmt"
	"log"
	"os"
	"postcode-polygons/routes"
	spatialindex "postcode-polygons/spatial-index"
	"time"

	"github.com/Depado/ginprom"
	"github.com/aurowora/compress"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/kofalt/go-memoize"
	"github.com/tavsec/gin-healthcheck/checks"
	cachecontrol "go.eigsys.de/gin-cachecontrol/v2"

	healthcheck "github.com/tavsec/gin-healthcheck"
	hc_config "github.com/tavsec/gin-healthcheck/config"
)

func ApiServer(zipFile string, port int, debug bool) {
	if _, err := os.Stat(zipFile); os.IsNotExist(err) {
		log.Fatalf("CodePoint zip file does not exist: %s", zipFile)
	}

	log.Printf("Loading CodePoint data from: %s", zipFile)
	spatialIndex, err := spatialindex.NewCodePointSpatialIndex(zipFile)
	if err != nil {
		log.Fatalf("failed to create spatial index: %v", err)
	}
	log.Printf("CodePoint spatial index created with %d entries", spatialIndex.Len())

	r := gin.New()

	prometheus := ginprom.New(
		ginprom.Engine(r),
		ginprom.Path("/metrics"),
		ginprom.Ignore("/healthz"),
	)

	r.Use(
		gin.Recovery(),
		gin.LoggerWithWriter(gin.DefaultWriter, "/healthz", "/metrics"),
		prometheus.Instrument(),
		compress.Compress(),
		cachecontrol.New(cachecontrol.CacheAssetsForeverPreset),
		cors.Default(),
	)

	if debug {
		log.Println("WARNING: pprof endpoints are enabled and exposed. Do not run with this flag in production.")
		pprof.Register(r)
	}

	err = healthcheck.New(r, hc_config.DefaultConfig(), []checks.Check{})
	if err != nil {
		log.Fatalf("failed to initialize healthcheck: %v", err)
	}

	cache := memoize.NewMemoizer(5*time.Minute, 10*time.Minute)

	r.GET("/v1/postcode/codepoints", routes.CodePointSearch(spatialIndex))
	r.GET("/v1/postcode/polygons", routes.PolygonSearch(spatialIndex, cache))

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP API Server on port %d...", port)
	err = r.Run(addr)
	log.Fatalf("HTTP API Server failed to start on port %d: %v", port, err)
}
