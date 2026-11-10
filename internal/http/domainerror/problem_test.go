package domainerror_test

import (
	"errors"
	"fmt"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/http/domainerror"
	"strings"
	"testing"
)

func TestProblem(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{trip.ErrNotFound, 404, "trip_not_found"}, {trip.ErrCompleted, 409, "trip_completed"}, {trip.ErrDriverBusy, 409, "driver_busy"}, {&trip.ValidationError{Field: "price", Reason: "negative"}, 400, "invalid_request"}, {errors.New("secret database password"), 500, "internal_error"},
	} {
		status, body := domainerror.Problem(fmt.Errorf("operation: %w", tc.err))
		if status != tc.status || body.Code != tc.code || int(body.Status) != status || body.Detail == nil {
			t.Fatalf("problem: %d %+v", status, body)
		}
		if strings.Contains(*body.Detail, "secret") {
			t.Fatalf("internal details exposed: %+v", body)
		}
	}
}
