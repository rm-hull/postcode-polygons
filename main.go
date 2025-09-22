package main

import (
	"log"
	"postcode-polygons/cmd"

	"github.com/spf13/cobra"
)

func main() {
	var err error
	var polygonTarBz2File string
	var codePointZipFile string
	var port int
	var debug bool

	rootCmd := &cobra.Command{
		Use:  "postcode-polygons",
		Long: `HTTP server & data extraction`,
	}

	apiServerCmd := &cobra.Command{
		Use:   "api-server [--codepoint <path>] [--port <port>] [--debug]",
		Short: "Start HTTP API server",
		Run: func(_ *cobra.Command, _ []string) {
			cmd.ApiServer(codePointZipFile, port, debug)
		},
	}
	apiServerCmd.Flags().StringVar(&codePointZipFile, "codepoint",
		"https://api.os.uk/downloads/v1/products/CodePointOpen/downloads?area=GB&format=CSV&redirect",
		"Path or URL to CodePoint Open zip file")
	apiServerCmd.Flags().IntVar(&port, "port", 8080, "Port to run HTTP server on")
	apiServerCmd.Flags().BoolVar(&debug, "debug", false, "Enable debugging (pprof) - WARING: do not enable in production")

	extractDataCmd := &cobra.Command{
		Use:   "extract-data [--polygon <path>]",
		Short: "Extract NSUL polygons",
		Run: func(_ *cobra.Command, _ []string) {
			cmd.ExtractData(polygonTarBz2File)
		},
	}
	extractDataCmd.Flags().StringVar(&polygonTarBz2File, "polygon", "./data/gb-postcodes-v5.tar.bz2", "Path to NSUL polygons tar.bz2 file")

	rootCmd.AddCommand(apiServerCmd)
	rootCmd.AddCommand(extractDataCmd)

	if err = rootCmd.Execute(); err != nil {
		log.Fatalf("failed to execute root command: %v", err)
	}
}
