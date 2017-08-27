package tradfri

type Gateway struct {
	observable
	tree *Tree

	// NTPServer that the gateway uses
	NTPServer string `json:"9023"`

	// Version of trådfri gateway
	Version string `json:"9029"`

	// UpdateState is set to 1 if the gateway is currently
	// updating its firmware?
	UpdateState int `json:"9054"`

	// UpdateProgress seems to be a percentage
	UpdateProgress int `json:"9055"`

	// UpdateURL links to details regarding the update
	UpdateURL string `json:"9056"`

	// Current time of the gateway in unix timestamp format
	Timestamp int64 `json:"9059"`

	// TimestampUtc is the current time of the gateway in the format 2017-08-20T23:13:42.006000Z
	TimestampUtc string `json:"9060"`

	// CommissioningMode ?
	CommissioningMode int `json:"9061"`

	// UpdatePriority ?
	UpdatePriority UpdatePriority `json:"9066"`

	// UpdateAcceptedTimestamp
	UpdateAcceptedTimestamp int64 `json:"9069"`

	// TimeSource ?
	TimeSource int `json:"9071"`

	// ForceCheckOTAUpdate ?
	//
	ForceCheckOTAUpdate string `json:"9032"`

	// Name
	Name string `json:"9035"`

	// Field* are unknown fields exposed by the gateway.
	// Do not depend on the names staying the same
	Field9060 int    `json:"9060"`
	Field9062 int    `json:"9062"`
	Field9072 int    `json:"9072"`
	Field9073 int    `json:"9073"`
	Field9074 int    `json:"9074"`
	Field9075 int    `json:"9075"`
	Field9076 int    `json:"9076"`
	Field9077 int    `json:"9077"`
	Field9078 int    `json:"9078"`
	Field9079 int    `json:"9079"`
	Field9080 int    `json:"9080"`
	Field9081 string `json:"9081"`
}
