package version

import "strings"

var (
	version   = "development"
	goVersion string
	buildTime string
	gitCommit string
	gitTag    string
	gitStatus string
)

// VerbosInfo returns verbos version info
func VerbosInfo() string {
	o := strings.Builder{}
	o.WriteString("Version:   " + version + "\n")
	o.WriteString("GoVersion: " + goVersion + "\n")
	o.WriteString("BuildTime: " + buildTime + "\n")
	o.WriteString("GitCommit: " + gitCommit + "\n")
	o.WriteString("GitTag:    " + gitTag + "\n")
	o.WriteString("GitStatus: " + gitStatus)
	return o.String()
}

// Version returns simple version
func Version() string {
	return version
}
