package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDate(t *testing.T) {
	d, err := ParseDate("1990-05-12")
	require.NoError(t, err)
	assert.Equal(t, time.Date(1990, 5, 12, 0, 0, 0, 0, time.UTC), d.Time)
	assert.Equal(t, "1990-05-12", d.String())

	for _, bad := range []string{"", "12/05/1990", "1990-5-12", "1990-02-30", "1990-05-12T00:00:00Z"} {
		_, err := ParseDate(bad)
		assert.Error(t, err, "input %q", bad)
	}
}

func TestDate_JSON(t *testing.T) {
	d, _ := ParseDate("1990-05-12")

	out, err := json.Marshal(struct {
		DOB  *Date `json:"dob"`
		None *Date `json:"none"`
	}{DOB: &d})
	require.NoError(t, err)
	assert.JSONEq(t, `{"dob":"1990-05-12","none":null}`, string(out), "date only, no time part; nil is null")
}

func TestDate_ScanAndValue(t *testing.T) {
	var d Date
	require.NoError(t, d.Scan(time.Date(1988, 3, 14, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, "1988-03-14", d.String())

	v, err := d.Value()
	require.NoError(t, err)
	assert.Equal(t, "1988-03-14", v)

	assert.Error(t, d.Scan("1988-03-14"), "only time.Time is accepted")
}

func TestPatientFilter_HISLookupID(t *testing.T) {
	assert.Equal(t, "", PatientFilter{FirstName: "Som"}.HISLookupID())
	assert.Equal(t, "CD9876543", PatientFilter{PassportID: "CD9876543"}.HISLookupID())
	assert.Equal(t, "1100500077777", PatientFilter{NationalID: "1100500077777", PassportID: "CD9876543"}.HISLookupID(),
		"national ID wins when both are given")
}

func TestStaff_PasswordHashNeverSerialized(t *testing.T) {
	out, err := json.Marshal(Staff{ID: 1, Username: "nurse01", PasswordHash: "$2a$secret"})
	require.NoError(t, err)
	assert.NotContains(t, string(out), "secret")
}
