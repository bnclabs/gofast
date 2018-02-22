package gofast

import "fmt"
import "runtime/debug"
import "sync/atomic"

func (t *Transport) doTx() {
	defer func() {
		if r := recover(); r != nil {
			errorf("doTx() panic: %v\n", r)
			errorf("\n%s", getStackTrace(2, debug.Stack()))
			go t.Close()
		}
	}()

	batch := make([]*txproto, 0, 64)
	tcpwriteBuf := make([]byte, t.batchsize*t.buffersize)

	drainbuffers := func() {
		atomic.AddUint64(&t.nFlushes, 1)
		var err error
		m, n := 0, 0
		// consolidate.
		for _, arg := range batch {
			if len(arg.packet) > 0 {
				//fmt.Println(hexstring(arg.packet))
				n += copy(tcpwriteBuf[n:], arg.packet)
				atomic.AddUint64(&t.nTx, 1)
			}
		}
		// send.
		if n > 0 {
			//TODO: Issue #2, remove or prevent value escape to heap
			//fmsg := "%v doTx() socket write %v:%v\n"
			//debugf(fmsg, t.logprefix, n, tcpwriteBuf[:n])
			m, err = t.conn.Write(tcpwriteBuf[:n])
			if m != n {
				err = fmt.Errorf("wrote only %d, expected %d", m, n)
			}
		}
		atomic.AddUint64(&t.nTxbyte, uint64(m))
		// unblock the callers.
		for _, arg := range batch {
			arg.n, arg.err = len(arg.packet), err
			if arg.async {
				arg.packet = arg.packet[:cap(arg.packet)]
				t.pTxcmd <- arg
			} else {
				arg.respch <- arg
			}
		}
		//TODO: Issue #2, remove or prevent value escape to heap
		//debugf("%v drained %v packets\n", t.logprefix, len(batch))
		batch = batch[:0] // reset the batch
	}

	infof("%v doTx(batch:%v) started ...\n", t.logprefix, t.batchsize)
loop:
	for {
		select {
		case arg := <-t.txch:
			batch = append(batch, arg)
			if arg.flush || uint64(len(batch)) >= t.batchsize {
				drainbuffers()
			}

		case <-t.killch:
			break loop
		}
	}
	infof("%v doTx() ... stopped\n", t.logprefix)
}
