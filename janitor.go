package dbc

import (
	"time"
)

type Janitor struct {
	interval time.Duration
	stop     chan bool
}

func NewJanitor(interval time.Duration) *Janitor {
	var j = &Janitor{}
	j.interval = interval
	j.stop = make(chan bool, 1)
	return j
}

func (this *Janitor) run(t Ticker) {
	if this.interval <= 0 {
		return
	}

	var ticker = time.NewTicker(this.interval)
	for {
		select {
		case <-ticker.C:
			t.Tick()
		case <-this.stop:
			ticker.Stop()
			return
		}
	}
}

func (this *Janitor) close() {
	close(this.stop)
}

type Ticker interface {
	Tick()
}
