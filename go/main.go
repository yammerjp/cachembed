package main

import (
	"github.com/yammerjp/cachembed/cmd/cachembed"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	cachembed.Run(cachembed.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
		BuiltBy: builtBy,
	})
}
