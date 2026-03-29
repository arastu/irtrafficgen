package assets

import _ "embed"

//go:embed geosite.dat
var GeoSiteDat []byte

//go:embed geoip.dat
var GeoIPDat []byte
