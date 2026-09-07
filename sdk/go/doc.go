// Package openlicensd is the official Go client for the OpenLicensd public API.
//
// It validates license keys, optionally issues Harbor registry credentials,
// and probes health endpoints. Admin APIs are out of scope.
//
//	client, err := openlicensd.New("https://licenses.example.com", "acme-widget")
//	result, err := client.Validate(ctx, key)
//	if err != nil { /* transport / *APIError */ }
//	if !result.Valid { /* result.Reason */ }
//
// Product is required unless WithAnyProduct is passed. Prefer New with a
// build-time URL in vendor binaries; NewFromEnv is for operator-controlled
// servers. When the server enforces max activations, use Fingerprint with
// WithFingerprint. CachedValidator caches nil-error results (including
// Valid=false) for a TTL. Guard revalidates in the background and is
// fail-closed at construction when the first Validate returns an error.
package openlicensd
