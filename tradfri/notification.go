package tradfri

type NotificationEvent int

const (
	EventNewFirmwareAvailable NotificationEvent = 1001
	EventGatewayReboot        NotificationEvent = 1003
	EventInternetUnreachable  NotificationEvent = 5001
)

type Notification struct {
	BaseType
	Event   NotificationEvent `json:"9015,omitempty"`
	Details []string          `json:"9017,omitempty"`
	State   int               `json:"9014"`
}

func (n *Notification) EventString() string {
	switch n.Event {
	case EventGatewayReboot:
		return "Gateway rebooting"
	case EventInternetUnreachable:
		return "Internet unreachable"
	case EventNewFirmwareAvailable:
		return "New firmware available"
	default:
		return "Unknown event"
	}
}
