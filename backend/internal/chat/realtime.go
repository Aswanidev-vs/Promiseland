package chat

import "github.com/Aswanidev-vs/promiseland/backend/internal/realtime"

// realtimeHub returns the process-wide hub owned by internal/realtime.
func realtimeHub() *realtime.Hub {
	return realtime.Default()
}

// publish fans a chat event out through the realtime hub. With no subscribers
// the hub is expected to drop the event, so chat handlers never depend on a
// WebSocket client being connected.
func (m *module) publish(eventType string, payload any, channelID, conversationID int64, userIDs []int64) {
	hub := realtimeHub()
	if hub == nil {
		return
	}
	hub.Publish(realtime.Event{
		Type:           eventType,
		Payload:        payload,
		ChannelID:      channelID,
		ConversationID: conversationID,
		UserIDs:        userIDs,
	})
}
