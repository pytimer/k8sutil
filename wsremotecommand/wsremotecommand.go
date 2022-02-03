package wsremotecommand

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/apiserver/pkg/util/wsstream"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// StreamOptions holds information pertaining to the current streaming session: input/output streams
type StreamOptions struct {
	Stdin *websocket.Conn
}

type Executor struct {
	Upgrader  *WebSocketUpgrader
	transport http.RoundTripper

	method    string
	url       *url.URL
	protocols []string
}

func NewWebSocketExecutor(config *rest.Config, url *url.URL, protocols []string) (*Executor, error) {
	upgradeRoundTripper, wrapper, err := RoundTripperFor(config)
	if err != nil {
		return nil, err
	}

	if protocols == nil || len(protocols) == 0 {
		protocols = append(protocols, wsstream.Base64ChannelWebSocketProtocol)
	}
	upgradeRoundTripper.Dialer.Subprotocols = protocols

	if url.Scheme == "https" {
		url.Scheme = "wss"
	} else {
		url.Scheme = "ws"
	}

	return &Executor{
		method:    "GET",
		url:       url,
		protocols: protocols,
		transport: wrapper,
		Upgrader:  upgradeRoundTripper,
	}, nil
}

// Close send the 'exit\r\n' command to k8s if recevied the kill or exit signal,
// cleanup the tty process in container.
func (e *Executor) Close() error {
	exitCmd := "exit\r\n"
	if e.Upgrader.Conn.Subprotocol() == wsstream.Base64ChannelWebSocketProtocol {
		exitCmd = base64.StdEncoding.EncodeToString([]byte(exitCmd))
	}
	if err := e.Upgrader.Conn.WriteMessage(websocket.TextMessage, []byte("0"+exitCmd)); err != nil {
		klog.Error(err)
	}
	e.Upgrader.Conn.Close()

	return nil
}

func (e *Executor) Stream(options StreamOptions) error {
	req, err := http.NewRequest(e.method, e.url.String(), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := e.transport.RoundTrip(req)
	if err != nil {
		return err
	}
	klog.V(5).Infof("response: %#v", resp)
	defer e.Upgrader.Conn.Close()

	s := streamer{
		remoteStdin: options.Stdin,
		errorChan:   make(chan error),
		subprotocol: e.Upgrader.Conn.Subprotocol(),
	}

	return s.stream(e.Upgrader.Conn)
}

// WebSocketUpgrade knows how to upgrade an HTTP request to one that supports
// multiplexed streams. After RoundTrip() is invoked, Conn will be set and usable.
type WebSocketUpgrader struct {
	//tlsConfig holds the TLS configuration settings to use when connecting
	//to the remote server.
	tlsConfig *tls.Config

	// Dialer is the dialer used to connect.  Used if non-nil.
	Dialer *websocket.Dialer

	Conn *websocket.Conn

	Header http.Header
}

// RoundTripperFor returns a http header with the rest config.
func RoundTripperFor(config *rest.Config) (*WebSocketUpgrader, http.RoundTripper, error) {
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, nil, err
	}

	proxy := http.ProxyFromEnvironment
	if config.Proxy != nil {
		proxy = config.Proxy
	}

	dialer := websocket.Dialer{
		Proxy:            proxy,
		HandshakeTimeout: 60 * time.Second,
		TLSClientConfig:  tlsConfig,
	}

	rt := &WebSocketUpgrader{
		Dialer: &dialer,
	}

	// 不能直接定义header，因为连接k8s需要认证信息等，因此要从rest里获取
	roundTripper, err := rest.HTTPWrappersForConfig(config, rt)
	if err != nil {
		return nil, nil, err
	}

	return rt, roundTripper, nil
}

func (w *WebSocketUpgrader) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := w.Dialer.Dial(r.URL.String(), r.Header)
	if err != nil {
		return nil, err
	}
	w.Conn = conn
	return resp, nil
}
