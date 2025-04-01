// pkg/subscription/client.go
package subscription

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents a generic WebSocket message for GraphQL subscriptions.
type WSMessage struct {
	Type    string          `json:"type"`
	Id      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// SubscribeToQuery attempts to establish a subscription using both "subscribe" and "start" message types.
// It returns the open WebSocket connection if one of the attempts is successful.
func SubscribeToQuery(wsURL string, query string) (*websocket.Conn, error) {
	msgTypes := []string{"subscribe", "start"}
	var lastErr error

	for _, msgType := range msgTypes {
		// Connect to the WebSocket endpoint.
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to connect: %w", err)
			continue
		}

		// Set a read deadline.
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		// Send connection_init message.
		initMsg := WSMessage{
			Type:    "connection_init",
			Payload: json.RawMessage(`{}`),
		}
		if err := conn.WriteJSON(initMsg); err != nil {
			conn.Close()
			lastErr = fmt.Errorf("failed to send connection_init: %w", err)
			continue
		}

		// Wait for connection_ack.
		_, ackMsg, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			lastErr = fmt.Errorf("failed to read connection_ack: %w", err)
			continue
		}
		log.Printf("Received ack using msgType %q: %s", msgType, string(ackMsg))

		// Prepare the subscription payload.
		subPayload := map[string]interface{}{
			"query": query,
		}
		payloadBytes, err := json.Marshal(subPayload)
		if err != nil {
			conn.Close()
			lastErr = fmt.Errorf("failed to marshal subscription payload: %w", err)
			continue
		}

		// Send the subscription message using the current msgType.
		subMsg := WSMessage{
			Type:    msgType,
			Id:      "1", // Use a unique ID if managing multiple subscriptions.
			Payload: payloadBytes,
		}
		if err := conn.WriteJSON(subMsg); err != nil {
			conn.Close()
			lastErr = fmt.Errorf("failed to send subscription message with type %q: %w", msgType, err)
			continue
		}

		// If we've reached this point, the subscription message was sent successfully.
		log.Printf("Subscription message sent successfully using msgType %q", msgType)
		return conn, nil
	}
	return nil, fmt.Errorf("failed to send subscription message using both 'subscribe' and 'start': %w", lastErr)
}

// Listen continuously reads messages from the WebSocket connection and processes them.
func Listen(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		log.Printf("Received message: %s", message)
	}
}
