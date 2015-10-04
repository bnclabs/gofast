//  Copyright (c) 2015 Couchbase, Inc.

package gofast

import "testing"
import "fmt"
import "reflect"
import "time"
import "regexp"
import "math/big"

var _ = fmt.Sprintf("dummy")

func TestCborMajor(t *testing.T) {
	if typ := cborMajor(0xff); typ != 0xe0 {
		t.Errorf("fail major() got %v wanted 0xe0", typ)
	}
}

func TestCborNil(t *testing.T) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()

	if n := value2cbor(nil, buf, config); n != 1 {
		t.Errorf("fail value2nil nil: %v want 1", n)
	} else if item, m := cbor2value(buf, config); m != 1 {
		t.Errorf("fail cbor2value on nil len: %v want 1", m)
	} else if item != nil {
		t.Errorf("fail cbor2value on nil: %x", item)
	}
}

func TestCborTrue(t *testing.T) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()

	if n := value2cbor(true, buf, config); n != 1 {
		t.Errorf("fail value2cbor true: %v want 1", n)
	} else if item, m := cbor2value(buf, config); m != 1 {
		t.Errorf("fail cbor2value on true len: %v want 1", m)
	} else if item.(bool) != true {
		t.Errorf("fail cbor2value on true: %v", item)
	}
}

func TestCborFalse(t *testing.T) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()

	if n := value2cbor(false, buf, config); n != 1 {
		t.Errorf("fail value2cbor false: %v want 1", n)
	} else if item, m := cbor2value(buf, config); m != 1 {
		t.Errorf("fail cbor2value on false len: %v want 1", m)
	} else if item.(bool) != false {
		t.Errorf("fail cbor2value on false: %v", item)
	}
}

func TestCborUint8(t *testing.T) {
	config := NewDefaultConfig()
	buf := make([]byte, 20)
	for i := uint16(0); i <= 255; i++ {
		n := valuint82cbor(uint8(i), buf)
		val, m := cbor2value(buf, config)
		if i < 24 && n != 1 {
			t.Errorf("fail code uint8(%v) < 24, got %v", i, n)
		} else if i > 24 && n != 2 {
			t.Errorf("fail code uint8(%v) > 24, got %v", i, n)
		}
		if n != m || val.(uint64) != uint64(i) {
			t.Errorf("fail codec uint8: %v %v %v", n, m, val)
		}
	}
}

func TestCborInt8(t *testing.T) {
	config := NewDefaultConfig()
	buf := make([]byte, 20)
	for i := int16(-128); i <= 127; i++ {
		n := valint82cbor(int8(i), buf)
		val, m := cbor2value(buf, config)
		if -23 <= i && i <= 23 && n != 1 {
			t.Errorf("fail code int8(%v), expected 1 got %v", i, n)
		} else if -23 > i && i > 23 && n != 2 {
			t.Errorf("fail code int8(%v), expected 2 got %v", i, n)
		}
		if num1, ok := val.(int64); ok && (n != m || num1 != int64(i)) {
			t.Errorf("fail codec uint8: %v %v %v", n, m, num1)
		} else if num2, ok := val.(uint64); ok && (n != m || num2 != uint64(i)) {
			t.Errorf("fail codec uint8: %v %v %v", n, m, num2)
		}
	}
}

