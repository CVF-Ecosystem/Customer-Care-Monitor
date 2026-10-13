package notifications

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func r091AssertDeliveryStatus(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("delivery returned nil; want %s", want)
	}
	var typed *deliveryError
	if !errors.As(err, &typed) || typed.status != want {
		t.Fatalf("delivery error=%T %v, want typed %s", err, err, want)
	}
}

func TestR091SMTPFinalDataReplyDeterminesAcceptance(t *testing.T) {
	oldTimeout := smtpOperationTimeout
	smtpOperationTimeout = 2 * time.Second
	t.Cleanup(func() { smtpOperationTimeout = oldTimeout })

	t.Run("accepted even when QUIT disconnects", func(t *testing.T) {
		server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: "250 accepted", closeConnectionQuit: true})
		notifier := NewEmailNotifier("127.0.0.1", server.port(), "", "", "sender@example.invalid", []string{"accepted@example.invalid"})
		if err := notifier.Send(context.Background(), "synthetic", "synthetic body"); err != nil {
			t.Fatalf("final DATA 250 was accepted; later QUIT disconnect must not reverse it: %v", err)
		}
		select {
		case <-server.accepted:
		case <-time.After(time.Second):
			t.Fatal("synthetic server did not observe accepted DATA")
		}
	})

	t.Run("explicit rejection is retryable failure", func(t *testing.T) {
		server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: "550 synthetic rejected"})
		notifier := NewEmailNotifier("127.0.0.1", server.port(), "", "", "sender@example.invalid", []string{"rejected@example.invalid"})
		r091AssertDeliveryStatus(t, notifier.Send(context.Background(), "synthetic", "synthetic body"), deliveryFailedStatus)
	})

	t.Run("close after DATA terminator is ambiguous", func(t *testing.T) {
		server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{defaultDataReply: ""})
		notifier := NewEmailNotifier("127.0.0.1", server.port(), "", "", "sender@example.invalid", []string{"unknown@example.invalid"})
		r091AssertDeliveryStatus(t, notifier.Send(context.Background(), "synthetic", "synthetic body"), deliveryUnknown)
	})
}

func TestR091SMTPDefaultDeadlineBoundsBackgroundContext(t *testing.T) {
	oldTimeout := smtpOperationTimeout
	smtpOperationTimeout = 120 * time.Millisecond
	t.Cleanup(func() { smtpOperationTimeout = oldTimeout })
	server := r091StartStalledSMTP(t)
	notifier := NewEmailNotifier("127.0.0.1", server.listener.Addr().(*net.TCPAddr).Port, "", "", "sender@example.invalid", []string{"deadline@example.invalid"})
	returned := make(chan error, 1)
	started := time.Now()
	go func() { returned <- notifier.Send(context.Background(), "synthetic", "body") }()
	select {
	case <-server.accepted:
	case <-time.After(time.Second):
		t.Fatal("synthetic SMTP server did not accept local connection")
	}
	select {
	case err := <-returned:
		if err == nil {
			t.Fatal("background context unexpectedly returned success")
		}
		if time.Since(started) > 700*time.Millisecond {
			t.Fatalf("default SMTP test deadline returned too slowly: %s", time.Since(started))
		}
	case <-time.After(400 * time.Millisecond):
		// Close the exact owned loopback socket and join before recording a detector failure.
		server.closeAndJoin(t)
		<-returned
		t.Fatal("background-context SMTP operation ignored the bounded default deadline")
	}
}

func TestR091R1SMTPHELOFallbackPreservesLegacyDelivery(t *testing.T) {
	server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{rejectEHLO: true, defaultDataReply: "250 synthetic accepted"})
	notifier := NewEmailNotifier("127.0.0.1", server.port(), "synthetic-user", "synthetic-password", "sender@example.invalid", []string{"helo-only@example.invalid"})
	if err := notifier.Send(context.Background(), "synthetic", "synthetic body"); err != nil {
		t.Fatalf("legacy HELO-only relay was rejected even though SMTP DATA acceptance was available: %v", err)
	}
	select {
	case <-server.accepted:
	case <-time.After(time.Second):
		t.Fatal("HELO-only relay did not observe accepted DATA")
	}
}

func TestR091R1SMTPESMTPWithoutAUTHStillRejects(t *testing.T) {
	server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{noAuth: true, defaultDataReply: "250 synthetic accepted"})
	notifier := NewEmailNotifier("127.0.0.1", server.port(), "synthetic-user", "synthetic-password", "sender@example.invalid", []string{"no-auth@example.invalid"})
	r091AssertDeliveryStatus(t, notifier.Send(context.Background(), "synthetic", "synthetic body"), deliveryFailedStatus)
	select {
	case recipient := <-server.dataReady:
		t.Fatalf("ESMTP without AUTH unexpectedly reached DATA for %s", recipient)
	default:
	}
}

func TestR091R1SMTPContextCancellationClosesStalledGreetingAndFinalDATA(t *testing.T) {
	t.Run("stalled EHLO command", func(t *testing.T) {
		gate := r091NewSMTPGate()
		server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{stallCommand: "EHLO", holdCommand: gate})
		notifier := NewEmailNotifier("127.0.0.1", server.port(), "", "", "sender@example.invalid", []string{"stalled-command@example.invalid"})
		ctx, cancel := context.WithCancel(context.Background())
		returned := make(chan error, 1)
		go func() { returned <- notifier.Send(ctx, "synthetic", "synthetic body") }()
		select {
		case <-server.commandSeen:
		case <-time.After(time.Second):
			server.closeAndJoin(t)
			cancel()
			<-returned
			t.Fatal("synthetic SMTP server did not receive the stalled EHLO command")
		}
		cancel()
		select {
		case <-returned:
		case <-time.After(500 * time.Millisecond):
			server.closeAndJoin(t)
			<-returned
			t.Fatal("SMTP Send did not return promptly after the EHLO context was canceled")
		}
		select {
		case <-server.connectionClosed:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("SMTP server did not observe client socket closure before cleanup")
		}
	})

	t.Run("stalled final DATA reply", func(t *testing.T) {
		gate := r091NewSMTPGate()
		server := r091StartDeliverySMTP(t, r091DeliverySMTPPlan{holdAfterData: gate, defaultDataReply: "250 synthetic accepted"})
		notifier := NewEmailNotifier("127.0.0.1", server.port(), "", "", "sender@example.invalid", []string{"stalled-data@example.invalid"})
		ctx, cancel := context.WithCancel(context.Background())
		returned := make(chan error, 1)
		go func() { returned <- notifier.Send(ctx, "synthetic", "synthetic body") }()
		select {
		case <-server.dataReady:
		case <-time.After(time.Second):
			server.closeAndJoin(t)
			cancel()
			<-returned
			t.Fatal("synthetic SMTP server did not receive the DATA terminator")
		}
		cancel()
		select {
		case err := <-returned:
			var typed *deliveryError
			if !errors.As(err, &typed) || typed.status != deliveryUnknown {
				t.Fatalf("canceled final DATA response=%T %v, want UNKNOWN", err, err)
			}
		case <-time.After(500 * time.Millisecond):
			server.closeAndJoin(t)
			<-returned
			t.Fatal("SMTP Send did not return promptly after final DATA context cancellation")
		}
		select {
		case <-server.connectionClosed:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("SMTP server did not observe client socket closure before cleanup")
		}
	})
}
