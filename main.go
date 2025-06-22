package main

import (
	"fmt"
	"log"
	"os"
	"postcode-polygons/routes"
	spatialindex "postcode-polygons/spatial-index"

	"github.com/Depado/ginprom"
	"github.com/aurowora/compress"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	hc_config "github.com/tavsec/gin-healthcheck/config"
	cachecontrol "go.eigsys.de/gin-cachecontrol/v2"
)

func main() {
	var err error
	var zipFile string
	var port int

	rootCmd := &cobra.Command{
		Use:   "http",
		Short: "Postcode Polygons API server",
		Run: func(cmd *cobra.Command, args []string) {
			server(zipFile, port)
		},
	}

	rootCmd.Flags().StringVar(&zipFile, "codepoint", "./data/codepo_gb.zip", "Path to CodePoint Open zip file")
	rootCmd.Flags().IntVar(&port, "port", 8080, "Port to run HTTP server on")

	if err = rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func server(zipFile string, port int) {
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
		ginprom.Namespace("postcode_polygons"),
		ginprom.Subsystem("api"),
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

	err = healthcheck.New(r, hc_config.DefaultConfig(), []checks.Check{})
	if err != nil {
		log.Fatalf("failed to initialize healthcheck: %v", err)
	}

	r.GET("/v1/postcode/search", routes.Search(spatialIndex))

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP API Server on port %d...", port)
	err = r.Run(addr)
	log.Fatalf("HTTP API Server failed to start on port %d: %v", port, err)
}
