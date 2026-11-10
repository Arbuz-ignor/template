package trip_test

import (
	"errors"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/google/uuid"
	"math"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	valid := trip.Trip{UserID: uuid.New(), DriverID: uuid.New(), Price: 0}
	cases := []struct {
		name   string
		change func(*trip.Trip)
		field  string
	}{
		{"valid", func(*trip.Trip) {}, ""},
		{"user", func(v *trip.Trip) { v.UserID = uuid.Nil }, "user_id"},
		{"driver", func(v *trip.Trip) { v.DriverID = uuid.Nil }, "driver_id"},
		{"price", func(v *trip.Trip) { v.Price = -1 }, "price"},
		{"start latitude", func(v *trip.Trip) { v.StartPoint.Latitude = 91 }, "start_point.latitude"},
		{"end latitude", func(v *trip.Trip) { v.EndPoint.Latitude = -91 }, "end_point.latitude"},
		{"start longitude", func(v *trip.Trip) { v.StartPoint.Longitude = 181 }, "start_point.longitude"},
		{"end longitude", func(v *trip.Trip) { v.EndPoint.Longitude = -181 }, "end_point.longitude"},
		{"nan", func(v *trip.Trip) { v.StartPoint.Latitude = math.NaN() }, "start_point.latitude"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := valid
			tc.change(&v)
			err := v.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var e *trip.ValidationError
			if !errors.As(err, &e) || e.Field != tc.field || e.Error() == "" {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
func TestFinish(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	v := trip.Trip{Status: trip.Active}
	if err := v.Finish(at); err != nil || v.Status != trip.Completed || v.FinishedAt == nil || !v.FinishedAt.Equal(at) {
		t.Fatalf("finish: %+v %v", v, err)
	}
	if err := v.Finish(at.Add(time.Hour)); !errors.Is(err, trip.ErrCompleted) || !v.FinishedAt.Equal(at) {
		t.Fatalf("repeat finish: %+v %v", v, err)
	}
}
