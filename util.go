//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import _ "fmt"

// failsafeOp can be used by gen-server implementors to avoid infinitely
// blocked API calls.
func failsafeOp(
	muxch, respch chan []interface{}, cmd []interface{},
	finch chan bool) ([]interface{}, error) {

	select {
	case muxch <- cmd:
		if respch != nil {
			select {
			case resp := <-respch:
				return resp, nil
			case <-finch:
				return nil, ErrorClosed
			}
		}
	case <-finch:
		return nil, ErrorClosed
	}
	return nil, nil
}

// failsafeOpAsync is same as FailsafeOp that can be used for
// asynchronous operation, that is, caller does not wait for response.
func failsafeOpAsync(
	muxch chan []interface{}, cmd []interface{}, finch chan bool) error {

	select {
	case muxch <- cmd:
	case <-finch:
		return ErrorClosed
	}
	return nil
}

// opError suppliments FailsafeOp used by gen-servers.
func opError(err error, vals []interface{}, idx int) error {
	if err != nil {
		return err
	} else if vals[idx] == nil {
		return nil
	} else if err, ok := vals[idx].(error); ok {
		return err
	}
	return nil
}
