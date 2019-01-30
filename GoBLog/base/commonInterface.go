package base

type IDispose interface {
	//please use defer call this method best.
	Dispose() error
}
