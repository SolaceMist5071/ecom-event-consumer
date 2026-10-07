package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"ecom-event-consumer/internal/infrai"
	"ecom-event-consumer/internal/orderflow"
)

var orderStates = map[string]orderflow.OrderState{}

func main() {
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		secret = "demo-secret"
	}
	endpoint := "https://my.host/ecom"
	// Register webhook and queue using the SAME key and base URL.
	_, err := infrai.WebhooksRegister(endpoint, []string{"*"}, secret)
	if err != nil {
		log.Printf("webhook register note: %v", err)
	}
	_, err = infrai.QueuePushSubscribe("ecom-events", endpoint)
	if err != nil {
		log.Printf("queue subscribe note: %v", err)
	}

	http.HandleFunc("/ecom", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sig := r.Header.Get("X-Infrai-Signature")
		if !orderflow.VerifySignature(secret, body, sig) {
			http.Error(w, "bad signature", http.StatusBadRequest)
			return
		}
		var ev orderflow.PlatformEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		cur := orderStates[ev.OrderID]
		next, err := orderflow.NextState(cur, ev)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		orderStates[ev.OrderID] = next
		log.Printf("order %s -> %s", ev.OrderID, next)
		w.WriteHeader(http.StatusOK)
	})
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
