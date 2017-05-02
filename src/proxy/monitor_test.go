package proxy

import (
	"testing"
	"time"
)

func Test_Monitor(t *testing.T) {
	GMonitor.RecordOne("a")
	GMonitor.RecordOne("a")
	GMonitor.RecordOne("b")
	GMonitor.RecordOneAndRt("b", 2000)
	cc, _ := monitorItems.MarshalJSON()
	time.Sleep(time.Second * 11)

	time.Sleep(time.Second * 1)
	GMonitor.RecordOneAndRt("b", 2)
	time.Sleep(time.Second * 11)
	GMonitor.Close()
	t.Error(string(cc))
}
