package discovery

// MessageType identifies the discovery message type
type MessageType string

const (
	// MessageTypeAnnounce is sent periodically to announce presence
	MessageTypeAnnounce MessageType = "ANNOUNCE"
	// MessageTypeLeave is sent when gracefully shutting down
	MessageTypeLeave MessageType = "LEAVE"
)

// DiscoveryMessage is the UDP broadcast payload (JSON encoded)
type DiscoveryMessage struct {
	Type      MessageType    `json:"type"`
	Version   uint8          `json:"version"`
	Timestamp int64          `json:"ts"`
	Device    DeviceAnnounce `json:"device"`
}

// DeviceAnnounce contains device information for discovery
type DeviceAnnounce struct {
	DeviceID         string `json:"device_id"`
	DeviceName       string `json:"device_name"`
	GrpcAddr         string `json:"grpc_addr"`
	HttpAddr         string `json:"http_addr"`
	Platform         string `json:"platform"`
	Arch             string `json:"arch"`
	HasCPU           bool   `json:"has_cpu"`
	HasGPU           bool   `json:"has_gpu"`
	HasNPU           bool   `json:"has_npu"`
	CanScreenCapture bool   `json:"can_screen_capture"`
}

// MaxMessageSize is the maximum UDP payload size (stay under MTU)
const MaxMessageSize = 1024