func TestCborNum(t *testing.T) {
	buf := make([]byte, 20)
	config := NewDefaultConfig()
	tests := [][2]interface{}{
		[2]interface{}{'a', uint64(97)},
		[2]interface{}{byte(0), uint64(0)},
		[2]interface{}{byte(23), uint64(23)},
		[2]interface{}{byte(24), uint64(24)},
		[2]interface{}{byte(255), uint64(255)},
		[2]interface{}{uint8(0), uint64(0)},
		[2]interface{}{uint8(23), uint64(23)},
		[2]interface{}{uint8(24), uint64(24)},
		[2]interface{}{uint8(255), uint64(255)},
		[2]interface{}{int8(-128), int64(-128)},
		[2]interface{}{int8(-24), int64(-24)},
		[2]interface{}{int8(-24), int64(-24)},
		[2]interface{}{int8(-1), int64(-1)},
		[2]interface{}{int8(-0), uint64(-0)},
		[2]interface{}{int8(0), uint64(0)},
		[2]interface{}{int8(23), uint64(23)},
		[2]interface{}{int8(24), uint64(24)},
		[2]interface{}{int8(127), uint64(127)},
		[2]interface{}{uint16(0), uint64(0)},
		[2]interface{}{uint16(23), uint64(23)},
		[2]interface{}{uint16(24), uint64(24)},
		[2]interface{}{uint16(255), uint64(255)},
		[2]interface{}{uint16(65535), uint64(65535)},
		[2]interface{}{int16(-32768), int64(-32768)},
		[2]interface{}{int16(-256), int64(-256)},
		[2]interface{}{int16(-255), int64(-255)},
		[2]interface{}{int16(-129), int64(-129)},
		[2]interface{}{int16(-128), int64(-128)},
		[2]interface{}{int16(-127), int64(-127)},
		[2]interface{}{int16(-24), int64(-24)},
		[2]interface{}{int16(-23), int64(-23)},
		[2]interface{}{int16(-1), int64(-1)},
		[2]interface{}{int16(-0), uint64(0)},
		[2]interface{}{int16(0), uint64(0)},
		[2]interface{}{int16(23), uint64(23)},
		[2]interface{}{int16(24), uint64(24)},
		[2]interface{}{int16(127), uint64(127)},
		[2]interface{}{int16(255), uint64(255)},
		[2]interface{}{int16(32767), uint64(32767)},
		[2]interface{}{uint32(0), uint64(0)},
		[2]interface{}{uint32(23), uint64(23)},
		[2]interface{}{uint32(24), uint64(24)},
		[2]interface{}{uint32(255), uint64(255)},
		[2]interface{}{uint32(65535), uint64(65535)},
		[2]interface{}{uint32(4294967295), uint64(4294967295)},
		[2]interface{}{int32(-2147483648), int64(-2147483648)},
		[2]interface{}{int32(-32769), int64(-32769)},
		[2]interface{}{int32(-32768), int64(-32768)},
		[2]interface{}{int32(-32767), int64(-32767)},
		[2]interface{}{int32(-256), int64(-256)},
		[2]interface{}{int32(-255), int64(-255)},
		[2]interface{}{int32(-129), int64(-129)},
		[2]interface{}{int32(-128), int64(-128)},
		[2]interface{}{int32(-127), int64(-127)},
		[2]interface{}{int32(-24), int64(-24)},
		[2]interface{}{int32(-23), int64(-23)},
		[2]interface{}{int32(-1), int64(-1)},
		[2]interface{}{int32(-0), uint64(-0)},
		[2]interface{}{int32(0), uint64(0)},
		[2]interface{}{int32(23), uint64(23)},
		[2]interface{}{int32(24), uint64(24)},
		[2]interface{}{int32(127), uint64(127)},
		[2]interface{}{int32(32767), uint64(32767)},
		[2]interface{}{int32(65535), uint64(65535)},
		[2]interface{}{int32(2147483647), uint64(2147483647)},
		[2]interface{}{int(-2147483648), int64(-2147483648)},
		[2]interface{}{uint(2147483647), uint64(2147483647)},
		[2]interface{}{uint64(0), uint64(0)},
		[2]interface{}{uint64(23), uint64(23)},
		[2]interface{}{uint64(24), uint64(24)},
		[2]interface{}{uint64(255), uint64(255)},
		[2]interface{}{uint64(65535), uint64(65535)},
		[2]interface{}{uint64(4294967295), uint64(4294967295)},
		[2]interface{}{uint64(18446744073709551615), uint64(18446744073709551615)},
		[2]interface{}{int64(-9223372036854775808), int64(-9223372036854775808)},
		[2]interface{}{int64(-2147483649), int64(-2147483649)},
		[2]interface{}{int64(-2147483648), int64(-2147483648)},
		[2]interface{}{int64(-2147483647), int64(-2147483647)},
		[2]interface{}{int64(-32769), int64(-32769)},
		[2]interface{}{int64(-32768), int64(-32768)},
		[2]interface{}{int64(-32767), int64(-32767)},
		[2]interface{}{int64(-256), int64(-256)},
		[2]interface{}{int64(-255), int64(-255)},
		[2]interface{}{int64(-129), int64(-129)},
		[2]interface{}{int64(-128), int64(-128)},
		[2]interface{}{int64(-127), int64(-127)},
		[2]interface{}{int64(-24), int64(-24)},
		[2]interface{}{int64(-23), int64(-23)},
		[2]interface{}{int64(-1), int64(-1)},
		[2]interface{}{int64(-0), uint64(-0)},
		[2]interface{}{int64(0), uint64(0)},
		[2]interface{}{int64(23), uint64(23)},
		[2]interface{}{int64(24), uint64(24)},
		[2]interface{}{int64(127), uint64(127)},
		[2]interface{}{int64(32767), uint64(32767)},
		[2]interface{}{int64(2147483647), uint64(2147483647)},
		[2]interface{}{int64(4294967295), uint64(4294967295)},
		[2]interface{}{int64(9223372036854775807), uint64(9223372036854775807)},
	}
	for _, test := range tests {
		n := value2cbor(test[0], buf, config)
		val, m := cbor2value(buf, config)
		//t.Logf("executing test case %v", test)
		if n != m || !reflect.DeepEqual(val, test[1]) {
			t.Errorf(
				"fail codec Num: %v %v %T(%v) %T(%v)",
				n, m, val, val, test[1], test[1])
		}
	}
	// test case for number exceeding int64
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic decoding int64 > 9223372036854775807")
			}
		}()
		value2cbor(uint64(9223372036854775808), buf, config)
		buf[0] = (buf[0] & 0x1f) | cborType1 // fix as negative integer
		cbor2value(buf, config)
	}()
}

