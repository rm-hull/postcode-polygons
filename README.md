# Postcode Polygons

## Data extraction and reprocessing

First download (or otherwise generate) the **gb-postcodes-v5.tar.bz2** data file. See https://longair.net/blog/2021/08/23/open-data-gb-postcode-unit-boundaries/.

```console
$ curl https://postcodes-mapit-static.s3.eu-west-2.amazonaws.com/data/gb-postcodes-v5.tar.bz2 -O data/gb-postcodes-v5.tar.bz2
$ go run main.go
```
