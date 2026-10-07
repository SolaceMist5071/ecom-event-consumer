package orderflow

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

type EventType string

const (
	EventCheckout    EventType = "checkout"
	EventFulfillment EventType = "fulfillment"
	EventReceipt     EventType = "receipt"
	EventOrderUpdate EventType = "order_update"
)

type OrderState string

const (
	StateNew        OrderState = "new"
	StateCheckedOut OrderState = "checked_out"
	StateFulfilled  OrderState = "fulfilled"
	StateReceipted  OrderState = "receipted"
	StateUpdated    OrderState = "updated"
)

type PlatformEvent struct {
	Type    EventType `json:"type"`
	OrderID string    `json:"order_id"`
}

// NextState returns the expected next state for an order given an event.
// It models the business workflow: checkout -> fulfillment -> receipt -> customer update.
func NextState(current OrderState, e PlatformEvent) (OrderState, error) {
	switch e.Type {
	case EventCheckout:
		if current != StateNew {
			return "", errors.New("checkout only valid from new")
		}
		return StateCheckedOut, nil
	case EventFulfillment:
		if current != StateCheckedOut {
			return "", errors.New("fulfillment only after checkout")
		}
		return StateFulfilled, nil
	case EventReceipt:
		if current != StateFulfilled {
			return "", errors.New("receipt only after fulfillment")
		}
		return StateReceipted, nil
	case EventOrderUpdate:
		if current != StateReceipted {
			return "", errors.New("customer update only after receipt")
		}
		return StateUpdated, nil
	default:
		return "", errors.New("unknown event type")
	}
}

// VerifySignature validates the HMAC-SHA256 signature of the payload.
func VerifySignature(secret string, payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
