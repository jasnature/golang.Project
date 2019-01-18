package base

type ByteSize int

const (
	Byte ByteSize = 1
	KB   ByteSize = Byte << 10 //1024
	MB   ByteSize = KB << 10   //1048576
	GB   ByteSize = MB << 10   //1073741824
)

type RunOutputModel int

const (
	None          RunOutputModel = 0
	ConsoleOutput RunOutputModel = 1
	FileOutput    RunOutputModel = 2
)
