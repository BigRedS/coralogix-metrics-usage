package scan

import (
	"testing"
	"time"
)

func TestBillingWindowUTC_Rolling(t *testing.T) {
	start, end, incl, err := billingWindowUTC(time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC), 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) || incl != 7 {
		t.Fatalf("got start=%v end=%v incl=%d", start, end, incl)
	}
}

func TestBillingWindowUTC_LastCompleteUTCMonths(t *testing.T) {
	// May 2026 → previous full month is April 2026; N=1 → April only.
	start, end, incl, err := billingWindowUTC(time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) || incl != 30 {
		t.Fatalf("got start=%v end=%v incl=%d", start, end, incl)
	}

	// N=3 from June → March 1 .. May 31 inclusive.
	start, end, incl, err = billingWindowUTC(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	wantStart = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	wantEnd = time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) || incl != 92 {
		t.Fatalf("N=3 got start=%v end=%v incl=%d", start, end, incl)
	}
}

func TestBillingWindowUTC_Errors(t *testing.T) {
	if _, _, _, err := billingWindowUTC(time.Now(), 0, 0); err == nil {
		t.Fatal("expected error for zero days and zero months")
	}
	if _, _, _, err := billingWindowUTC(time.Now(), -1, 0); err == nil {
		t.Fatal("expected error for negative lookback")
	}
}
