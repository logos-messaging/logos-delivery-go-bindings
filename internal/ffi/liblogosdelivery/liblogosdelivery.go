// Package liblogosdelivery is the cgo bridge over liblogosdelivery (the
// logos-delivery Messaging API C library). It owns the synchronous
// request/response callback plumbing, the shared async event callback, and the
// handle->handler registry, and exposes Go-typed primitives so the public
// messaging package stays pure Go.
//
// It links liblogosdelivery via a #cgo directive; it must never be linked into
// the same binary as the libwaku bridge (overlapping symbols) until
// logos-delivery#3851 consolidates the two libraries.
package liblogosdelivery

/*
#cgo LDFLAGS: -llogosdelivery
#include <liblogosdelivery.h>
#include <stdlib.h>

// logosGoCallback (sync request/response) and logosEventCallback (async events)
// are implemented in Go and exported below.
extern void logosGoCallback(int ret, char* msg, size_t len, void* resp);
extern void logosEventCallback(int ret, char* msg, size_t len, void* userData);

// logosResp carries a single synchronous call's result back from the callback,
// plus a pointer to the Go sync.WaitGroup the caller blocks on.
typedef struct {
	int    ret;
	char*  msg;
	size_t len;
	void*  wg;
} logosResp;

static void* allocLogosResp(void* wg) {
	logosResp* r = (logosResp*) calloc(1, sizeof(logosResp));
	r->wg = wg;
	return r;
}
static void   freeLogosResp(void* resp) { if (resp != NULL) free(resp); }
static char*  logosRespMsg(void* resp)  { return resp ? ((logosResp*)resp)->msg : NULL; }
static size_t logosRespLen(void* resp)  { return resp ? ((logosResp*)resp)->len : 0; }
static int    logosRespRet(void* resp)  { return resp ? ((logosResp*)resp)->ret : RET_ERR; }

// Thin wrappers binding the shared Go callback to each entry point.
static void* cGoCreateNode(const char* cfg, void* resp) {
	return logosdelivery_create_node(cfg, (FFICallBack) logosGoCallback, resp);
}
static void cGoStartNode(void* ctx, void* resp) {
	logosdelivery_start_node(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoStopNode(void* ctx, void* resp) {
	logosdelivery_stop_node(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoDestroyNode(void* ctx, void* resp) {
	logosdelivery_destroy(ctx, (FFICallBack) logosGoCallback, resp);
}
static void cGoSubscribe(void* ctx, const char* topic, void* resp) {
	logosdelivery_subscribe(ctx, (FFICallBack) logosGoCallback, resp, topic);
}
static void cGoUnsubscribe(void* ctx, const char* topic, void* resp) {
	logosdelivery_unsubscribe(ctx, (FFICallBack) logosGoCallback, resp, topic);
}
static void cGoSend(void* ctx, const char* msgJson, void* resp) {
	logosdelivery_send(ctx, (FFICallBack) logosGoCallback, resp, msgJson);
}
static void cGoSetEventCallback(void* ctx) {
	// ctx doubles as userData so the shared event callback can route the event
	// to the right registered handler.
	logosdelivery_set_event_callback(ctx, (FFICallBack) logosEventCallback, ctx);
}
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

// Handle is an opaque pointer to a node context owned by the C library.
type Handle = unsafe.Pointer

// RetOK is the return code callbacks report on success.
const RetOK = C.RET_OK

// EventHandler receives every event liblogosdelivery emits for a node: the raw
// event JSON when ret == RetOK, an error message otherwise.
type EventHandler func(ret int, msg string)

// eventHandlers maps a node handle to the Go function that receives its
// events. The shared C event callback looks the handler up by handle.
var (
	eventHandlersMu sync.RWMutex
	eventHandlers   = make(map[Handle]EventHandler)
)

//export logosGoCallback
func logosGoCallback(ret C.int, msg *C.char, length C.size_t, resp unsafe.Pointer) {
	if resp == nil {
		return
	}
	r := (*C.logosResp)(resp)
	r.ret = ret
	r.msg = msg
	r.len = length
	wg := (*sync.WaitGroup)(r.wg)
	wg.Done()
}

//export logosEventCallback
func logosEventCallback(ret C.int, msg *C.char, length C.size_t, userData unsafe.Pointer) {
	eventHandlersMu.RLock()
	fn := eventHandlers[userData] // userData carries the node's handle
	eventHandlersMu.RUnlock()
	if fn != nil {
		fn(int(ret), C.GoStringN(msg, C.int(length)))
	}
}

// call runs a synchronous entry point that reports its result through the
// response callback, blocks until it completes, and returns the callback
// message (on RetOK) or an error built from it.
func call(invoke func(resp unsafe.Pointer)) (string, error) {
	var wg sync.WaitGroup
	wg.Add(1)
	resp := C.allocLogosResp(unsafe.Pointer(&wg))
	defer C.freeLogosResp(resp)

	invoke(resp)
	wg.Wait()

	msg := C.GoStringN(C.logosRespMsg(resp), C.int(C.logosRespLen(resp)))
	if C.logosRespRet(resp) != C.RET_OK {
		return "", errors.New(msg)
	}
	return msg, nil
}

// New builds a node from a WakuNodeConf JSON string and returns its handle.
// The handle must be released with Destroy.
func New(configJSON string) (Handle, error) {
	cCfg := C.CString(configJSON)
	defer C.free(unsafe.Pointer(cCfg))

	var wg sync.WaitGroup
	wg.Add(1)
	resp := C.allocLogosResp(unsafe.Pointer(&wg))
	defer C.freeLogosResp(resp)

	ctx := C.cGoCreateNode(cCfg, resp)
	wg.Wait()

	if C.logosRespRet(resp) != C.RET_OK || ctx == nil {
		msg := C.GoStringN(C.logosRespMsg(resp), C.int(C.logosRespLen(resp)))
		if msg == "" {
			msg = "logosdelivery_create_node returned no context"
		}
		return nil, errors.New(msg)
	}
	return Handle(ctx), nil
}

// SetEventHandler registers fn to receive events for the node and wires up the
// underlying C event callback. Call before Start.
func SetEventHandler(h Handle, fn EventHandler) {
	eventHandlersMu.Lock()
	eventHandlers[h] = fn
	eventHandlersMu.Unlock()
	C.cGoSetEventCallback(h)
}

// Start starts the node's protocols and Messaging API services.
func Start(h Handle) error {
	_, err := call(func(resp unsafe.Pointer) { C.cGoStartNode(h, resp) })
	return err
}

// Stop stops the node.
func Stop(h Handle) error {
	_, err := call(func(resp unsafe.Pointer) { C.cGoStopNode(h, resp) })
	return err
}

// Destroy releases the node context and unregisters its event handler.
func Destroy(h Handle) error {
	_, err := call(func(resp unsafe.Pointer) { C.cGoDestroyNode(h, resp) })
	eventHandlersMu.Lock()
	delete(eventHandlers, h)
	eventHandlersMu.Unlock()
	return err
}

// Subscribe subscribes the node to a content topic.
func Subscribe(h Handle, contentTopic string) error {
	cTopic := C.CString(contentTopic)
	defer C.free(unsafe.Pointer(cTopic))
	_, err := call(func(resp unsafe.Pointer) { C.cGoSubscribe(h, cTopic, resp) })
	return err
}

// Unsubscribe unsubscribes the node from a content topic.
func Unsubscribe(h Handle, contentTopic string) error {
	cTopic := C.CString(contentTopic)
	defer C.free(unsafe.Pointer(cTopic))
	_, err := call(func(resp unsafe.Pointer) { C.cGoUnsubscribe(h, cTopic, resp) })
	return err
}

// Send sends a message (JSON: {contentTopic, payload(base64), ephemeral}) and
// returns the request id used to correlate later send events.
func Send(h Handle, messageJSON string) (requestID string, err error) {
	cMsg := C.CString(messageJSON)
	defer C.free(unsafe.Pointer(cMsg))
	return call(func(resp unsafe.Pointer) { C.cGoSend(h, cMsg, resp) })
}
