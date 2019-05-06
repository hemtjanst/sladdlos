package tradfri

import (
	"sync"
	"time"
)

type DeviceType uint8
type YesNo uint8
type UpdatePriority uint8

const (
	TypeGroup                       = 255
	TypeRemote       DeviceType     = 0
	TypeLight        DeviceType     = 2
	TypePlug         DeviceType     = 3
	TypeMotionSensor DeviceType     = 4
	No               YesNo          = 0
	Yes              YesNo          = 1
	PrioNormal       UpdatePriority = 0
	PrioCritical     UpdatePriority = 1
	PrioRequired     UpdatePriority = 2
	PrioForced       UpdatePriority = 5
)

type Instance interface {
	GetInstanceID() int
}

type BaseType struct {
	sync.RWMutex
	tree       *Tree
	Name       string `json:"9001,omitempty"`
	CreatedAt  int64  `json:"9002,omitempty"`
	InstanceID int    `json:"9003,omitempty"`
}

func (b *BaseType) GetInstanceID() int {
	return b.InstanceID
}

func (b *BaseType) CreateTime() time.Time {
	return time.Unix(b.CreatedAt, 0)
}

func ToYesNo(t bool) YesNo {
	if t {
		return Yes
	}
	return No
}

func (y YesNo) Bool() bool {
	return y != No
}

func (y YesNo) String() string {
	if y == No {
		return "no"
	}
	return "yes"
}

func (u UpdatePriority) String() string {
	switch u {
	case PrioCritical:
		return "critical"
	case PrioForced:
		return "forced"
	case PrioNormal:
		return "normal"
	case PrioRequired:
		return "required"
	default:
		return ""
	}
}
