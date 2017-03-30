package gofast

import "time"

// FlushPeriod to periodically flush batched packets.
func (t *Transport) FlushPeriod(ms time.Duration) {
	tick := time.Tick(ms)
	go func() {
		for {
			<-tick
			if t.tx([]byte{} /*empty*/, true /*flush*/) != nil {
				return
			}

			//TODO: Issue #2, remove or prevent value escape to heap
			//log.Debugf("%v flushed ... \n", t.logprefix)

			select {
			case <-t.killch:
				return
			default:
			}
		}
	}()
}
