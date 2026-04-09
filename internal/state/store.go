package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tailstick/tailstick/internal/model"
)

// Load reads the local state file. Returns an empty but valid state if the file
// does not exist yet.
func Load(path string) (model.LocalState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.LocalState{SchemaVersion: 1, UpdatedAt: time.Now().UTC(), Records: []model.LeaseRecord{}}, nil
		}
		return model.LocalState{}, fmt.Errorf("read state: %w", err)
	}
	var st model.LocalState
	if err := json.Unmarshal(b, &st); err != nil {
		return model.LocalState{}, fmt.Errorf("parse state: %w", err)
	}
	if st.SchemaVersion == 0 {
		st.SchemaVersion = 1
	}
	if st.Records == nil {
		st.Records = []model.LeaseRecord{}
	}
	return st, nil
}

// Save writes state to disk atomically. It updates SchemaVersion and UpdatedAt
// on a copy to avoid mutating the caller's struct.
func Save(path string, st model.LocalState) error {
	st.SchemaVersion = 1
	st.UpdatedAt = time.Now().UTC()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// UpsertRecord inserts or replaces a lease record by LeaseID.
func UpsertRecord(st model.LocalState, rec model.LeaseRecord) model.LocalState {
	for i := range st.Records {
		if st.Records[i].LeaseID == rec.LeaseID {
			st.Records[i] = rec
			return st
		}
	}
	st.Records = append(st.Records, rec)
	return st
}

// AppendAudit appends a JSON-encoded audit entry to the log file.
// The entry's Timestamp field is overwritten with the current UTC time.
func AppendAudit(path string, entry model.AuditEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	entry.Timestamp = time.Now().UTC()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}
