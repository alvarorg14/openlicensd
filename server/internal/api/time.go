package api

import "time"

const timeRFC3339 = time.RFC3339

func formatRFC3339(t time.Time) string {
	return t.UTC().Format(timeRFC3339)
}

func formatRFC3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := formatRFC3339(*t)
	return &formatted
}

func formatUnixRFC3339(secs int64) string {
	return time.Unix(secs, 0).UTC().Format(timeRFC3339)
}
