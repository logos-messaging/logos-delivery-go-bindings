// Package ffi is the cgo bridge over the single liblogosdelivery C library,
// which exposes the full logos-delivery API. It is split across two files
// mirroring the two API tiers: this file (liblogosdelivery.go) owns the shared
// plumbing — the reply callback, the pending-call registry, the CBOR request
// encoding and the node lifecycle — plus the stable Messaging API; libwaku.go
// adds the low-level Kernel API (waku_*). It exposes Go-typed primitives so
// pkg/kernel and pkg/messaging stay pure Go.
//
// liblogosdelivery is built on nim-ffi's CBOR ABI. Every entry point except the
// constructor takes the context handle, one FFICallback and a CBOR-encoded
// request map keyed by argument name. On success the callback receives the
// CBOR-encoded reply; on failure, the error text. Event listeners receive the
// event JSON as is.
//
// All callback buffers are borrowed for the duration of the call, so every
// callback here copies before it hands anything back to Go.
package ffi

/*
#cgo LDFLAGS: -llogosdelivery
#include <stdint.h>
#include <stdlib.h>

// The raw liblogosdelivery exports. liblogosdelivery.h is not included: its
// generated helpers need TinyCBOR, and these bindings encode CBOR in Go instead.
#define LOGOS_RET_OK         0
#define LOGOS_RET_STALE_WARN 3

typedef void (*FFICallback)(int ret, const char* msg, size_t len, void* userData);
typedef int (*logosRawFn)(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);

void* logosdelivery_create_node(const uint8_t* req, size_t reqLen, FFICallback callback, void* userData);
int logosdelivery_destroy(void* ctx);
int logosdelivery_start_node(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int logosdelivery_stop_node(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int logosdelivery_subscribe(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int logosdelivery_unsubscribe(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
int logosdelivery_send(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
uint64_t logosdelivery_add_event_listener(void* ctx, const char* eventName, FFICallback callback, void* userData);
int logosdelivery_remove_event_listener(void* ctx, uint64_t listenerId);

// Both callbacks are implemented in Go and exported below. `userData` carries a
// runtime/cgo.Handle cast to void*.
extern void logosReply(int ret, char* msg, size_t len, void* userData);
extern void logosEvent(int ret, char* msg, size_t len, void* userData);

// The wrappers take the cgo.Handle and the context as uintptr_t and widen them
// to void* here: the context is a token, not an address.
static uintptr_t cGoCreateNode(const void* req, size_t reqLen, uintptr_t ud) {
	return (uintptr_t) logosdelivery_create_node((const uint8_t*) req, reqLen, (FFICallback) logosReply, (void*) ud);
}
static int cGoCall(logosRawFn fn, uintptr_t ctx, const void* req, size_t reqLen, uintptr_t ud) {
	return fn((void*) ctx, (FFICallback) logosReply, (void*) ud, (const uint8_t*) req, reqLen);
}
static int cGoDestroy(uintptr_t ctx) {
	return logosdelivery_destroy((void*) ctx);
}
static uint64_t cGoAddEventListener(uintptr_t ctx, const char* eventName, uintptr_t ud) {
	return logosdelivery_add_event_listener((void*) ctx, eventName, (FFICallback) logosEvent, (void*) ud);
}
static int cGoRemoveEventListener(uintptr_t ctx, uint64_t listenerId) {
	return logosdelivery_remove_event_listener((void*) ctx, listenerId);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime/cgo"
	"sync"
	"unsafe"

	"github.com/fxamacker/cbor/v2"
)

// RetOK is the return code callbacks report on success.
const RetOK = C.LOGOS_RET_OK

// staleWarn is the non-terminal "still running" code the library emits every
// few seconds for a long call. It is always followed by a terminal code, so
// the reply callback ignores it rather than settling the pending call.
const staleWarn = C.LOGOS_RET_STALE_WARN

// ListenerID identifies one registered event listener within a node context.
// It is only meaningful together with the Handle it was registered on.
type ListenerID uint64

// EventHandler receives every event liblogosdelivery emits for the event name
// it was registered under: the raw event JSON when ret == RetOK, an error
// message otherwise.
type EventHandler func(ret int, msg string)

// noArgs is the request of an entry point that takes no arguments.
type noArgs struct{}

// pending is one in-flight synchronous call. The C callback settles it and the
// caller reads reply/err after done is closed.
type pending struct {
	done  chan struct{}
	once  sync.Once
	reply []byte
	err   error
}

func newPending() *pending { return &pending{done: make(chan struct{})} }

// settle records a terminal result exactly once and wakes the caller.
func (p *pending) settle(ret C.int, msg []byte) {
	p.once.Do(func() {
		if ret == RetOK {
			p.reply = msg
		} else {
			text := string(msg)
			if text == "" {
				text = fmt.Sprintf("liblogosdelivery call failed (code %d)", int(ret))
			}
			p.err = errors.New(text)
		}
		close(p.done)
	})
}

// goBytes copies a borrowed, length-delimited byte run.
func goBytes(s *C.char, length C.size_t) []byte {
	if s == nil || length == 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(s), C.int(length))
}

//export logosReply
func logosReply(ret C.int, msg *C.char, length C.size_t, userData unsafe.Pointer) {
	if ret == staleWarn {
		return
	}
	p, ok := cgo.Handle(uintptr(userData)).Value().(*pending)
	if !ok {
		return
	}
	p.settle(ret, goBytes(msg, length))
}

//export logosEvent
func logosEvent(ret C.int, msg *C.char, length C.size_t, userData unsafe.Pointer) {
	fn, ok := cgo.Handle(uintptr(userData)).Value().(EventHandler)
	if !ok {
		return
	}
	fn(int(ret), string(goBytes(msg, length)))
}

// await runs one entry point and blocks until its callback reports a terminal
// result, returning the raw reply (on RetOK) or an error built from it. invoke
// receives the CBOR request and the cgo.Handle to pass through as userData, and
// returns the entry point's immediate return code; a non-zero code means the
// call was never dispatched, so no callback will arrive.
func await(req any, invoke func(buf unsafe.Pointer, n C.size_t, ud C.uintptr_t) C.int) ([]byte, error) {
	enc, err := cbor.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	buf := C.CBytes(enc)
	defer C.free(buf)

	p := newPending()
	h := cgo.NewHandle(p)
	defer h.Delete()

	if rc := invoke(buf, C.size_t(len(enc)), C.uintptr_t(h)); rc != RetOK {
		return nil, fmt.Errorf("liblogosdelivery call was not dispatched (code %d)", int(rc))
	}
	<-p.done
	return p.reply, p.err
}

// call runs one context-bound entry point with req as its argument map and
// decodes the string it replies with.
func call(fn C.logosRawFn, h Handle, req any) (string, error) {
	raw, err := await(req, func(buf unsafe.Pointer, n C.size_t, ud C.uintptr_t) C.int {
		return C.cGoCall(fn, C.uintptr_t(h.ctx), buf, n, ud)
	})
	if err != nil {
		return "", err
	}
	var reply string
	if err := cbor.Unmarshal(raw, &reply); err != nil {
		return "", fmt.Errorf("decode reply: %w", err)
	}
	return reply, nil
}

// New builds a node from a configuration JSON string and returns its handle.
// Creation is asynchronous: this waits for the library to report the node ready
// before returning. The handle must be released with Destroy.
func New(configJSON string) (Handle, error) {
	req := struct {
		ConfigJSON string `cbor:"configJson"`
	}{configJSON}

	// The constructor's return value is the context handle; the callback
	// reports whether construction actually succeeded.
	var h Handle
	_, err := await(req, func(buf unsafe.Pointer, n C.size_t, ud C.uintptr_t) C.int {
		h = Handle{uintptr(C.cGoCreateNode(buf, n, ud))}
		return RetOK
	})
	if err != nil {
		return Handle{}, err
	}
	if !h.Valid() {
		return Handle{}, errors.New("logosdelivery_create_node returned no context")
	}
	return h, nil
}

// Start starts the node's protocols and Messaging API services.
func Start(h Handle) error {
	_, err := call(C.logosRawFn(C.logosdelivery_start_node), h, noArgs{})
	return err
}

// Stop stops the node. It can be started again.
func Stop(h Handle) error {
	_, err := call(C.logosRawFn(C.logosdelivery_stop_node), h, noArgs{})
	return err
}

// Destroy releases the node context. Unlike the other entry points it is
// synchronous, and it also drops every event listener registered on the
// context, so h must not be used afterwards.
func Destroy(h Handle) error {
	if rc := C.cGoDestroy(C.uintptr_t(h.ctx)); rc != RetOK {
		return fmt.Errorf("logosdelivery_destroy failed (code %d)", int(rc))
	}
	return nil
}

type contentTopicReq struct {
	ContentTopic string `cbor:"contentTopicStr"`
}

// Subscribe subscribes the node to a content topic.
func Subscribe(h Handle, contentTopic string) error {
	_, err := call(C.logosRawFn(C.logosdelivery_subscribe), h, contentTopicReq{contentTopic})
	return err
}

// Unsubscribe unsubscribes the node from a content topic.
func Unsubscribe(h Handle, contentTopic string) error {
	_, err := call(C.logosRawFn(C.logosdelivery_unsubscribe), h, contentTopicReq{contentTopic})
	return err
}

// Send sends a message (JSON: {contentTopic, payload(base64), ephemeral}) and
// returns the request id used to correlate later send events.
func Send(h Handle, messageJSON string) (requestID string, err error) {
	req := struct {
		MessageJSON string `cbor:"messageJson"`
	}{messageJSON}
	return call(C.logosRawFn(C.logosdelivery_send), h, req)
}

// listeners keeps the cgo.Handle backing each registered listener alive until
// it is removed, keyed by the context and listener id that identify it.
var (
	listenersMu sync.Mutex
	listeners   = make(map[listenerKey]cgo.Handle)
)

type listenerKey struct {
	h  Handle
	id ListenerID
}

// AddEventListener registers fn to receive the named event for the node, and
// returns the id that removes it again. Event names are the library's wire
// names, e.g. "onMessageReceived". Register before Start so no event is missed.
func AddEventListener(h Handle, eventName string, fn EventHandler) (ListenerID, error) {
	cName := C.CString(eventName)
	defer C.free(unsafe.Pointer(cName))

	handle := cgo.NewHandle(fn)
	id := ListenerID(C.cGoAddEventListener(C.uintptr_t(h.ctx), cName, C.uintptr_t(handle)))
	if id == 0 {
		handle.Delete()
		return 0, fmt.Errorf("failed to add %q event listener: invalid context", eventName)
	}

	listenersMu.Lock()
	listeners[listenerKey{h, id}] = handle
	listenersMu.Unlock()
	return id, nil
}

// RemoveEventListener removes a listener previously added with
// AddEventListener. Removing an unknown listener is an error.
func RemoveEventListener(h Handle, id ListenerID) error {
	key := listenerKey{h, id}

	listenersMu.Lock()
	handle, known := listeners[key]
	delete(listeners, key)
	listenersMu.Unlock()

	rc := C.cGoRemoveEventListener(C.uintptr_t(h.ctx), C.uint64_t(id))
	if known {
		handle.Delete()
	}
	if rc != RetOK {
		return fmt.Errorf("failed to remove event listener %d", uint64(id))
	}
	return nil
}
