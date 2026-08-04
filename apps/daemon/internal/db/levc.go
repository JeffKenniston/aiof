package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// LEVC represents a Logical Epoch Vector Clock governing monotonic consistency
// and bitemporal ordering across agent operations per ADR-13.
type LEVC struct {
	EpochID       uint64 `json:"epoch_id" db:"epoch_id"`
	LSN           uint64 `json:"lsn" db:"lsn"`
	TransactionID uint64 `json:"transaction_id" db:"transaction_id"`
}

// NewLEVC creates a new Logical Epoch Vector Clock instance.
func NewLEVC(epochID, lsn, txID uint64) LEVC {
	return LEVC{
		EpochID:       epochID,
		LSN:           lsn,
		TransactionID: txID,
	}
}

// Compare performs a strict lexicographical vector comparison between two LEVC instances.
// Returns:
//   -1 if l < other
//    0 if l == other
//   +1 if l > other
func (l LEVC) Compare(other LEVC) int {
	if l.EpochID != other.EpochID {
		if l.EpochID < other.EpochID {
			return -1
		}
		return 1
	}
	if l.LSN != other.LSN {
		if l.LSN < other.LSN {
			return -1
		}
		return 1
	}
	if l.TransactionID != other.TransactionID {
		if l.TransactionID < other.TransactionID {
			return -1
		}
		return 1
	}
	return 0
}

// IsMonotonicAfter returns true if this clock is strictly greater than the provided clock.
func (l LEVC) IsMonotonicAfter(other LEVC) bool {
	return l.Compare(other) > 0
}

// Equals returns true if both LEVC instances have identical vector values.
func (l LEVC) Equals(other LEVC) bool {
	return l.Compare(other) == 0
}

// IncrementLSN returns a copy of the clock with LSN incremented by 1.
func (l LEVC) IncrementLSN() LEVC {
	return LEVC{
		EpochID:       l.EpochID,
		LSN:           l.LSN + 1,
		TransactionID: l.TransactionID,
	}
}

// IncrementTxID returns a copy of the clock with TransactionID incremented by 1.
func (l LEVC) IncrementTxID() LEVC {
	return LEVC{
		EpochID:       l.EpochID,
		LSN:           l.LSN,
		TransactionID: l.TransactionID + 1,
	}
}

// NextEpoch returns a copy of the clock advanced to a new EpochID, resetting LSN and TxID to 1.
func (l LEVC) NextEpoch(newEpochID uint64) LEVC {
	return LEVC{
		EpochID:       newEpochID,
		LSN:           1,
		TransactionID: 1,
	}
}

// String returns a canonical string representation of the LEVC vector: "(EpochID, LSN, TransactionID)".
func (l LEVC) String() string {
	return fmt.Sprintf("(%d,%d,%d)", l.EpochID, l.LSN, l.TransactionID)
}

// ParseLEVC parses a string formatted as "(EpochID, LSN, TransactionID)" or "EpochID:LSN:TransactionID".
func ParseLEVC(s string) (LEVC, error) {
	cleaned := strings.TrimSpace(s)
	cleaned = strings.TrimPrefix(cleaned, "(")
	cleaned = strings.TrimSuffix(cleaned, ")")
	
	var parts []string
	if strings.Contains(cleaned, ",") {
		parts = strings.Split(cleaned, ",")
	} else if strings.Contains(cleaned, ":") {
		parts = strings.Split(cleaned, ":")
	} else {
		return LEVC{}, fmt.Errorf("invalid LEVC format: %q", s)
	}

	if len(parts) != 3 {
		return LEVC{}, fmt.Errorf("invalid LEVC vector component count: expected 3, got %d", len(parts))
	}

	epoch, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return LEVC{}, fmt.Errorf("invalid epoch_id in LEVC: %w", err)
	}

	lsn, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return LEVC{}, fmt.Errorf("invalid lsn in LEVC: %w", err)
	}

	txID, err := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
	if err != nil {
		return LEVC{}, fmt.Errorf("invalid transaction_id in LEVC: %w", err)
	}

	return LEVC{
		EpochID:       epoch,
		LSN:           lsn,
		TransactionID: txID,
	}, nil
}

// MarshalJSON implements json.Marshaler interface for LEVC.
func (l LEVC) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		EpochID       uint64 `json:"epoch_id"`
		LSN           uint64 `json:"lsn"`
		TransactionID uint64 `json:"transaction_id"`
		Formatted     string `json:"formatted"`
	}{
		EpochID:       l.EpochID,
		LSN:           l.LSN,
		TransactionID: l.TransactionID,
		Formatted:     l.String(),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface for LEVC.
func (l *LEVC) UnmarshalJSON(data []byte) error {
	type Alias LEVC
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(l),
	}
	return json.Unmarshal(data, aux)
}

// Value implements driver.Valuer for database serialization.
func (l LEVC) Value() (driver.Value, error) {
	return l.String(), nil
}

// Scan implements sql.Scanner for database deserialization.
func (l *LEVC) Scan(src interface{}) error {
	if src == nil {
		*l = LEVC{}
		return nil
	}
	switch v := src.(type) {
	case string:
		parsed, err := ParseLEVC(v)
		if err != nil {
			return err
		}
		*l = parsed
		return nil
	case []byte:
		parsed, err := ParseLEVC(string(v))
		if err != nil {
			return err
		}
		*l = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan %T into LEVC", src)
	}
}