func TestCborFloat16(t *testing.T) {
	config := NewDefaultConfig()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic while decoding float16")
		}
	}()
	cbor2value([]byte{0xf9, 0, 0, 0, 0}, config)
}

func TestCborFloat32(t *testing.T) {
	buf, ref := make([]byte, 10), float32(10.11)
	config := NewDefaultConfig()
	n := value2cbor(ref, buf, config)
	t.Logf("%v", buf)
	val, m := cbor2value(buf, config)
	if n != 5 || n != m || !reflect.DeepEqual(val, ref) {
		t.Errorf("fail code float32: %v %v %T(%v)", n, m, val, val)
	}
}

func TestCborFloat64(t *testing.T) {
	buf, ref := make([]byte, 10), float64(10.11)
	config := NewDefaultConfig()
	n := value2cbor(ref, buf, config)
	val, m := cbor2value(buf, config)
	if n != 9 || n != m || !reflect.DeepEqual(val, ref) {
		t.Errorf("fail code float32: %v %v %T(%v)", n, m, val, val)
	}
}

func TestCborBytes(t *testing.T) {
	buf, ref := make([]byte, 200), make([]uint8, 100)
	config := NewDefaultConfig()
	for i := 0; i < len(ref); i++ {
		ref[i] = uint8(i)
	}
	n := value2cbor(ref, buf, config)
	val, m := cbor2value(buf, config)
	if n != 102 || n != m || !reflect.DeepEqual(val, ref) {
		t.Errorf("fail code bytes: %v %v %T(%v)", n, m, val, val)
	}
	// test byte-start
	if n := bytesStart(buf); n != 1 {
		t.Errorf("fail code bytes-start len: %v wanted 1", n)
	} else if val, m := cbor2value(buf, config); m != n {
		t.Errorf("fail code bytes-start size: %v wanted %v", m, n)
	} else if !reflect.DeepEqual(val, CborIndefinite(0x5f)) {
		t.Errorf("fail code bytes-start: %v wanted 0x5f", buf[0])
	}
}

func TestCborText(t *testing.T) {
	buf, ref := make([]byte, 200), "hello world"
	config := NewDefaultConfig()
	n := value2cbor(ref, buf, config)
	val, m := cbor2value(buf, config)
	if n != 12 || n != m || !reflect.DeepEqual(val, ref) {
		t.Errorf("fail code text: %v %v %T(%v)", n, m, val, val)
	}

	// test text-start
	if n := textStart(buf); n != 1 {
		t.Errorf("fail code text-start len: %v wanted 1", n)
	} else if val, m := cbor2value(buf, config); m != n {
		t.Errorf("fail code text-start size: %v wanted %v", m, n)
	} else if !reflect.DeepEqual(val, CborIndefinite(0x7f)) {
		t.Errorf("fail code text-start: %x wanted 0x7f", buf[0])
	}
}

func TestCborArray(t *testing.T) {
	buf := make([]byte, 1024)
	ref := []interface{}{10.2, "hello world"}

	// encoding use LengthPrefix
	config := NewDefaultConfig().ContainerEncoding(LengthPrefix)
	n := value2cbor(ref, buf, config)
	val, m := cbor2value(buf, config)
	if n != 22 || n != m || !reflect.DeepEqual(ref, val) {
		t.Errorf("fail code text: %v %v %T(%v)", n, m, val, val)
	}
	// encoding use Stream
	config = NewDefaultConfig()
	n = value2cbor(ref, buf, config)
	val, m = cbor2value(buf, config)
	if n != 23 || n != m || !reflect.DeepEqual(ref, val) {
		t.Errorf("fail code text: %v %v %T(%v)", n, m, val, val)
	}
}

