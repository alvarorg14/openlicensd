package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const maxMachinesPerLicense = 1000

type Machine struct {
	ID              uuid.UUID
	LicenseID       uuid.UUID
	Fingerprint     string
	Name            *string
	Hostname        *string
	FirstSeenAt     time.Time
	LastSeenAt      time.Time
	LastSeenIP      *string
	ValidationCount int64
	DeactivatedAt   *time.Time
	DeactivatedBy   *uuid.UUID
}

type MachineListParams struct {
	ListParams
	LicenseID uuid.UUID
	Status    string // "active", "released", or ""
}

func sanitizeClientString(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	value = b.String()
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	return value
}

const (
	maxFingerprintLen = 128
	maxHostnameLen    = 253
)

func SanitizeFingerprint(value string) string {
	return sanitizeClientString(value, maxFingerprintLen)
}

func SanitizeHostname(value string) string {
	return sanitizeClientString(value, maxHostnameLen)
}

const machineColumns = `
	m.id, m.license_id, m.fingerprint, m.name, m.hostname,
	m.first_seen_at, m.last_seen_at, m.last_seen_ip, m.validation_count,
	m.deactivated_at, m.deactivated_by
`

// RecordActivation registers or refreshes a machine for a license.
// When max is non-nil, new or reactivated machines consume a seat.
// Returns allowed=false when the activation limit is reached.
func (s *Store) RecordActivation(ctx context.Context, licenseID uuid.UUID, fingerprint, hostname, ip string, max *int) (*Machine, bool, error) {
	fingerprint = SanitizeFingerprint(fingerprint)
	if fingerprint == "" {
		return nil, false, fmt.Errorf("fingerprint is required")
	}
	hostname = SanitizeHostname(hostname)
	if ip == "" {
		ip = ""
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM licenses WHERE id = $1)`, licenseID).Scan(&exists); err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	if _, err := tx.Exec(ctx, `SELECT id FROM licenses WHERE id = $1 FOR UPDATE`, licenseID); err != nil {
		return nil, false, err
	}

	var machine Machine
	var name, storedHostname, lastSeenIP *string
	err = tx.QueryRow(ctx, `
		SELECT `+machineColumns+`
		FROM license_machines m
		WHERE m.license_id = $1 AND m.fingerprint = $2
	`, licenseID, fingerprint).Scan(
		&machine.ID, &machine.LicenseID, &machine.Fingerprint, &name, &storedHostname,
		&machine.FirstSeenAt, &machine.LastSeenAt, &lastSeenIP, &machine.ValidationCount,
		&machine.DeactivatedAt, &machine.DeactivatedBy,
	)
	machine.Name = name
	machine.Hostname = storedHostname
	machine.LastSeenIP = lastSeenIP

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	needsSeat := errors.Is(err, pgx.ErrNoRows) || machine.DeactivatedAt != nil

	if needsSeat && max != nil {
		var activeCount int64
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)::bigint
			FROM license_machines
			WHERE license_id = $1 AND deactivated_at IS NULL
		`, licenseID).Scan(&activeCount); err != nil {
			return nil, false, err
		}
		if int(activeCount) >= *max {
			return nil, false, nil
		}
	}

	if needsSeat && max == nil {
		var activeCount int64
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)::bigint
			FROM license_machines
			WHERE license_id = $1 AND deactivated_at IS NULL
		`, licenseID).Scan(&activeCount); err != nil {
			return nil, false, err
		}
		if activeCount >= maxMachinesPerLicense {
			return nil, false, nil
		}
	}

	var hostnameArg any
	if hostname != "" {
		hostnameArg = hostname
	}
	var ipArg any
	if ip != "" {
		ipArg = ip
	}

	if errors.Is(err, pgx.ErrNoRows) {
		const insertFull = `
			INSERT INTO license_machines (license_id, fingerprint, hostname, last_seen_ip, validation_count)
			VALUES ($1, $2, $3, $4, 1)
			RETURNING id, license_id, fingerprint, name, hostname,
				first_seen_at, last_seen_at, last_seen_ip, validation_count,
				deactivated_at, deactivated_by
		`
		err = tx.QueryRow(ctx, insertFull, licenseID, fingerprint, hostnameArg, ipArg).Scan(
			&machine.ID, &machine.LicenseID, &machine.Fingerprint, &name, &storedHostname,
			&machine.FirstSeenAt, &machine.LastSeenAt, &lastSeenIP, &machine.ValidationCount,
			&machine.DeactivatedAt, &machine.DeactivatedBy,
		)
		if err != nil {
			return nil, false, err
		}
		machine.Name = name
		machine.Hostname = storedHostname
		machine.LastSeenIP = lastSeenIP
	} else {
		const updateQ = `
			UPDATE license_machines
			SET last_seen_at = NOW(),
			    validation_count = validation_count + 1,
			    hostname = COALESCE($3, hostname),
			    last_seen_ip = COALESCE($4, last_seen_ip),
			    deactivated_at = NULL,
			    deactivated_by = NULL
			WHERE id = $1 AND license_id = $2
			RETURNING id, license_id, fingerprint, name, hostname,
				first_seen_at, last_seen_at, last_seen_ip, validation_count,
				deactivated_at, deactivated_by
		`
		err = tx.QueryRow(ctx, updateQ, machine.ID, licenseID, hostnameArg, ipArg).Scan(
			&machine.ID, &machine.LicenseID, &machine.Fingerprint, &name, &storedHostname,
			&machine.FirstSeenAt, &machine.LastSeenAt, &lastSeenIP, &machine.ValidationCount,
			&machine.DeactivatedAt, &machine.DeactivatedBy,
		)
		if err != nil {
			return nil, false, err
		}
		machine.Name = name
		machine.Hostname = storedHostname
		machine.LastSeenIP = lastSeenIP
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}

	return &machine, true, nil
}

func (s *Store) CountActiveMachines(ctx context.Context, licenseID uuid.UUID) (int64, error) {
	const q = `
		SELECT COUNT(*)::bigint
		FROM license_machines
		WHERE license_id = $1 AND deactivated_at IS NULL
	`
	var count int64
	err := s.pool.QueryRow(ctx, q, licenseID).Scan(&count)
	return count, err
}

func (s *Store) ListLicenseMachines(ctx context.Context, params MachineListParams) ([]Machine, int64, error) {
	qb := newQueryBuilder()
	qb.add("m.license_id = $%d::uuid", params.LicenseID)

	switch params.Status {
	case "active":
		qb.addExpr("m.deactivated_at IS NULL")
	case "released":
		qb.addExpr("m.deactivated_at IS NOT NULL")
	}

	sortExpr := params.Sort
	if sortExpr == "" {
		sortExpr = "m.last_seen_at"
	}
	orderBy := buildOrderBy(sortExpr, params.Order, "m.id")

	q := `
		SELECT ` + machineColumns + `, COUNT(*) OVER() AS total_count
		FROM license_machines m` + qb.whereClause() + orderBy + limitOffsetClause(len(qb.args)+1)

	args := append(qb.args, params.Limit, params.Offset)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanMachinesWithTotal(rows)
}

func (s *Store) GetMachine(ctx context.Context, licenseID, machineID uuid.UUID) (*Machine, error) {
	const q = `
		SELECT ` + machineColumns + `
		FROM license_machines m
		WHERE m.license_id = $1 AND m.id = $2
	`
	row := s.pool.QueryRow(ctx, q, licenseID, machineID)
	return scanMachine(row)
}

func (s *Store) UpdateMachineName(ctx context.Context, licenseID, machineID uuid.UUID, name *string) (*Machine, error) {
	var nameArg any
	if name != nil {
		trimmed := sanitizeClientString(*name, 255)
		if trimmed == "" {
			nameArg = nil
		} else {
			nameArg = trimmed
		}
	}

	const q = `
		UPDATE license_machines
		SET name = $3
		WHERE license_id = $1 AND id = $2
		RETURNING id, license_id, fingerprint, name, hostname,
			first_seen_at, last_seen_at, last_seen_ip, validation_count,
			deactivated_at, deactivated_by
	`

	row := s.pool.QueryRow(ctx, q, licenseID, machineID, nameArg)
	machine, err := scanMachine(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return machine, nil
}

func (s *Store) DeactivateMachine(ctx context.Context, licenseID, machineID uuid.UUID, byUserID *uuid.UUID) (*Machine, error) {
	const q = `
		UPDATE license_machines
		SET deactivated_at = NOW(), deactivated_by = $3
		WHERE license_id = $1 AND id = $2 AND deactivated_at IS NULL
		RETURNING id, license_id, fingerprint, name, hostname,
			first_seen_at, last_seen_at, last_seen_ip, validation_count,
			deactivated_at, deactivated_by
	`

	row := s.pool.QueryRow(ctx, q, licenseID, machineID, byUserID)
	machine, err := scanMachine(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return machine, nil
}

func scanMachine(row pgx.Row) (*Machine, error) {
	var m Machine
	var name, hostname, lastSeenIP *string
	err := row.Scan(
		&m.ID, &m.LicenseID, &m.Fingerprint, &name, &hostname,
		&m.FirstSeenAt, &m.LastSeenAt, &lastSeenIP, &m.ValidationCount,
		&m.DeactivatedAt, &m.DeactivatedBy,
	)
	if err != nil {
		return nil, err
	}
	m.Name = name
	m.Hostname = hostname
	m.LastSeenIP = lastSeenIP
	return &m, nil
}

func scanMachinesWithTotal(rows pgx.Rows) ([]Machine, int64, error) {
	var (
		machines   []Machine
		totalCount int64
	)
	for rows.Next() {
		var m Machine
		var name, hostname, lastSeenIP *string
		if err := rows.Scan(
			&m.ID, &m.LicenseID, &m.Fingerprint, &name, &hostname,
			&m.FirstSeenAt, &m.LastSeenAt, &lastSeenIP, &m.ValidationCount,
			&m.DeactivatedAt, &m.DeactivatedBy,
			&totalCount,
		); err != nil {
			return nil, 0, err
		}
		m.Name = name
		m.Hostname = hostname
		m.LastSeenIP = lastSeenIP
		machines = append(machines, m)
	}
	return machines, totalCount, rows.Err()
}

func MachineDisplayName(m *Machine) string {
	if m.Name != nil && *m.Name != "" {
		return *m.Name
	}
	if m.Hostname != nil && *m.Hostname != "" {
		return *m.Hostname
	}
	fp := m.Fingerprint
	if len(fp) > 12 {
		return fp[:12] + "…"
	}
	return fp
}
