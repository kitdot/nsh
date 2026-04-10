package core

// BlockType represents the type of a block in SSH config
type BlockType int

const (
	BlockHost BlockType = iota
	BlockMatch
	BlockInclude
	BlockComment
	BlockBlank
)

// NSHBlock represents a block in the SSH config file
type NSHBlock struct {
	Type     BlockType
	Host     *NSHHost // only for BlockHost
	NshLine  string   // original "# nsh: ..." line (empty if none)
	HostLine string   // original "Host xxx" line
	Raw      string   // raw content for Match/Include/Comment/Blank
}
