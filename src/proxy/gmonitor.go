package proxy

import (
	"github.com/streamrail/concurrent-map"
	log "github.com/cihub/seelog"
	"time"
)

type Monitor interface {
	RecordOne(name string)
	RecordOneAndRt(name string, rtMilliseconds int)
	RecordManyAndRt(name string, count int, rtMilliseconds int)
	Close()
}

var GMonitor = newMonitor()

var done chan bool

func newMonitor() Monitor {
	done = startTicker(func() {
		results := dumpAndClear()
		for k, v := range results {
			item := v.(*MonitorItem)
			log.Infof("monitor item,key=%s,qps=%f,count=%d,rt=%d", k, computeQPS(item), item.TotalCount, item.TotalRt)
		}
	})
	return &MonitorItem{}
}

func computeQPS(item *MonitorItem) float32 {
	if item.TotalRt > 0 {
		return float32(item.TotalCount / item.TotalRt * 1000.0);
	} else {
		return -1.0
	}
}

var monitorItems = cmap.New()

type MonitorItem struct {
	TotalCount int
	TotalRt    int
}

func (monitorItem *MonitorItem) RecordOne(name string) {
	monitorItem.RecordManyAndRt(name, 1, 0)
}

func (monitorItem *MonitorItem) Close() {
	done <- true
}

func (monitorItem *MonitorItem) RecordManyAndRt(name string, count int, rt int) {
	result, ok := monitorItems.Get(name)
	if !ok {
		result = &MonitorItem{}
		monitorItems.Set(name, result)
	}
	item := result.(*MonitorItem)
	item.TotalCount++
	item.TotalRt += rt
}

func (monitorItem *MonitorItem) RecordOneAndRt(name string, rt int) {
	monitorItem.RecordManyAndRt(name, 1, rt)
}

func dumpAndClear() map[string]interface{} {
	result := monitorItems.Items()
	for _, key := range monitorItems.Keys() {
		result, _ := monitorItems.Get(key)
		item := result.(*MonitorItem)
		if item.TotalCount == 0 {
			monitorItems.Remove(key)
		} else {
			monitorItems.Set(key, &MonitorItem{})
		}
	}
	return result
}

func startTicker(f func()) chan bool {
	done := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				f()
			case <-done:
				log.Info("GMonitor log closed")
				return
			}
		}
	}()
	return done
}