func TestCborMapSlice(t *testing.T) {
	buf := make([]byte, 1024)
	ref := [][2]interface{}{
		[2]interface{}{"10.2", "hello world"},
		[2]interface{}{"hello world", 10.2},
	}
	refm := CborMap2golangMap(ref)
	// encoding use LengthPrefix
	config := NewDefaultConfig().ContainerEncoding(LengthPrefix)
	n := value2cbor(ref, buf, config)
	val, m := cbor2value(buf, config)
	if n != 39 || n != m || !reflect.DeepEqual(refm, val) {
		t.Errorf("fail code text: %v %v %v %v", n, m, refm, val)
	}
	// encoding use Stream
	config = NewDefaultConfig()
	n = value2cbor(ref, buf, config)
	val, m = cbor2value(buf, config)
	if n != 40 || n != m || !reflect.DeepEqual(refm, val) {
		t.Errorf("fail code text: %v %v %v %v", n, m, refm, val)
	}
}

func TestCborMap(t *testing.T) {
	buf := make([]byte, 1024)
	ref := map[string]interface{}{
		"10.2":        "hello world",
		"hello world": float64(10.2),
	}
	// encoding use LengthPrefix
	config := NewDefaultConfig().ContainerEncoding(LengthPrefix)
	n := value2cbor(ref, buf, config)
	val, m := cbor2value(buf, config)
	val = CborMap2golangMap(val)
	if n != 40 || n != m || !reflect.DeepEqual(ref, val) {
		t.Errorf("fail code text: %v %v %v %v", n, m, ref, val)
	}
	// encoding use Stream
	config = NewDefaultConfig()
	n = value2cbor(ref, buf, config)
	val, m = cbor2value(buf, config)
	val = CborMap2golangMap(val)
	if n != 40 || n != m || !reflect.DeepEqual(ref, val) {
		t.Errorf("fail code text: %v %v %v %v", n, m, ref, val)
	}
}

func TestCborBreakStop(t *testing.T) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()
	if n := breakStop(buf); n != 1 {
		t.Errorf("fail code text-start len: %v wanted 1", n)
	} else if val, m := cbor2value(buf, config); m != n {
		t.Errorf("fail code text-start: %v wanted %v", m, n)
	} else if !reflect.DeepEqual(val, CborBreakStop(0xff)) {
		t.Errorf("fail code text-start: %x wanted 0xff", buf[0])
	}
}

func TestCborUndefined(t *testing.T) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()
	if n := valundefined2cbor(buf); n != 1 {
		t.Errorf("fail value2cbor undefined: %v want 1", n)
	} else if item, m := cbor2value(buf, config); m != 1 {
		t.Errorf("fail cbor2value on undefined len: %v want 1", m)
	} else if item.(CborUndefined) != CborUndefined(cborSimpleUndefined) {
		t.Errorf("fail cbor2value on undefined: %T %v", item, item)
	}
}

func TestCborReserved(t *testing.T) {
	config := NewDefaultConfig()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic while decoding reserved")
		}
	}()
	cbor2value([]byte{cborHdr(cborType0, 28)}, config)
}

//---- test cases for tag function

func TestDateTime(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()

	ref, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05+07:00")
	if err != nil {
		t.Errorf("time.Parse() failed: %v", err)
	}

	n := config.ValueToCbor(ref, buf)
	item, m := config.CborToValue(buf[:n])
	if n != 28 || n != m {
		t.Errorf("expected %v got %v %v", 28, n, m)
	}
	if !ref.Equal(item.(time.Time)) {
		t.Errorf("expected %v got %v", ref, item.(time.Time))
	}

	// malformed.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		buf[5] = 'a'
		config.CborToValue(buf[:n])
	}()
}

func TestEpoch(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()

	// positive and negative epoch
	for _, val := range [2]int64{1000000, -100000} {
		ref := CborEpoch(val)
		n := config.ValueToCbor(ref, buf)
		item, m := config.CborToValue(buf[:n])
		if n != 6 || n != m {
			t.Errorf("expected %v got %v %v", 6, n, m)
		}
		if !reflect.DeepEqual(ref, item) {
			t.Errorf("expected %v got %v", ref, item)
		}
	}
	// malformed epoch
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		buf[1] = 0x5a // instead of 0x3a
		config.CborToValue(buf)
	}()
	// malformed epoch
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		buf := make([]byte, 16)
		n := tag2cbor(tagEpoch, buf)
		n += valbytes2cbor([]byte{1, 2}, buf[n:])
		config.CborToValue(buf)
	}()
}

