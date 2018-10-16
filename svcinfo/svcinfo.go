package svcinfo // import "github.com/dollarshaveclub/go-productionize/svcinfo"

import (
	"fmt"
	"os"
)

const (
	commitSHAEnv = "COMMIT_SHA"
	buildDateEnv = "BUILD_DATE"
	versionEnv   = "VERSION"
)

var (
	// The following should be set at compile time if they are wanted.
	// -ldflags "-X github.com/dollarshaveclub/go-productionize/svcinfo.CommitSHA=$(COMMIT)"

	// CommitSHA is the latest commit for the built the binary
	CommitSHA string
	// BuildDate is the date for the binary build
	BuildDate string
	// Version is a tagged version for the binary
	Version string
)

// ServiceInfo provides information about the service
type ServiceInfo struct {
	BuildDate string
	CommitSHA string
	Version   string
}

func init() {
	// Default to the compiled in version but allow for environment variables too
	if CommitSHA == "" && os.Getenv(commitSHAEnv) != "" {
		CommitSHA = os.Getenv(commitSHAEnv)
	}
	if BuildDate == "" && os.Getenv(buildDateEnv) != "" {
		BuildDate = os.Getenv(buildDateEnv)
	}
	if Version == "" && os.Getenv(versionEnv) != "" {
		Version = os.Getenv(versionEnv)
	}
}

// GetDDTags will return the info from this library into a string slice that is formatted
// to be used with DataDog
func GetDDTags() []string {
	// Build tags for information compiled into the binary
	infoTags := []string{}
	if CommitSHA != "" {
		infoTags = append(infoTags, fmt.Sprintf("commit:%s", CommitSHA))
	}
	if BuildDate != "" {
		infoTags = append(infoTags, fmt.Sprintf("build_date:%s", BuildDate))
	}
	if Version != "" {
		infoTags = append(infoTags, fmt.Sprintf("version:%s", Version))
	}

	return infoTags
}

// GetInfo returns a new struct pointer with the info from this package
func GetInfo() ServiceInfo {
	return ServiceInfo{
		BuildDate: BuildDate,
		CommitSHA: CommitSHA,
		Version:   Version,
	}
}
