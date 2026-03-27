package buildinfo

import "fmt"

type Info struct {
	Version string
	Commit  string
	Date    string
}

func (i Info) String() string {
	return fmt.Sprintf("version=%s commit=%s date=%s", i.Version, i.Commit, i.Date)
}
