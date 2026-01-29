package realtime

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go"
)

// NATSBridge subscribes to NATS subjects and pushes messages into the Hub.
type NATSBridge struct {
	conn     *nats.Conn
	hub      *Hub
	tenantID string
}

func NewNATSBridge(natsURL, tenantID string, hub *Hub) (*NATSBridge, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &NATSBridge{conn: nc, hub: hub, tenantID: tenantID}, nil
}

// Subscribe listens for progress messages on tenant.<tenantID>.job.*.progress
func (b *NATSBridge) Subscribe() error {
	subject := fmt.Sprintf("tenant.%s.job.*.progress", b.tenantID)
	_, err := b.conn.Subscribe(subject, func(msg *nats.Msg) {
		jobID, err := parseJobIDFromSubject(msg.Subject)
		if err != nil {
			log.Printf("nats: bad subject %q: %v", msg.Subject, err)
			return
		}

		// Wrap the raw progress payload in the outgoing envelope
		envelope := outgoingMsg{
			Type:    "job.progress",
			JobID:   jobID,
			Payload: json.RawMessage(msg.Data),
		}
		data, err := json.Marshal(envelope)
		if err != nil {
			log.Printf("nats: marshal envelope: %v", err)
			return
		}

		b.hub.broadcast <- broadcastMsg{jobID: jobID, payload: data}
	})
	if err != nil {
		return fmt.Errorf("nats subscribe %q: %w", subject, err)
	}

	log.Printf("NATS bridge subscribed to: %s", subject)
	return nil
}

// Close drains the NATS connection.
func (b *NATSBridge) Close() {
	if err := b.conn.Drain(); err != nil {
		log.Printf("nats drain: %v", err)
	}
}

// parseJobIDFromSubject extracts jobID from "tenant.<tid>.job.<jobID>.progress"
func parseJobIDFromSubject(subject string) (uint, error) {
	parts := strings.Split(subject, ".")
	if len(parts) != 5 {
		return 0, fmt.Errorf("expected 5 parts, got %d", len(parts))
	}
	id, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid job id %q: %w", parts[3], err)
	}
	return uint(id), nil
}
