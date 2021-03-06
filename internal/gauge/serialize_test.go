package gauge

import (
	"bytes"
	"testing"
	"time"
)

func TestEncodeDecode(t *testing.T) {
	station := Station{
		DataURL:   "http://environment.data.gov.uk/flood-monitoring/id/measures/1029TH-level-downstage-i-15_min-mASD",
		AliasURL:  "rloi://1234",
		HumanURL:  "http://environment.data.gov.uk/flood-monitoring/id/stations/1029TH",
		Name:      "Bourton Dickler",
		RiverName: "Dikler",
		Lat:       51.874767,
		Lg:        -1.740083,
		Type:      "level",
		Unit:      "metre",
	}
	timestamp, _ := time.Parse(time.RFC3339, "2016-01-01T10:30:00Z")
	var readings []Reading
	readings = append(readings, Reading{
		EventTime: timestamp.Add(time.Second),
		Value:     1.23,
	})
	readings = append(readings, Reading{
		EventTime: timestamp.Add(time.Second * 10),
		Value:     4.56,
	})

	before := Snapshot{
		Station:       station,
		Readings:      readings,
		CorrelationID: "ABCDE",
		CausationID:   "FGHIJ",
		ProcessedTime: timestamp,
	}
	var bb bytes.Buffer
	err := before.Encode(&bb)
	if err != nil {
		t.Error(err)
	}
	after := Snapshot{}
	err = after.Decode(&bb)
	if err != nil {
		t.Error(err)
	}

	// check fields individually (not using reflect.DeepEqual as
	// some custom compare needed for the dates)
	if before.Station.DataURL != after.Station.DataURL {
		t.Error("Url mis-match", after)
	}
	if before.Station.AliasURL != after.Station.AliasURL {
		t.Error("Url mis-match", after)
	}
	if before.Station.HumanURL != after.Station.HumanURL {
		t.Error("Station Url mis-match", after)
	}
	if before.Station.Name != after.Station.Name {
		t.Error("Name mis-match", after)
	}
	if before.Station.RiverName != after.Station.RiverName {
		t.Error("River name mis-match", after)
	}
	if before.Station.Lat != after.Station.Lat {
		t.Error("Url mis-match", after)
	}
	if before.Station.Lg != after.Station.Lg {
		t.Error("Lg mis-match", after)
	}
	if before.Station.Type != after.Station.Type {
		t.Error("Type mis-match", after)
	}
	if before.Station.Unit != after.Station.Unit {
		t.Error("Unit mis-match", after)
	}

	if len(after.Readings) != len(before.Readings) {
		t.Error("length mismatch", len(before.Readings), len(after.Readings))
		return
	}

	for i, b := range before.Readings {
		a := after.Readings[i]
		if b.EventTime.Unix() != a.EventTime.Unix() {
			t.Error("Timestamp mis-match", i, b.EventTime.Unix(), a.EventTime.Unix())
		}
		if b.Value != a.Value {
			t.Error("Value mis-match", i, b.Value, a.Value)
		}
	}

	if !before.ProcessedTime.Equal(after.ProcessedTime) {
		t.Error("Processing time mis-match", after)
	}
	if before.CorrelationID != after.CorrelationID {
		t.Error("CorrelationID mis-match", after)
	}
	if before.CausationID != after.CausationID {
		t.Error("CausationID mis-match", after)
	}
}
