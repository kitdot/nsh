package core

// NSHHost represents a single Host block in SSH config
type NSHHost struct {
	Group        string
	Desc         string
	Auth         string
	Order        int
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	IsWildcard   bool
	// Raw property lines (excluding the Host line itself) for lossless output
	RawPropertyLines []string
}
