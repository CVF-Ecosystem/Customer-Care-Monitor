package notifications

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

type r091StalledSMTP struct {
	listener net.Listener
	accepted chan struct{}
	conn     net.Conn
	done     chan struct{}
}

func r091StartStalledSMTP(t *testing.T) *r091StalledSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on synthetic stalled SMTP: %v", err)
	}
	s := &r091StalledSMTP{listener: ln, accepted: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(s.done)
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		s.conn = conn
		close(s.accepted)
		_, _ = io.Copy(io.Discard, conn) // no greeting: exact blocked client socket is closed by test cleanup
		_ = conn.Close()
	}()
	t.Cleanup(func() { s.closeAndJoin(t) })
	return s
}

func (s *r091StalledSMTP) closeAndJoin(t *testing.T) {
	t.Helper()
	_ = s.listener.Close()
	if s.conn != nil {
		_ = s.conn.Close()
	}
	select {
	case <-s.done:
	case <-time.After(2 * time.Second):
		t.Errorf("synthetic stalled SMTP goroutine did not join")
	}
}

func TestR091OldSMTPContextCancelsStalledGreeting(t *testing.T) {
	for _, operation := range []string{"send", "health"} {
		t.Run(operation, func(t *testing.T) {
			s := r091StartStalledSMTP(t)
			port := s.listener.Addr().(*net.TCPAddr).Port
			n := NewEmailNotifier("127.0.0.1", port, "", "", "sender@example.invalid", []string{"recipient@example.invalid"})
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
			defer cancel()
			returned := make(chan error, 1)
			go func() {
				if operation == "send" {
					returned <- n.Send(ctx, "synthetic", "body")
					return
				}
				returned <- n.HealthCheck(ctx)
			}()
			select {
			case <-s.accepted:
			case <-time.After(time.Second):
				t.Fatal("synthetic SMTP did not accept the local test connection")
			}
			behavioralReturn := false
			select {
			case err := <-returned:
				behavioralReturn = true
				if err == nil {
					t.Errorf("deadline cancellation returned success")
				}
			case <-time.After(350 * time.Millisecond):
				// A missed bounded return is the old-source behavioral failure. Close/join below
				// before reporting it, so no blocked test goroutine is abandoned.
			}
			s.closeAndJoin(t)
			if !behavioralReturn {
				<-returned
				t.Errorf("behavioral detector: %s remained blocked beyond its 120ms caller deadline", operation)
			}
		})
	}
}
