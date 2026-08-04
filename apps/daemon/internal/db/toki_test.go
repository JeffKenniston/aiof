package db

import (
	"encoding/json"
	"testing"
)

func TestLEVCComparison(t *testing.T) {
	c1 := NewLEVC(1, 100, 1)
	c2 := NewLEVC(1, 100, 2)
	c3 := NewLEVC(1, 101, 1)
	c4 := NewLEVC(2, 1, 1)

	if c1.Compare(c1) != 0 {
		t.Errorf("Expected c1.Compare(c1) == 0")
	}

	if c1.Compare(c2) >= 0 {
		t.Errorf("Expected c1 < c2")
	}

	if c2.Compare(c3) >= 0 {
		t.Errorf("Expected c2 < c3")
	}

	if c3.Compare(c4) >= 0 {
		t.Errorf("Expected c3 < c4")
	}

	if !c2.IsMonotonicAfter(c1) {
		t.Errorf("Expected c2 to be monotonic after c1")
	}

	if c1.IsMonotonicAfter(c2) {
		t.Errorf("Did not expect c1 to be monotonic after c2")
	}
}

func TestLEVCIncrementAndEpoch(t *testing.T) {
	c1 := NewLEVC(1, 10, 5)

	c2 := c1.IncrementLSN()
	if c2.LSN != 11 || c2.EpochID != 1 || c2.TransactionID != 5 {
		t.Errorf("IncrementLSN failed: got %s", c2)
	}

	c3 := c1.IncrementTxID()
	if c3.TransactionID != 6 || c3.LSN != 10 || c3.EpochID != 1 {
		t.Errorf("IncrementTxID failed: got %s", c3)
	}

	c4 := c1.NextEpoch(2)
	if c4.EpochID != 2 || c4.LSN != 1 || c4.TransactionID != 1 {
		t.Errorf("NextEpoch failed: got %s", c4)
	}
}

func TestLEVCFormattingAndParsing(t *testing.T) {
	c1 := NewLEVC(5, 120, 3)
	str := c1.String()

	parsed, err := ParseLEVC(str)
	if err != nil {
		t.Fatalf("ParseLEVC failed: %v", err)
	}

	if !parsed.Equals(c1) {
		t.Errorf("Parsed LEVC %s != original LEVC %s", parsed, c1)
	}

	altStr := "5:120:3"
	parsedAlt, err := ParseLEVC(altStr)
	if err != nil {
		t.Fatalf("ParseLEVC with alt format failed: %v", err)
	}
	if !parsedAlt.Equals(c1) {
		t.Errorf("Parsed alt LEVC %s != original LEVC %s", parsedAlt, c1)
	}
}

func TestLEVCJSON(t *testing.T) {
	c1 := NewLEVC(2, 45, 12)
	data, err := json.Marshal(c1)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var unmarshaled LEVC
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if !unmarshaled.Equals(c1) {
		t.Errorf("Unmarshaled LEVC %s != original %s", unmarshaled, c1)
	}
}

func TestMonotonicWALFence(t *testing.T) {
	initial := NewLEVC(1, 1, 1)
	fence := NewMonotonicWALFence(initial)

	// Valid monotonic advance
	c2 := NewLEVC(1, 2, 1)
	if err := fence.ValidateAndAdvance(c2); err != nil {
		t.Fatalf("Valid monotonic advance failed: %v", err)
	}

	if !fence.GetLastClock().Equals(c2) {
		t.Errorf("Fence clock not updated to c2")
	}

	// Clock regression attempt (Equal clock)
	if err := fence.ValidateAndAdvance(c2); err == nil {
		t.Errorf("Expected clock regression error for duplicate clock, got nil")
	}

	// Clock regression attempt (Older clock)
	cOlder := NewLEVC(1, 1, 5)
	if err := fence.ValidateAndAdvance(cOlder); err == nil {
		t.Errorf("Expected clock regression error for older clock, got nil")
	}

	// Valid epoch advance
	cNewEpoch := NewLEVC(2, 1, 1)
	if err := fence.ValidateAndAdvance(cNewEpoch); err != nil {
		t.Fatalf("Valid epoch advance failed: %v", err)
	}

	if !fence.GetLastClock().Equals(cNewEpoch) {
		t.Errorf("Fence clock not updated to new epoch clock")
	}
}
