package proxy

import (
	"fmt"
)

type ProtocolError string

func (pe ProtocolError) Error() string {
	return fmt.Sprintf("proxy: %s", string(pe))
}

type movedError struct {
	Slot    int64
	Address string
}

func (me movedError) Error() string {
	return fmt.Sprintf("MOVED %d %s", me.Slot, me.Address)
}

type askError struct {
	Slot    int64
	Address string
}

func (ae askError) Error() string {
	return fmt.Sprintf("ASK %d %s", ae.Slot, ae.Address)
}
