package main

import (
	"postcode-polygons/extraction"
)

func main() {
	extraction.Extract("./data/gb-postcodes-v5.tar.bz2")
}
