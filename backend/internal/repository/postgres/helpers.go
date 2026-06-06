package postgres

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// dbTime is a time.Time wrapper that scans from both time.Time and string
// values returned by the pgx stdlib driver.
type dbTime struct {
	Time time.Time
}

func (t *dbTime) Scan(src any) error {
	switch v := src.(type) {
	case time.Time:
		t.Time = v
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return fmt.Errorf("dbTime: parse %q: %w", v, err)
		}
		t.Time = parsed
	case nil:
		t.Time = time.Time{}
	default:
		return fmt.Errorf("dbTime: unsupported type %T", src)
	}
	return nil
}

func (t dbTime) Value() (driver.Value, error) {
	return t.Time, nil
}

// dbNullTime is a nullable time.Time that scans NULL as a nil *time.Time.
type dbNullTime struct {
	Time  time.Time
	Valid bool
}

func (t *dbNullTime) Scan(src any) error {
	if src == nil {
		t.Valid = false
		return nil
	}
	t.Valid = true
	switch v := src.(type) {
	case time.Time:
		t.Time = v
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return fmt.Errorf("dbNullTime: parse %q: %w", v, err)
		}
		t.Time = parsed
	default:
		return fmt.Errorf("dbNullTime: unsupported type %T", src)
	}
	return nil
}

// Ptr returns a pointer to the time, or nil when not valid.
func (t dbNullTime) Ptr() *time.Time {
	if !t.Valid {
		return nil
	}
	cp := t.Time
	return &cp
}

// dbNullUUID is a nullable uuid.UUID that scans NULL as a nil *uuid.UUID.
type dbNullUUID struct {
	UUID  uuid.UUID
	Valid bool
}

func (u *dbNullUUID) Scan(src any) error {
	if src == nil {
		u.Valid = false
		return nil
	}
	u.Valid = true
	switch v := src.(type) {
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return fmt.Errorf("dbNullUUID: parse %q: %w", v, err)
		}
		u.UUID = parsed
	case [16]byte:
		u.UUID = uuid.UUID(v)
	default:
		return fmt.Errorf("dbNullUUID: unsupported type %T", src)
	}
	return nil
}

// Ptr returns a pointer to the UUID, or nil when not valid.
func (u dbNullUUID) Ptr() *uuid.UUID {
	if !u.Valid {
		return nil
	}
	cp := u.UUID
	return &cp
}

// parseUUID parses a string into a uuid.UUID with a standardised error message.
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid %q: %w", s, err)
	}
	return id, nil
}

// allowedSort maps a caller-supplied column name to a safe SQL identifier using
// the provided whitelist. Returns defaultCol when the input is not whitelisted.
func allowedSort(col, defaultCol string, allowed map[string]string) string {
	if mapped, ok := allowed[col]; ok {
		return mapped
	}
	return defaultCol
}

// normSortDir returns "ASC" or "DESC", defaulting to "ASC" for any other input.
func normSortDir(dir string) string {
	if dir == "desc" || dir == "DESC" {
		return "DESC"
	}
	return "ASC"
}

// pageOffset converts 1-based page + pageSize into SQL LIMIT / OFFSET values.
// page < 1 is clamped to 1; pageSize < 1 defaults to 20.
func pageOffset(page, pageSize int) (limit, offset int) {
	if pageSize < 1 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}
	return pageSize, (page - 1) * pageSize
}
