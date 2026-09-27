package version

// Version is the current program version.
// It is overridden at build time via -ldflags "-X m3u-merge-astra/version.Version=vX.Y.Z".
var Version = "dev"
