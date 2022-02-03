package terminal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pytimer/k8sutil/wsremotecommand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
)

type ExecutorType string

const (
	// EndOfTransmission end
	EndOfTransmission = "\u0004"

	WebsocketExecutorType ExecutorType = "websocket"
	SPDYExecutorType      ExecutorType = "spdy"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ExecOptions struct {
	Stdin    bool
	Stdout   bool
	Stderr   bool
	TTY      bool
	Executor ExecutorType
}

type TerminalSession struct {
	wsConn   *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}
	client   kubernetes.Interface
	once     sync.Once
}

func NewTerminalSession(c kubernetes.Interface, w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*TerminalSession, error) {
	subprotocols := websocket.Subprotocols(r)
	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}
	upgrader.Subprotocols = subprotocols

	return &TerminalSession{
		wsConn:   conn,
		sizeChan: make(chan remotecommand.TerminalSize),
		doneChan: make(chan struct{}),
		client:   c,
	}, nil
}

func (t *TerminalSession) Close() error {
	return t.wsConn.Close()
}

func (t *TerminalSession) Done() {
	t.once.Do(func() {
		close(t.doneChan)
	})
}

// watchAndCloseRemoteExecutor close container websocket process when TerminalSession have done.
func (t *TerminalSession) watchAndCloseRemoteExecutor(executor *wsremotecommand.Executor) {
	klog.V(5).Info("listen terminal session stream.")
	select {
	case <-t.doneChan:
		klog.V(5).Info("executor closing...")
		klog.V(5).Info(executor.Close())
	}
}

func (t *TerminalSession) Read(p []byte) (n int, err error) {
	mt, message, err := t.wsConn.ReadMessage()
	if err != nil {
		klog.Errorf("read message err: %v", err)
		return copy(p, EndOfTransmission), err
	}

	klog.V(8).Info(mt, string(message))
	var msg TerminalMessage
	if err := json.Unmarshal([]byte(message), &msg); err != nil {
		return copy(p, EndOfTransmission), err
	}

	switch msg.Op {
	case "stdin":
		return copy(p, msg.Data), nil
	case "resize":
		t.sizeChan <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}
		return 0, nil
	case "ping":
		klog.Info("ping")
		return 0, nil
	}

	return copy(p, EndOfTransmission), fmt.Errorf("unknown message type '%s'", msg.Op)
}

func (t *TerminalSession) Write(p []byte) (n int, err error) {
	msg, err := json.Marshal(TerminalMessage{
		Op:   "stdout",
		Data: string(p),
	})
	if err != nil {
		klog.Errorf("parse write message err: %v", err)
		return 0, err
	}
	klog.V(8).Info(string(p))
	if err := t.wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
		klog.Errorf("write message err: %v", err)
		return 0, err
	}
	return len(p), nil
}

func (t *TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		klog.V(8).Infof("resize container terminal: %#v", size)
		return &size
	case <-t.doneChan:
		return nil
	}
}

// TerminalMessage is the messaging protocol between ShellController and TerminalSession.
//
// OP      DIRECTION  FIELD(S) USED  DESCRIPTION
// ---------------------------------------------------------------------
// bind    fe->be     SessionID      Id sent back from TerminalResponse
// stdin   fe->be     Data           Keystrokes/paste buffer
// resize  fe->be     Rows, Cols     New terminal size
// stdout  be->fe     Data           Output from the process
type TerminalMessage struct {
	Op   string `json:"op"`
	Data string `json:"data"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}
