package types


import (
"context"
"github.com/33cn/chain33/queue"
"github.com/libp2p/go-libp2p"
"github.com/stretchr/testify/assert"
"reflect"
"testing"
)


func initProtocolMap() {

	protocolTypeMap = make(map[string]reflect.Type)
	streamHandlerTypeMap = make(map[string]reflect.Type)
	eventHandlerMap = make(map[int64]EventHandler)
}

type testProtocol struct {
	BaseProtocol
}

type testProtocol2 struct {
	*BaseProtocol
}

type testStreamHandler struct {
	BaseStreamHandler
}

type testStreamHandler2 struct {
	*BaseStreamHandler
}


func testRegisterProtocolPanic(typeName string, protocol IProtocol) (isPanic bool) {

	defer func() {
		if e := recover(); e != nil {
			isPanic = true
		}
	}()

	RegisterProtocolType(typeName, protocol)
	return false
}


func TestRegisterProtocolType(t *testing.T) {

	initProtocolMap()
	RegisterProtocolType("test1", &testProtocol{})
	RegisterProtocolType("test2", &testProtocol2{})
	RegisterProtocolType("test3", testProtocol2{})
	assert.True(t, testRegisterProtocolPanic("test1", &testProtocol{}))
	assert.True(t, testRegisterProtocolPanic("test", nil))
	assert.Equal(t, int(3), len(protocolTypeMap))
	assert.Equal(t, protocolTypeMap["test1"], reflect.TypeOf(testProtocol{}))
	assert.Equal(t, protocolTypeMap["test2"], reflect.TypeOf(testProtocol2{}))
	assert.Equal(t, protocolTypeMap["test3"], reflect.TypeOf(testProtocol2{}))
}

func testRegisterStreamPanic(typeName, msgID string, stream StreamHandler) (isPanic bool) {

	defer func() {
		if e := recover(); e != nil {
			isPanic = true
		}
	}()

	RegisterStreamHandlerType(typeName, msgID, stream)
	return false
}

func TestRegisterStreamHandlerType(t *testing.T) {

	initProtocolMap()
	RegisterProtocolType("test", &testProtocol{})
	RegisterStreamHandlerType("test", "stream1", &testStreamHandler{})
	RegisterStreamHandlerType("test", "stream2", testStreamHandler2{})
	RegisterStreamHandlerType("test", "stream3", &testStreamHandler2{})

	//invalid protocol type
	assert.True(t, testRegisterStreamPanic("test1", "stream", &testStreamHandler{}))
	//duplicate msg id
	assert.True(t, testRegisterStreamPanic("test", "stream1", &testStreamHandler{}))
	//nil handler
	assert.True(t, testRegisterStreamPanic("test", "stream", nil))

	assert.Equal(t, int(3), len(streamHandlerTypeMap))
	assert.Equal(t, streamHandlerTypeMap[formatHandlerTypeID("test", "stream1")], reflect.TypeOf(testStreamHandler{}))
	assert.Equal(t, streamHandlerTypeMap[formatHandlerTypeID("test", "stream2")], reflect.TypeOf(testStreamHandler2{}))
	assert.Equal(t, streamHandlerTypeMap[formatHandlerTypeID("test", "stream3")], reflect.TypeOf(testStreamHandler2{}))
}

func testEventHandler(*queue.Message) {

}

func testRegisterEventPanic(eventID int64, handler EventHandler) (isPanic bool) {

	defer func() {
		if e := recover(); e != nil {
			isPanic = true
		}
	}()

	RegisterEventHandler(eventID, handler)
	return false
}

func TestRegisterEventHandler(t *testing.T) {

	initProtocolMap()
	RegisterEventHandler(1, testEventHandler)
	RegisterEventHandler(2, testEventHandler)
	assert.True(t, testRegisterEventPanic(1, testEventHandler))
	assert.True(t, testRegisterEventPanic(3, nil))
	assert.Equal(t, int(2), len(eventHandlerMap))
	_, exist := GetEventHandler(1)
	assert.True(t, exist)
}

func TestProtocolManager_Init(t *testing.T) {

	initProtocolMap()

	RegisterProtocolType("base", &BaseProtocol{})
	RegisterProtocolType("test1", &testProtocol{})
	RegisterProtocolType("test2", &testProtocol2{})

	RegisterStreamHandlerType("base", "base", &BaseStreamHandler{})
	RegisterStreamHandlerType("base", "stream1", &testStreamHandler{})
	RegisterStreamHandlerType("base", "stream2", &testStreamHandler2{})
	RegisterStreamHandlerType("test1", "stream3", &testStreamHandler{})
	RegisterStreamHandlerType("test1", "stream4", &testStreamHandler2{})
	RegisterStreamHandlerType("test2", "stream5", &testStreamHandler{})
	RegisterStreamHandlerType("test2", "stream6", &testStreamHandler2{})

	global := &GlobalData{}
	global.Host, _ = libp2p.New(context.Background())
	manager := &ProtocolManager{}
	manager.Init(global)

}
