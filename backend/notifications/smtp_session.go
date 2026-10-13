package notifications

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"
)

var smtpOperationTimeout = 30 * time.Second

type smtpSession struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   net.Conn
	client *smtp.Client
	stop   func() bool
	ehlo   *smtpEHLOTracker
}

// smtpEHLOTracker records whether the server accepted EHLO. net/smtp's
// Client.Hello falls back to HELO when EHLO is rejected, matching the behavior
// of the pinned SendMail implementation. Extension("AUTH") alone cannot
// distinguish that legacy HELO path from successful ESMTP without AUTH.
type smtpEHLOTracker struct {
	net.Conn
	mu             sync.Mutex
	commandBuffer  []byte
	responseBuffer []byte
	awaitingEHLO   bool
	ehloAccepted   bool
}

func (c *smtpEHLOTracker) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		c.mu.Lock()
		c.commandBuffer = append(c.commandBuffer, p[:n]...)
		for {
			lineEnd := bytes.IndexByte(c.commandBuffer, '\n')
			if lineEnd < 0 {
				break
			}
			line := strings.TrimSpace(string(c.commandBuffer[:lineEnd]))
			c.commandBuffer = c.commandBuffer[lineEnd+1:]
			if strings.HasPrefix(strings.ToUpper(line), "EHLO ") {
				c.awaitingEHLO = true
				c.ehloAccepted = false
				c.responseBuffer = nil
			}
		}
		c.mu.Unlock()
	}
	return n, err
}

func (c *smtpEHLOTracker) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.mu.Lock()
		if c.awaitingEHLO {
			c.responseBuffer = append(c.responseBuffer, p[:n]...)
			for {
				lineEnd := bytes.IndexByte(c.responseBuffer, '\n')
				if lineEnd < 0 {
					break
				}
				line := strings.TrimRight(string(c.responseBuffer[:lineEnd]), "\r")
				c.responseBuffer = c.responseBuffer[lineEnd+1:]
				if len(line) < 4 || line[0] < '0' || line[0] > '9' || line[1] < '0' || line[1] > '9' || line[2] < '0' || line[2] > '9' {
					continue
				}
				if line[3] == '-' {
					continue
				}
				c.ehloAccepted = line[:3] == "250"
				c.awaitingEHLO = false
				c.responseBuffer = nil
				break
			}
		}
		c.mu.Unlock()
	}
	return n, err
}

func (c *smtpEHLOTracker) accepted() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ehloAccepted
}

func openSMTPSession(parent context.Context, host string, port int) (*smtpSession, error) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, smtpOperationTimeout)
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		cancel()
		return nil, failedDelivery("connect")
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			stop()
			_ = conn.Close()
			cancel()
			return nil, failedDelivery("deadline")
		}
	}
	tracker := &smtpEHLOTracker{Conn: conn}
	client, err := smtp.NewClient(tracker, host)
	if err != nil {
		stop()
		_ = conn.Close()
		cancel()
		return nil, failedDelivery(smtpStage("greeting", err))
	}
	return &smtpSession{ctx: ctx, cancel: cancel, conn: conn, client: client, stop: stop, ehlo: tracker}, nil
}

func (s *smtpSession) close() {
	if s == nil {
		return
	}
	if s.stop != nil {
		s.stop()
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.client != nil {
		_ = s.client.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
}

func smtpStage(stage string, err error) string {
	var response *textproto.Error
	if errors.As(err, &response) && response.Code >= 100 && response.Code <= 599 {
		return fmt.Sprintf("%s_%d", stage, response.Code)
	}
	return stage
}

func smtpAcceptedDataError(err error) error {
	var response *textproto.Error
	if errors.As(err, &response) {
		return failedDelivery(smtpStage("data_reply", err))
	}
	return unknownDelivery("data_reply")
}

func smtpStartTLS(client *smtp.Client, host string) error {
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return failedDelivery(smtpStage("starttls", err))
		}
	}
	return nil
}
