// Package version exposes the build version injected at link time.
package version

// Version is overridden with -ldflags "-X github.com/openlicensd/openlicensd/server/internal/version.Version=vX.Y.Z".
var Version = "dev"
