package internal

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

func applyDateTemplate(uri string) string {
	now := time.Now()
	yyyy := now.Format("2006")
	mm := now.Format("01")
	dd := now.Format("02")

	r := strings.NewReplacer(
		"{{yyyy}}", yyyy,
		"{{mm}}", mm,
		"{{dd}}", dd,
	)
	return r.Replace(uri)
}

func isValidUrl(uri string) bool {
	u, err := url.Parse(uri)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

func TransientDownload[T any](uri string, handler func(tmpfile string) (T, error)) (T, error) {
	uri = applyDateTemplate(uri)
	if !isValidUrl(uri) {
		return handler(uri)
	}

	var zero T
	log.Printf("Retrieving: %s", uri)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return zero, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("failed to fetch from %s: %w", uri, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close body: %v", err)
		}
	}()

	if resp.StatusCode > 299 {
		return zero, fmt.Errorf("error response from %s: %s", uri, resp.Status)
	}

	tmp, err := os.CreateTemp("", "download-*")
	if err != nil {
		return zero, err
	}
	tmpfile := tmp.Name()

	filesize := "unknown size"
	if resp.ContentLength >= 0 {
		filesize = humanize.Bytes(uint64(resp.ContentLength))
	}
	log.Printf("Downloading content (%s) to %s", filesize, tmpfile)

	defer func() {
		log.Printf("Removing temporary file: %s", tmpfile)
		if err := os.Remove(tmpfile); err != nil {
			log.Printf("failed to remove file %s: %v", tmpfile, err)
		}
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return zero, fmt.Errorf("failed to copy response body: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return zero, fmt.Errorf("failed to close temporary file: %w", err)
	}
	return handler(tmpfile)
}
