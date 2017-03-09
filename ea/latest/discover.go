package main

import (
	"encoding/json"
	"errors"
	"github.com/rainchasers/com.rainchasers.gauge/gauge"
	"io/ioutil"
	"net/http"
	"strconv"
)

type stationListJson struct {
	Stations []stationJson `json:"items"`
}

type stationJson struct {
	Url              string `json:"@id"`
	Name             string
	NameRawJson      json.RawMessage `json:"label"`
	RiverName        string
	RiverNameRawJson json.RawMessage `json:"riverName"`
	Lat              float32
	Lg               float32
	LatRawJson       json.RawMessage `json:"lat"`
	LgRawJson        json.RawMessage `json:"long"`
	Measures         []measureJson   `json:"measures"`
}

type measureJson struct {
	Url  string `json:"@id"`
	Name string `json:"qualifier"`
	Type string `json:"parameter"`
	Unit string `json:"unitName"`
}

func discoverStations() ([]gauge.Snapshot, error) {
	url := "http://environment.data.gov.uk/flood-monitoring/id/stations"
	var snapshots []gauge.Snapshot

	resp, err := http.Get(url)
	if err != nil {
		return snapshots, err
	}
	if resp.StatusCode != http.StatusOK {
		return snapshots, errors.New("Status code " + strconv.Itoa(resp.StatusCode))
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []gauge.Snapshot{}, err
	}

	list := stationListJson{}
	err = json.Unmarshal(bodyBytes, &list)
	if err != nil {
		return []gauge.Snapshot{}, err
	}

	for _, s := range list.Stations {
		// a known inconsistency is that the API can provide Lat, Lg or label as an array
		// so we use a defensive mechanism to parse these fields and let them be missing completely
		s.Lat, _ = parseFloatFromScalarOrArray(s.LatRawJson)
		s.Lg, _ = parseFloatFromScalarOrArray(s.LgRawJson)
		s.Name, _ = parseStringFromScalarOrArray(s.NameRawJson)
		s.RiverName, _ = parseStringFromScalarOrArray(s.RiverNameRawJson)

		for _, m := range s.Measures {

			switch m.Type {
			case "flow", "level", "temperature", "rainfall":
			default:
				continue
			}

			// no snapshot readings are available
			// s.DateTime and s.Value left as defaults

			snapshots = append(snapshots, gauge.Snapshot{
				DataURL:   m.Url,
				HumanURL:  s.Url,
				Name:      s.Name,
				RiverName: s.RiverName,
				Lat:       s.Lat,
				Lg:        s.Lg,
				Type:      m.Type,
				Unit:      m.Unit,
			})
		}
	}

	return snapshots, nil
}