func TestEpochMicro(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	// positive and negative epoch in uS.
	for _, val := range [2]float64{1000000.123456, -100000.123456} {
		ref := CborEpochMicro(val)
		n := config.ValueToCbor(ref, buf)
		item, m := config.CborToValue(buf[:n])
		if n != 10 || n != m {
			t.Errorf("expected %v got %v %v", 10, n, m)
		}
		if !reflect.DeepEqual(ref, item) {
			t.Errorf("expected %v got %v", ref, item)
		}
	}
}

func TestBigNum(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	// positive and negative bignums
	for _, val := range [2]int64{1000, -1000} {
		bigx := big.NewInt(9223372036854775807)
		bigy := big.NewInt(val)
		bigz := big.NewInt(0).Mul(bigx, bigy)
		n := config.ValueToCbor(bigz, buf)
		item, m := config.CborToValue(buf[:n])
		if n != 12 || n != m {
			t.Errorf("expected %v got %v %v", 12, n, m)
		}
		if bigz.Cmp(item.(*big.Int)) != 0 {
			t.Errorf("expected %v got %v", bigz, item.(*big.Int))
		}
	}
}

func TestDecimalFraction(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	// for positive
	refs := []CborDecimalFraction{
		CborDecimalFraction([2]int64{int64(-10), int64(-23)}),
		CborDecimalFraction([2]int64{int64(-10), int64(23)}),
		CborDecimalFraction([2]int64{int64(10), int64(-23)}),
		CborDecimalFraction([2]int64{int64(10), int64(23)}),
	}
	for _, ref := range refs {
		n := config.ValueToCbor(ref, buf)
		item, m := config.CborToValue(buf[:n])
		if n != 3 || n != m {
			t.Errorf("expected %v got %v %v", 3, n, m)
		}
		if !reflect.DeepEqual(ref, item) {
			t.Errorf("expected %v got %v", ref, item)
		}
	}
}

func TestBigFloat(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	refs := []CborBigFloat{
		CborBigFloat([2]interface{}{int64(-10), int64(-23)}),
		CborBigFloat([2]interface{}{int64(-10), int64(23)}),
		CborBigFloat([2]interface{}{int64(10), int64(-23)}),
		CborBigFloat([2]interface{}{int64(10), int64(23)}),
	}
	for _, ref := range refs {
		n := config.ValueToCbor(ref, buf)
		item, m := config.CborToValue(buf[:n])
		if n != 3 || n != m {
			t.Errorf("expected %v got %v %v", 3, n, m)
		}
		if !reflect.DeepEqual(ref, item) {
			t.Errorf("expected %v got %v", ref, item)
		}
	}
}

func TestCbor(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	ref := Cbor([]byte("hello world"))
	n := config.ValueToCbor(ref, buf)
	item, m := config.CborToValue(buf[:n])
	if n != 14 || n != m {
		t.Errorf("expected %v got %v %v", 14, n, m)
	}
	if !reflect.DeepEqual(ref, item) {
		t.Errorf("exptected %v got %v", ref, item)
	}
}

func TestRegexp(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	ref, err := regexp.Compile(`a([0-9]t*)+`)
	if err != nil {
		t.Errorf("compiling regex")
	}
	n := config.ValueToCbor(ref, buf)
	item, m := config.CborToValue(buf[:n])
	if n != 14 || n != m {
		t.Errorf("expected %v got %v %v", 14, n, m)
	}
	if ref.String() != (item.(*regexp.Regexp)).String() {
		t.Errorf("expected %v got %v", ref, item)
	}
	// malformed reg-ex
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic")
			}
		}()
		n := tag2cbor(tagRegexp, buf)
		n += valtext2cbor(`a([0-9]t*+`, buf[n:])
		config.CborToValue(buf)
	}()
}

func TestCborPrefix(t *testing.T) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	ref := CborPrefix([]byte("hello world"))
	n := config.ValueToCbor(ref, buf)
	item, m := config.CborToValue(buf[:n])
	if n != 15 || n != m {
		t.Errorf("expected %v got %v %v", 15, n, m)
	}
	if !reflect.DeepEqual(ref, item) {
		t.Errorf("exptected %v got %v", ref, item)
	}
}

//---- benchmarks

