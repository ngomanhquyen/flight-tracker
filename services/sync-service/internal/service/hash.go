package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/flighttracker/services/sync-service/internal/model"
)

// snapshotHash returns a stable sha256 hex digest of s, used as the cheap
// "did anything change since the last poll" comparison
// (docs/database/sync-service.sql's raw_hash column). encoding/json
// marshals struct fields in declaration order, so this is deterministic
// for a given FlightSnapshot value.
func snapshotHash(s model.FlightSnapshot) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
