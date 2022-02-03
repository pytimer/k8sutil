package wsremotecommand

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"
)

type streamer struct {
	// web -> server
	remoteStdin *websocket.Conn
	// server -> web
	proxyStream       *websocket.Conn
	errorChan         chan error
	recordCommandChan chan string
	subprotocol       string
}

func (s *streamer) stream(conn *websocket.Conn) error {
	s.proxyStream = conn

	var wg sync.WaitGroup
	s.copyStdin(&wg)
	s.copyStdout(&wg)

	// send the error (or nil if the remote command exited successfully) to the returned
	// error channel, and closes it.
	go func() {
		err, ok := <-s.errorChan
		if ok && err != nil {
			s.errorChan <- fmt.Errorf("error reading from error channel: %s", err)
		} else if ok {
			s.errorChan <- nil
		}
		close(s.errorChan)
	}()

	// we're waiting for stdin/stdout to finish copying
	wg.Wait()
	return <-s.errorChan
}

func (s *streamer) copyStdin(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msgType, msg, err := s.remoteStdin.ReadMessage()
			if err != nil {
				m := formatCloseMessage(err)
				if err := s.proxyStream.WriteMessage(websocket.CloseMessage, m); err != nil {
					klog.Error(err)
					s.errorChan <- err
					klog.Infof("error channel: %d", len(s.errorChan))
					return
				}
				break
			}

			if err := s.proxyStream.WriteMessage(msgType, msg); err != nil {
				klog.Error(err)
				s.errorChan <- err
				return
			}
		}
	}()

}

func (s *streamer) copyStdout(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msgType, msg, err := s.proxyStream.ReadMessage()
			if err != nil {
				m := formatCloseMessage(err)
				if err := s.remoteStdin.WriteMessage(websocket.CloseMessage, m); err != nil {
					klog.Error(err)
					s.errorChan <- err
					return
				}
				break
			}
			if err := s.remoteStdin.WriteMessage(msgType, msg); err != nil {
				klog.Error(err)
				break
			}
		}
	}()
}

func formatCloseMessage(err error) []byte {
	m := websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, err.Error())
	if e, ok := err.(*websocket.CloseError); ok {
		if e.Code != websocket.CloseNoStatusReceived {
			m = websocket.FormatCloseMessage(e.Code, e.Text)
		}
	}
	return m
}
