package archive

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/tormoder/fit"
)

// Activity holds the fields extracted from a Strava archive.
type Activity struct {
	ID               int64
	Name             string
	StartDate        time.Time
	DistanceMeters   float64
	MovingTimeSec    int
	AverageSpeed     float64
	HasHeartrate     bool
	AverageHeartrate float64
	StartLat, StartLng float64
	MidLat, MidLng     float64
	EndLat, EndLng     float64
}

// Column indices in activities.csv. "Distance" and "Elapsed Time" appear
// twice; we want the second Distance (meters) at index 17.
const (
	colID       = 0
	colDate     = 1
	colName     = 2
	colFilename = 12
	colMovTime  = 16
	colDistM    = 17
	colAvgSpeed = 19
	colAvgHR    = 31
)

const csvDateLayout = "Jan 2, 2006, 3:04:05 PM"

// ParseZip reads a Strava export zip and returns one Activity per entry in
// activities.csv that has an associated GPS file (GPX or FIT).
//
// Strava archives contain two activity file formats: GPX (older activities
// recorded via phone) and FIT (newer activities, including Apple Watch).
// Metadata (distance, time, HR, speed) comes from activities.csv rather than
// being recomputed from track points — Strava's pre-computed values are more
// accurate and avoid re-deriving moving time, which requires pause detection.
func ParseZip(data []byte) ([]Activity, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	fileIndex := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		fileIndex[f.Name] = f
	}

	csvFile, ok := fileIndex["activities.csv"]
	if !ok {
		return nil, fmt.Errorf("activities.csv not found in archive")
	}
	rc, err := csvFile.Open()
	if err != nil {
		return nil, err
	}
	metas, err := parseCSV(rc)
	rc.Close()
	if err != nil {
		return nil, err
	}

	var activities []Activity
	for _, m := range metas {
		if m.filename == "" || m.distanceMeters == 0 {
			continue
		}
		f, ok := fileIndex[m.filename]
		if !ok {
			continue
		}
		sLat, sLng, mLat, mLng, eLat, eLng, err := extractCoords(f)
		if err != nil || (sLat == 0 && sLng == 0) {
			continue
		}
		activities = append(activities, Activity{
			ID:               m.id,
			Name:             m.name,
			StartDate:        m.startDate,
			DistanceMeters:   m.distanceMeters,
			MovingTimeSec:    m.movingTimeSec,
			AverageSpeed:     m.avgSpeed,
			HasHeartrate:     m.hasHR,
			AverageHeartrate: m.avgHR,
			StartLat:         sLat,
			StartLng:         sLng,
			MidLat:           mLat,
			MidLng:           mLng,
			EndLat:           eLat,
			EndLng:           eLng,
		})
	}
	return activities, nil
}

type csvMeta struct {
	id             int64
	name           string
	startDate      time.Time
	distanceMeters float64
	movingTimeSec  int
	avgSpeed       float64
	hasHR          bool
	avgHR          float64
	filename       string
}

func parseCSV(r io.Reader) ([]csvMeta, error) {
	rd := csv.NewReader(r)
	if _, err := rd.Read(); err != nil {
		return nil, err
	}
	var out []csvMeta
	for {
		row, err := rd.Read()
		if err != nil {
			break
		}
		if len(row) <= colAvgHR {
			continue
		}
		id, _ := strconv.ParseInt(row[colID], 10, 64)
		distM, _ := strconv.ParseFloat(row[colDistM], 64)
		movSec, _ := strconv.ParseInt(row[colMovTime], 10, 64)
		avgSpeed, _ := strconv.ParseFloat(row[colAvgSpeed], 64)
		avgHR, _ := strconv.ParseFloat(row[colAvgHR], 64)
		startDate, _ := time.Parse(csvDateLayout, row[colDate])
		out = append(out, csvMeta{
			id:             id,
			name:           row[colName],
			startDate:      startDate,
			distanceMeters: distM,
			movingTimeSec:  int(movSec),
			avgSpeed:       avgSpeed,
			hasHR:          avgHR > 0,
			avgHR:          avgHR,
			filename:       row[colFilename],
		})
	}
	return out, nil
}

func extractCoords(f *zip.File) (sLat, sLng, mLat, mLng, eLat, eLng float64, err error) {
	rc, err := f.Open()
	if err != nil {
		return
	}
	defer rc.Close()

	name := path.Base(f.Name)
	var reader io.Reader = rc

	if strings.HasSuffix(name, ".gz") {
		gr, gerr := gzip.NewReader(rc)
		if gerr != nil {
			err = gerr
			return
		}
		defer gr.Close()
		reader = gr
		name = strings.TrimSuffix(name, ".gz")
	}

	switch {
	case strings.HasSuffix(name, ".gpx"):
		sLat, sLng, mLat, mLng, eLat, eLng, err = gpxCoords(reader)
	case strings.HasSuffix(name, ".fit"):
		sLat, sLng, mLat, mLng, eLat, eLng, err = fitCoords(reader)
	}
	return
}

type gpxTrkpt struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
}

type gpxTrkseg struct {
	Trkpts []gpxTrkpt `xml:"trkpt"`
}

type gpxTrk struct {
	Trksegs []gpxTrkseg `xml:"trkseg"`
}

type gpxDoc struct {
	Trks []gpxTrk `xml:"trk"`
}

func gpxCoords(r io.Reader) (sLat, sLng, mLat, mLng, eLat, eLng float64, err error) {
	var doc gpxDoc
	if err = xml.NewDecoder(r).Decode(&doc); err != nil {
		return
	}
	var pts []gpxTrkpt
	for _, trk := range doc.Trks {
		for _, seg := range trk.Trksegs {
			pts = append(pts, seg.Trkpts...)
		}
	}
	if len(pts) < 2 {
		return
	}
	sLat, sLng = pts[0].Lat, pts[0].Lon
	mid := pts[len(pts)/2]
	mLat, mLng = mid.Lat, mid.Lon
	eLat, eLng = pts[len(pts)-1].Lat, pts[len(pts)-1].Lon
	return
}

func fitCoords(r io.Reader) (sLat, sLng, mLat, mLng, eLat, eLng float64, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return
	}
	fitFile, err := fit.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	activity, err := fitFile.Activity()
	if err != nil {
		return
	}

	// Collect all valid GPS records so we can find start, mid, and end.
	type latLng struct{ lat, lng float64 }
	var pts []latLng
	for _, rec := range activity.Records {
		if rec.PositionLat.Invalid() || rec.PositionLong.Invalid() {
			continue
		}
		pts = append(pts, latLng{rec.PositionLat.Degrees(), rec.PositionLong.Degrees()})
	}
	if len(pts) < 2 {
		return
	}
	sLat, sLng = pts[0].lat, pts[0].lng
	mid := pts[len(pts)/2]
	mLat, mLng = mid.lat, mid.lng
	eLat, eLng = pts[len(pts)-1].lat, pts[len(pts)-1].lng
	return
}
