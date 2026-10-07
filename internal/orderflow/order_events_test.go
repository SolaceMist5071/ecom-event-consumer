package orderflow

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestNextState(t *testing.T) {
	tests := []struct {
		name    string
		current OrderState
		event   PlatformEvent
		want    OrderState
		wantErr bool
	}{
		{"checkout from new", StateNew, PlatformEvent{EventCheckout, "o1"}, StateCheckedOut, false},
		{"fulfillment after checkout", StateCheckedOut, PlatformEvent{EventFulfillment, "o1"}, StateFulfilled, false},
		{"receipt after fulfillment", StateFulfilled, PlatformEvent{EventReceipt, "o1"}, StateReceipted, false},
		{"update after receipt", StateReceipted, PlatformEvent{EventOrderUpdate, "o1"}, StateUpdated, false},
		{"bad checkout", StateCheckedOut, PlatformEvent{EventCheckout, "o1"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextState(tt.current, tt.event)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "topsecret"
	payload := []byte(`{"type":"checkout","order_id":"o1"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	valid := hex.EncodeToString(mac.Sum(nil))
	if !VerifySignature(secret, payload, valid) {
		t.Fatal("expected valid signature to verify")
	}
	if VerifySignature(secret, payload, "deadbeef") {
		t.Fatal("expected invalid signature to fail")
	}
}
