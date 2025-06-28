package internal

import (
	"fmt"

	"github.com/kofalt/go-memoize"
	"github.com/paulmach/orb/geojson"
)

type PolygonsRepo interface {
	RetrieveFeatureCollection(target string, district string) (*geojson.FeatureCollection, error)
}

type CachedPolygonsRepo struct {
	cache *memoize.Memoizer
}

func NewPolygonsRepo(cache *memoize.Memoizer) PolygonsRepo {
	return &CachedPolygonsRepo{cache: cache}
}

func (cp *CachedPolygonsRepo) RetrieveFeatureCollection(target string, district string) (*geojson.FeatureCollection, error) {
	filename := fmt.Sprintf("./data/postcodes/%s/%s.geojson.bz2", target, district)
	featureCollection, err, _ := memoize.Call(cp.cache, filename, func() (*geojson.FeatureCollection, error) {
		return DecompressFeatureCollection(filename)
	})
	return featureCollection, err
}
