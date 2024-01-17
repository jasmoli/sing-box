//go:build go1.27

package v2rayhttp

import (
	"net/http"
	"sync"
	"unsafe"

	"golang.org/x/net/http2"
)

// net/http/internal/http2.Transport
type internalTransport struct {
	t1       [2]uintptr // TransportConfig
	connPool *clientConnPool
}

// net/http/internal/http2.clientConnPool
type clientConnPool struct {
	t     *internalTransport
	mu    sync.Mutex
	conns map[string][]unsafe.Pointer // key is host:port, value is []*ClientConn
}

func closeHTTP2Connections(transport *http2.Transport) {
	h2Transport := transportFromH1Transport(transportInit(transport))
	t := (*internalTransport)((*efaceWords)(unsafe.Pointer(&h2Transport)).data)
	if t == nil {
		return
	}
	p := t.connPool
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, vv := range p.conns {
		for _, cc := range vv {
			clientConnClose(cc)
		}
	}
}

//go:linkname transportInit golang.org/x/net/http2.(*Transport).init
func transportInit(t *http2.Transport) *http.Transport

//go:linkname transportFromH1Transport net/http/internal/http2_test.transportFromH1Transport
func transportFromH1Transport(t *http.Transport) any

//go:linkname clientConnClose net/http/internal/http2.(*ClientConn).Close
func clientConnClose(cc unsafe.Pointer) error
