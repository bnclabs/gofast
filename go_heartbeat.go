package gofast

import "time"

// SendHeartbeat periodically to remote peer, this can help in detecting
// inactive, or half-open connections.
func (t *Transport) SendHeartbeat(ms time.Duration) {
	if ms == 0 {
		return
	}

	count, tick := uint64(0), time.Tick(ms)
	go func() {
		for {
			<-tick
			msg := newHeartbeat(count)
			if t.Post(msg, true /*flush*/) != nil {
				return
			}
			count++

			//TODO: Issue #2, remove or prevent value escape to heap
			//log.Debugf("%v posted heartbeat %v\n", t.logprefix, count)

			select {
			case <-t.killch:
				return
			default:
			}
		}
	}()
}
