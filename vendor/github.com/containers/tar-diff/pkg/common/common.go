package common

const (
	DeltaOpData    = iota
	DeltaOpOpen    = iota
	DeltaOpCopy    = iota
	DeltaOpAddData = iota
	DeltaOpSeek    = iota
)

var DeltaHeader = [...]byte{'t', 'a', 'r', 'd', 'f', '1', '\n', 0}
