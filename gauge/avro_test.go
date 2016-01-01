package gauge_test

import (
	"testing"
	"time"

	"github.com/rainchasers/com.rainchasers.gauge/gauge"
)

func TestSnapshotEncodeDecode(t *testing.T) {
	timestamp, _ := time.Parse(time.RFC3339, "2016-01-01T10:30:00Z")

	before := gauge.Snapshot{
		"http://environment.data.gov.uk/flood-monitoring/id/measures/1029TH-level-downstage-i-15_min-mASD",
		"http://environment.data.gov.uk/flood-monitoring/id/stations/1029TH",
		"Bourton Dickler",
		"Dikler",
		51.874767,
		-1.740083,
		"level",
		"metre",
		timestamp,
		-0.14,
	}

	bb, err := gauge.Encode(before)
	if err != nil {
		t.Error(err)
	}

	after, err := gauge.Decode(bb)
	if err != nil {
		t.Error(err)
	}

	// check fields individually (not using reflect.DeepEqual as
	// some custom compare needed for the dates)
	if before.Url != after.Url {
		t.Error("Url mis-match", after)
	}
	if before.StationUrl != after.StationUrl {
		t.Error("Station Url mis-match", after)
	}
	if before.Name != after.Name {
		t.Error("Name mis-match", after)
	}
	if before.RiverName != after.RiverName {
		t.Error("River name mis-match", after)
	}
	if before.Lat != after.Lat {
		t.Error("Url mis-match", after)
	}
	if before.Lg != after.Lg {
		t.Error("Lg mis-match", after)
	}
	if before.DateTime.Unix() != after.DateTime.Unix() {
		t.Error("Timestamp mis-match", before.DateTime.Unix(), after.DateTime.Unix(), after)
	}
	if before.Value != after.Value {
		t.Error("Value mis-match", after)
	}
	if before.Unit != after.Unit {
		t.Error("Unit mis-match", after)
	}
}
