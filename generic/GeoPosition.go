package generic

import (
	"errors"
	"log"
	"strconv"
	"strings"
)

type GeoPosition struct {
	Latitude  float32
	Longitude float32
	Altitude  float32
}

func ConvertDDMMSSSSLatitudeToFloat(lat string) (float32, error) {
	lat = strings.Trim(lat, " ")
	var positiveSign bool
	var deg int
	var min int
	var sec float32

	//Identify the nort south
	p := 0
	if strings.Contains(lat, "N") {
		positiveSign = true
		p = strings.Index(lat, "N")
	} else if strings.Contains(lat, "S") {
		positiveSign = false
		p = strings.Index(lat, "S")
	} else {
		return 0.0, errors.New(lat + " Not a valid format DDMMSS.SS{N|S}")
	}

	if (p + 1) < len(lat) {
		initlong := lat
		log.Printf("%s string has been clean to keep only %s", initlong, lat[0:p+1])
	}

	//extract the seconds
	lat = lat[0:p] //remove the last char
	if strings.Contains(lat, ".") {
		p := strings.Index(lat, ".") - 2 //get two digit before the 2
		secs := lat[p:]
		secp, err := strconv.ParseFloat(secs, 32)
		if err != nil {
			return 0.0, err
		} else {
			sec = float32(secp)
			lat = lat[0:p] //keep only the left part
		}
	} else {
		secs := lat[len(lat)-2:]
		secp, err := strconv.ParseFloat(secs, 32)
		if err != nil {
			return 0.0, err
		} else {
			sec = float32(secp)
			lat = lat[0 : len(lat)-2]
		}
	}

	//extract the minutes and DD
	mm := lat[len(lat)-2:]
	mmp, err := strconv.Atoi(mm)
	if err != nil {
		return 0.0, err
	} else {
		min = mmp
		lat = lat[0 : len(lat)-2]
	}

	dd := lat
	ddp, err := strconv.Atoi(dd)
	if err != nil {
		return 0.0, err
	} else {
		deg = ddp
	}

	results := float32(deg) + float32(min)/60 + sec/3600
	if positiveSign {
		return results, nil
	} else {
		return -results, nil
	}
}

func ConvertDDDMMSSSSLongitudeToFloat(long string) (float32, error) {
	long = strings.Trim(long, " ")
	var positiveSign bool
	var deg int
	var min int
	var sec float32
	p := 0
	if strings.Contains(long, "E") {
		positiveSign = true
		p = strings.Index(long, "E")
	} else if strings.Contains(long, "W") {
		positiveSign = false
		p = strings.Index(long, "W")
	} else {
		return 0.0, errors.New(long + " Not a valid format DDDMMSS.SS{E|W}")
	}

	if (p + 1) < len(long) {
		initlong := long
		log.Printf("%s string has been clean to keep only %s", initlong, long[0:p+1])
	}

	//extract the seconds
	long = long[0:p] //remove the last char
	if strings.Contains(long, ".") {
		p := strings.Index(long, ".") - 2 //get two digit before the 2
		secs := long[p:]
		secp, err := strconv.ParseFloat(secs, 32)
		if err != nil {
			return 0.0, err
		} else {
			sec = float32(secp)
			long = long[0:p] //keep only the left part
		}
	} else {
		secs := long[len(long)-2:]
		secp, err := strconv.ParseFloat(secs, 32)
		if err != nil {
			return 0.0, err
		} else {
			sec = float32(secp)
			long = long[0 : len(long)-2]
		}
	}

	//extract the minutes and DD
	mm := long[len(long)-2:]
	mmp, err := strconv.Atoi(mm)
	if err != nil {
		return 0.0, err
	} else {
		min = mmp
		long = long[0 : len(long)-2]
	}

	dd := long
	ddp, err := strconv.Atoi(dd)
	if err != nil {
		return 0.0, err
	} else {
		deg = ddp
	}

	results := float32(deg) + float32(min)/60 + sec/3600
	if positiveSign {
		return results, nil
	} else {
		return -results, nil
	}

}
