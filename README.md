# Postcode Polygons

## Data extraction and reprocessing

First download (or otherwise generate) the **gb-postcodes-v5.tar.bz2** data file. See https://longair.net/blog/2021/08/23/open-data-gb-postcode-unit-boundaries/.

```console
$ curl https://postcodes-mapit-static.s3.eu-west-2.amazonaws.com/data/gb-postcodes-v5.tar.bz2 -O data/gb-postcodes-v5.tar.bz2
$ go run main.go extract-data
```

This will regenerate the (checked-in) data files under `./data/postcodes`. There is typically no need to run this unless
the NSUL datafile got updated

## Starting the API server

```console
$ go run main.go api-server
```
