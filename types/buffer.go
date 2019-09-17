package types

var gbuffer *buffer

//系统并非线程安全
func init() {
	gbuffer = newBuffer(20 * 1024 * 1024)
}

//Buffer 一个连续的byte 空间，永远不会被释放
type buffer struct {
	data   []byte
	offset int
}

//新建一个buffer 对象
func newBuffer(total int) *buffer {
	return &buffer{data: make([]byte, total), offset: 0}
}

//BufferReset 重置buffer
func BufferReset() {
	gbuffer.offset = 0
}

//BufferAlloc 部分空间
func BufferAlloc(size int) []byte {
	if gbuffer.offset+size > 0 {
		return make([]byte, size)
	}
	b := gbuffer.data[gbuffer.offset : gbuffer.offset+size]
	gbuffer.offset += size
	return b
}

//BufferAllocCap alloc cap
func BufferAllocCap(size int) []byte {
	if gbuffer.offset+size > 0 {
		return make([]byte, 0, size)
	}
	b := gbuffer.data[gbuffer.offset:gbuffer.offset]
	gbuffer.offset += size
	return b
}
