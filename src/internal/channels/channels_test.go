package channels

import (
	"errors"
	"testing"
)

func TestMsgCh_Fields(t *testing.T) {
	sentinel := errors.New("boom")
	msg := MsgCh{
		Err:     sentinel,
		Message: "test message",
		Level:   "INFO",
	}
	if msg.Err != sentinel {
		t.Errorf("expected sentinel error, got %v", msg.Err)
	}
	if msg.Message != "test message" {
		t.Errorf("expected 'test message', got %s", msg.Message)
	}
	if msg.Level != "INFO" {
		t.Errorf("expected 'INFO', got %s", msg.Level)
	}
}

func TestMsgCh_NilErr(t *testing.T) {
	msg := MsgCh{Message: "ok", Level: "WARN"}
	if msg.Err != nil {
		t.Errorf("expected nil Err, got %v", msg.Err)
	}
}

func TestMsgCh_Channel(t *testing.T) {
	ch := make(chan MsgCh, 2)

	ch <- MsgCh{Message: "first", Level: "INFO"}
	ch <- MsgCh{Message: "second", Level: "ERROR"}

	first := <-ch
	if first.Message != "first" {
		t.Errorf("expected 'first', got %s", first.Message)
	}

	second := <-ch
	if second.Level != "ERROR" {
		t.Errorf("expected 'ERROR', got %s", second.Level)
	}
}

func TestMsgCh_SendOnlyChannel(t *testing.T) {
	ch := make(chan MsgCh, 1)

	var sendOnly chan<- MsgCh = ch
	sendOnly <- MsgCh{Message: "send-only", Level: "INFO"}

	received := <-ch
	if received.Message != "send-only" {
		t.Errorf("expected 'send-only', got %s", received.Message)
	}
}