func BenchmarkVal2CborNull(b *testing.B) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()
	for i := 0; i < b.N; i++ {
		value2cbor(nil, buf, config)
	}
}

func BenchmarkCbor2ValNull(b *testing.B) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()
	n := value2cbor(nil, buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborTrue(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(true)
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValTrue(b *testing.B) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()
	n := value2cbor(true, buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborFalse(b *testing.B) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()
	val := interface{}(false)
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValFalse(b *testing.B) {
	buf := make([]byte, 10)
	config := NewDefaultConfig()
	n := value2cbor(false, buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborUint8(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(uint8(255))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValUint8(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(uint8(255), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborInt8(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(int8(-128))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValInt8(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(int8(-128), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborUint16(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(uint16(65535))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValUint16(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(uint16(65535), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborInt16(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(int16(-32768))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValInt16(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(int16(-32768), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborUint32(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(uint32(4294967295))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValUint32(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(uint32(4294967295), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborInt32(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(int32(-2147483648))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValInt32(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(int32(-2147483648), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborUint64(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(uint64(18446744073709551615))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValUint64(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(uint64(18446744073709551615), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborInt64(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(int64(-2147483648))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValInt64(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(int64(-2147483648), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborFlt32(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(float32(10.2))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValFlt32(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(float32(10.2), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborFlt64(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}(float64(10.2))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValFlt64(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor(float64(10.2), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborBytes(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}([]byte("hello world"))
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValBytes(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor([]byte("hello world"), buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborText(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	val := interface{}("hello world")
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValText(b *testing.B) {
	buf := make([]byte, 64)
	config := NewDefaultConfig()
	n := value2cbor("hello world", buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborArr0(b *testing.B) {
	buf, arr := make([]byte, 1024), make([]interface{}, 0)
	config := NewDefaultConfig()
	val := interface{}(arr)
	for i := 0; i < b.N; i++ {
		value2cbor(val, buf, config)
	}
}

func BenchmarkCbor2ValArr0(b *testing.B) {
	buf, arr := make([]byte, 1024), make([]interface{}, 0)
	config := NewDefaultConfig()
	n := value2cbor(arr, buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborArr5(b *testing.B) {
	buf := make([]byte, 1024)
	config := NewDefaultConfig()
	arr := interface{}([]interface{}{5, 5.0, "hello world", true, nil})
	for i := 0; i < b.N; i++ {
		value2cbor(arr, buf, config)
	}
}

func BenchmarkCbor2ValArr5(b *testing.B) {
	buf := make([]byte, 1024)
	config := NewDefaultConfig()
	arr := []interface{}{5, 5.0, "hello world", true, nil}
	n := value2cbor(arr, buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborMap0(b *testing.B) {
	buf := make([]byte, 1024)
	config := NewDefaultConfig()
	m := interface{}(make([][2]interface{}, 0))
	for i := 0; i < b.N; i++ {
		value2cbor(m, buf, config)
	}
}

func BenchmarkCbor2ValMap0(b *testing.B) {
	buf := make([]byte, 1024)
	config := NewDefaultConfig()
	m := make([][2]interface{}, 0)
	n := value2cbor(m, buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func BenchmarkVal2CborMap5(b *testing.B) {
	buf := make([]byte, 1024)
	config := NewDefaultConfig()
	m := interface{}([][2]interface{}{
		[2]interface{}{"key0", 5}, [2]interface{}{"key1", 5.0},
		[2]interface{}{"key2", "hello world"},
		[2]interface{}{"key3", true}, [2]interface{}{"key4", nil},
	})
	for i := 0; i < b.N; i++ {
		value2cbor(m, buf, config)
	}
}

func BenchmarkCbor2ValMap5(b *testing.B) {
	buf := make([]byte, 1024)
	config := NewDefaultConfig()
	m := [][2]interface{}{
		[2]interface{}{"key0", 5}, [2]interface{}{"key1", 5.0},
		[2]interface{}{"key2", "hello world"},
		[2]interface{}{"key3", true}, [2]interface{}{"key4", nil},
	}
	n := value2cbor(m, buf, config)
	for i := 0; i < b.N; i++ {
		cbor2value(buf[:n], config)
	}
}

func fixFloats(val interface{}) interface{} {
	switch v := val.(type) {
	case float64:
		return float32(v)
	case []interface{}:
		for i, x := range v {
			v[i] = fixFloats(x)
		}
		return v
	case map[string]interface{}:
		for p, q := range v {
			v[p] = fixFloats(q)
		}
		return v
	}
	return val
}
