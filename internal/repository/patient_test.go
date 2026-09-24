package repository

import (
	"context"
	"testing"
	"time"

	"github.com/naive555/hms-api/internal/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPatientWhere(t *testing.T) {
	dob, _ := model.ParseDate("1990-05-12")

	tests := []struct {
		name      string
		filter    model.PatientFilter
		wantWhere string
		wantArgs  []any
	}{
		{
			name:      "no filters is still scoped to the hospital",
			filter:    model.PatientFilter{},
			wantWhere: "hospital_id = $1",
			wantArgs:  []any{int64(1)},
		},
		{
			name:      "first name matches Thai or English with one shared arg",
			filter:    model.PatientFilter{FirstName: "Som"},
			wantWhere: "hospital_id = $1 AND (lower(first_name_th) LIKE $2 OR lower(first_name_en) LIKE $2)",
			wantArgs:  []any{int64(1), "som%"},
		},
		{
			name: "every filter, numbered in order",
			filter: model.PatientFilter{
				FirstName: "Som", MiddleName: "M", LastName: "Jai",
				NationalID: "1234567890123", PassportID: "AB1234567",
				DateOfBirth: &dob, PhoneNumber: "0812345678", Email: "A@Example.com",
			},
			wantWhere: "hospital_id = $1" +
				" AND (lower(first_name_th) LIKE $2 OR lower(first_name_en) LIKE $2)" +
				" AND (lower(middle_name_th) LIKE $3 OR lower(middle_name_en) LIKE $3)" +
				" AND (lower(last_name_th) LIKE $4 OR lower(last_name_en) LIKE $4)" +
				" AND national_id = $5 AND passport_id = $6 AND date_of_birth = $7" +
				" AND phone_number = $8 AND lower(email) = lower($9)",
			wantArgs: []any{int64(1), "som%", "m%", "jai%", "1234567890123", "AB1234567", dob, "0812345678", "A@Example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			where, args := buildPatientWhere(1, tt.filter)
			assert.Equal(t, tt.wantWhere, where)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestBuildPatientWhere_InputNeverReachesSQL(t *testing.T) {
	where, args := buildPatientWhere(1, model.PatientFilter{FirstName: "x' OR 1=1 --"})

	assert.NotContains(t, where, "OR 1=1")
	assert.Equal(t, "x' or 1=1 --%", args[1])
}

func TestPrefixPattern(t *testing.T) {
	tests := map[string]string{
		"Som":       "som%",
		"สม":        "สม%",
		"100%":      `100\%%`,
		"a_b":       `a\_b%`,
		`back\sl`:   `back\\sl%`,
		"MiXeD_%\\": `mixed\_\%\\%`,
	}
	for in, want := range tests {
		assert.Equal(t, want, prefixPattern(in), "input %q", in)
	}
}

var patientRowColumns = []string{
	"id", "hospital_id", "patient_hn",
	"first_name_th", "middle_name_th", "last_name_th",
	"first_name_en", "middle_name_en", "last_name_en",
	"date_of_birth", "national_id", "passport_id", "phone_number", "email", "gender",
	"created_at", "updated_at",
}

func TestPatientRepo_Search(t *testing.T) {
	mock := newMock(t)
	now := time.Now()
	dob, _ := model.ParseDate("1990-05-12")

	mock.ExpectQuery(sql("SELECT COUNT(*) FROM patients WHERE hospital_id = $1 AND (lower(first_name_th) LIKE $2")).
		WithArgs(int64(1), "som%").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(sql("FROM patients WHERE hospital_id = $1")+".*"+sql("ORDER BY id LIMIT $3 OFFSET $4")).
		WithArgs(int64(1), "som%", 2, 1).
		WillReturnRows(pgxmock.NewRows(patientRowColumns).
			AddRow(int64(2), int64(1), "HN0002", ptr("สมหญิง"), nil, ptr("รักดี"), ptr("Somying"), nil, ptr("Rakdee"),
				&dob, ptr("1100700012345"), nil, ptr("0898765432"), ptr("somying@example.com"), ptr("F"), now, now).
			AddRow(int64(4), int64(1), "HN0004", ptr("สมศักดิ์"), nil, nil, ptr("Somsak"), nil, nil,
				nil, ptr("1509900054321"), nil, nil, nil, ptr("M"), now, now))

	patients, total, err := NewPatientRepo(mock).Search(context.Background(), 1, model.PatientFilter{FirstName: "Som", Limit: 2, Offset: 1})
	require.NoError(t, err)

	assert.Equal(t, 3, total, "total counts all pages")
	require.Len(t, patients, 2)
	assert.Equal(t, "HN0002", patients[0].PatientHN)
	assert.Equal(t, "Somying", *patients[0].FirstNameEN)
	assert.Nil(t, patients[0].MiddleNameEN)
	assert.Equal(t, "1990-05-12", patients[0].DateOfBirth.String())
	assert.Equal(t, "HN0004", patients[1].PatientHN)
	assert.Nil(t, patients[1].DateOfBirth)
	assert.Nil(t, patients[1].Email)
}

func TestPatientRepo_Search_NoMatchesSkipsSelect(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(sql("SELECT COUNT(*) FROM patients WHERE hospital_id = $1")).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	patients, total, err := NewPatientRepo(mock).Search(context.Background(), 1, model.PatientFilter{Limit: 20})
	require.NoError(t, err)

	assert.Equal(t, 0, total)
	assert.NotNil(t, patients, "empty slice, not nil, so JSON renders []")
	assert.Empty(t, patients)
}

func TestPatientRepo_Search_Errors(t *testing.T) {
	t.Run("count fails", func(t *testing.T) {
		mock := newMock(t)
		mock.ExpectQuery(sql("SELECT COUNT(*)")).WithArgs(int64(1)).WillReturnError(errBoom)

		_, _, err := NewPatientRepo(mock).Search(context.Background(), 1, model.PatientFilter{Limit: 20})
		assert.ErrorIs(t, err, errBoom)
	})

	t.Run("select fails", func(t *testing.T) {
		mock := newMock(t)
		mock.ExpectQuery(sql("SELECT COUNT(*)")).WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(sql("ORDER BY id")).WithArgs(int64(1), 20, 0).WillReturnError(errBoom)

		_, _, err := NewPatientRepo(mock).Search(context.Background(), 1, model.PatientFilter{Limit: 20})
		assert.ErrorIs(t, err, errBoom)
	})

	t.Run("row iteration fails", func(t *testing.T) {
		mock := newMock(t)
		mock.ExpectQuery(sql("SELECT COUNT(*)")).WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(sql("ORDER BY id")).WithArgs(int64(1), 20, 0).
			WillReturnRows(pgxmock.NewRows(patientRowColumns).RowError(0, errBoom).
				AddRow(int64(1), int64(1), "HN0001", nil, nil, nil, nil, nil, nil, nil, ptr("1"), nil, nil, nil, nil, time.Now(), time.Now()))

		_, _, err := NewPatientRepo(mock).Search(context.Background(), 1, model.PatientFilter{Limit: 20})
		assert.ErrorIs(t, err, errBoom)
	})
}

func TestPatientRepo_Upsert(t *testing.T) {
	dob, _ := model.ParseDate("1988-03-14")
	now := time.Now()

	tests := []struct {
		name         string
		patient      model.Patient
		wantConflict string
	}{
		{
			name:         "national ID conflicts on the national ID partial index",
			patient:      model.Patient{HospitalID: 1, PatientHN: "HN9001", NationalID: ptr("1100500077777"), DateOfBirth: &dob},
			wantConflict: "ON CONFLICT (hospital_id, national_id) WHERE national_id IS NOT NULL DO UPDATE",
		},
		{
			name:         "passport-only conflicts on the passport partial index",
			patient:      model.Patient{HospitalID: 1, PatientHN: "HN9002", PassportID: ptr("CD9876543")},
			wantConflict: "ON CONFLICT (hospital_id, passport_id) WHERE passport_id IS NOT NULL DO UPDATE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMock(t)
			p := tt.patient
			mock.ExpectQuery(sql("INSERT INTO patients")+".*"+sql(tt.wantConflict)).
				WithArgs(p.HospitalID, p.PatientHN,
					p.FirstNameTH, p.MiddleNameTH, p.LastNameTH,
					p.FirstNameEN, p.MiddleNameEN, p.LastNameEN,
					p.DateOfBirth, p.NationalID, p.PassportID, p.PhoneNumber, p.Email, p.Gender).
				WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(9), now, now))

			require.NoError(t, NewPatientRepo(mock).Upsert(context.Background(), &p))
			assert.Equal(t, int64(9), p.ID)
		})
	}
}

func TestPatientRepo_Upsert_Errors(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{"patient_hn clash maps to ErrDuplicate", uniqueViolation("uq_patients_hospital_hn"), ErrDuplicate},
		{"other errors are wrapped", errBoom, errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMock(t)
			mock.ExpectQuery(sql("INSERT INTO patients")).
				WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
					pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
					pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnError(tt.dbErr)

			err := NewPatientRepo(mock).Upsert(context.Background(), &model.Patient{HospitalID: 1, PatientHN: "HN9001", NationalID: ptr("1100500077777")})
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
