package realtime

import "log"

// Hub manages WebSocket clients and routes messages by jobID.
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// jobID -> set of subscribed clients
	subscriptions map[uint]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	subscribe  chan subscribeMsg
	broadcast  chan broadcastMsg
}

type subscribeMsg struct {
	client *Client
	jobID  uint
}

type broadcastMsg struct {
	jobID   uint
	payload []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		subscriptions: make(map[uint]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		subscribe:     make(chan subscribeMsg),
		broadcast:     make(chan broadcastMsg, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("client registered (total: %d)", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove from all subscriptions
				for jobID, subs := range h.subscriptions {
					delete(subs, client)
					if len(subs) == 0 {
						delete(h.subscriptions, jobID)
					}
				}
				log.Printf("client unregistered (total: %d)", len(h.clients))
			}

		case msg := <-h.subscribe:
			if _, ok := h.subscriptions[msg.jobID]; !ok {
				h.subscriptions[msg.jobID] = make(map[*Client]bool)
			}
			h.subscriptions[msg.jobID][msg.client] = true
			log.Printf("client subscribed to job %d (subscribers: %d)", msg.jobID, len(h.subscriptions[msg.jobID]))

		case msg := <-h.broadcast:
			if subs, ok := h.subscriptions[msg.jobID]; ok {
				for client := range subs {
					select {
					case client.send <- msg.payload:
					default:
						// Client buffer full, remove it
						close(client.send)
						delete(subs, client)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}
