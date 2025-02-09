package cachembed

import (
	"fmt"
	"os"
)

func runVersion() {
	tmpl := `cachembed version %s
  commit: %s
  date: %s
  built by: %s
`
	fmt.Printf(tmpl, buildInfo.Version, buildInfo.Commit, buildInfo.Date, buildInfo.BuiltBy)
	os.Exit(0)
}
