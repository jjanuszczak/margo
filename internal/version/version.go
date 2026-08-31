package version

import "runtime/debug"

const Name = "margo"

var Version = "0.0.0-dev"

// Current returns the release version embedded by the release build. When a
// binary was installed with `go install module/cmd@version`, Go's build
// metadata provides the module version instead.
func Current() string {
	if Version != "0.0.0-dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return Version
}
