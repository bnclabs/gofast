//  Copyright (c) 2015 Couchbase, Inc.

package gofast

import "math"
import "math/big"
import "regexp"
import "time"
import "encoding/binary"
import "fmt"

func value2cbor(item interface{}, out []byte) int {
	n := 0
	switch v := item.(type) {
	case nil:
		n += cborNull(out)
	case bool:
		if v {
			n += cborTrue(out)
		} else {
			n += cborFalse(out)
		}
	case int8:
		n += valint82cbor(v, out)
	case uint8:
		n += valuint82cbor(v, out)
	case int16:
		n += valint162cbor(v, out)
	case uint16:
		n += valuint162cbor(v, out)
	case int32:
		n += valint322cbor(v, out)
	case uint32:
		n += valuint322cbor(v, out)
	case int:
		n += valint642cbor(int64(v), out)
	case int64:
		n += valint642cbor(v, out)
	case uint:
		n += valuint642cbor(uint64(v), out)
	case uint64:
		n += valuint642cbor(v, out)
	case float32:
		n += valfloat322cbor(v, out)
	case float64:
		n += valfloat642cbor(v, out)
	case []byte:
		n += valbytes2cbor(v, out)
	case string:
		n += valtext2cbor(v, out)
	case []interface{}:
		n += valarray2cbor(v, out)
	case [][2]interface{}:
		n += valmap2cbor(v, out)
	case map[string]interface{}:
		var item [2]interface{}
		n += mapStart(out[n:])
		for key, value := range v {
			item[0], item[1] = key, value
			n += mapitem2cbor(item, out[n:])
		}
		n += breakStop(out[n:])
	// simple types
	case CborUndefined:
		n += valundefined2cbor(out)
	// tagged encoding
	case time.Time: // tag-0
		n += valtime2cbor(v, out)
	case CborEpoch: // tag-1
		n += valtime2cbor(v, out)
	case CborEpochMicro: // tag-1
		n += valtime2cbor(v, out)
	case *big.Int: // tag-2 (positive) or tag-3 (negative)
		n += valbignum2cbor(v, out)
	case CborDecimalFraction: // tag-4
		n += valdecimal2cbor(v, out)
	case CborBigFloat: // tag-5
		n += valbigfloat2cbor(v, out)
	case Cbor: // tag-24
		n += valcbor2cbor(v, out)
	case *regexp.Regexp: // tag-35
		n += valregexp2cbor(v, out)
	case CborPrefix: // tag-55799
		n += valcborprefix2cbor(v, out)
	default:
		panic(fmt.Errorf("cbor encode unknownType %T", v))
	}
	return n
}

func cbor2value(buf []byte) (interface{}, int) {
	item, n := cbor2valueM[buf[0]](buf)
	if _, ok := item.(CborIndefinite); ok {
		switch cborMajor(buf[0]) {
		case cborType4:
			arr := make([]interface{}, 0, 2)
			for buf[n] != brkstp {
				item, n1 := cbor2value(buf[n:])
				arr = append(arr, item)
				n += n1
			}
			return arr, n + 1

		case cborType5:
			mapv := make(map[string]interface{})
			for buf[n] != brkstp {
				key, n1 := cbor2value(buf[n:])
				value, n2 := cbor2value(buf[n+n1:])
				mapv[key.(string)] = value
				n = n + n1 + n2
			}
			return mapv, n + 1
		}
	}
	return item, n
}

func tag2cbor(tag uint64, buf []byte) int {
	n := valuint642cbor(tag, buf)
	buf[0] = (buf[0] & 0x1f) | cborType6 // fix the type as tag.
	return n
}

func cbor2tag(buf []byte) (interface{}, int) {
	byt := (buf[0] & 0x1f) | cborType0 // fix as positive num
	item, n := cbor2valueM[byt](buf)
	switch item.(uint64) {
	case tagDateTime:
		item, m := cbor2dtval(buf[n:])
		return item, n + m

	case tagEpoch:
		item, m := cbor2epochval(buf[n:])
		return item, n + m

	case tagPosBignum:
		item, m := cbor2bignumval(buf[n:])
		return item, n + m

	case tagNegBignum:
		item, m := cbor2bignumval(buf[n:])
		return big.NewInt(0).Mul(item.(*big.Int), big.NewInt(-1)), n + m

	case tagDecimalFraction:
		item, m := cbor2decimalval(buf[n:])
		return item, n + m

	case tagBigFloat:
		item, m := cbor2bigfloatval(buf[n:])
		return item, n + m

	case tagCborEnc:
		item, m := cbor2cborval(buf[n:])
		return item, n + m

	case tagRegexp:
		item, m := cbor2regexpval(buf[n:])
		return item, n + m

	case tagCborPrefix:
		item, m := cbor2cborprefixval(buf[n:])
		return item, n + m
	}
	// skip tags
	item, m := cbor2value(buf[n:])
	return item, n + m
}

//---- encode basic data types

func cborNull(buf []byte) int {
	buf[0] = cborHdr(cborType7, cborSimpleTypeNil)
	return 1
}

func cborTrue(buf []byte) int {
	buf[0] = cborHdr(cborType7, cborSimpleTypeTrue)
	return 1
}

func cborFalse(buf []byte) int {
	buf[0] = cborHdr(cborType7, cborSimpleTypeFalse)
	return 1
}

func valuint82cbor(item byte, buf []byte) int {
	if item <= CborMaxSmallInt {
		buf[0] = cborHdr(cborType0, item) // 0..23
		return 1
	}
	buf[0] = cborHdr(cborType0, cborInfo24)
	buf[1] = item // 24..255
	return 2
}

func valint82cbor(item int8, buf []byte) int {
	if item > CborMaxSmallInt {
		buf[0] = cborHdr(cborType0, cborInfo24)
		buf[1] = byte(item) // 24..127
		return 2
	} else if item < -CborMaxSmallInt {
		buf[0] = cborHdr(cborType1, cborInfo24)
		buf[1] = byte(-(item + 1)) // -128..-24
		return 2
	} else if item < 0 {
		buf[0] = cborHdr(cborType1, byte(-(item + 1))) // -23..-1
		return 1
	}
	buf[0] = cborHdr(cborType0, byte(item)) // 0..23
	return 1
}

func valuint162cbor(item uint16, buf []byte) int {
	if item < 256 {
		return valuint82cbor(byte(item), buf)
	}
	buf[0] = cborHdr(cborType0, cborInfo25)
	binary.BigEndian.PutUint16(buf[1:], item) // 256..65535
	return 3
}

func valint162cbor(item int16, buf []byte) int {
	if item > 127 {
		if item < 256 {
			buf[0] = cborHdr(cborType0, cborInfo24)
			buf[1] = byte(item) // 128..255
			return 2
		}
		buf[0] = cborHdr(cborType0, cborInfo25)
		binary.BigEndian.PutUint16(buf[1:], uint16(item)) // 256..32767
		return 3

	} else if item < -128 {
		if item > -256 {
			buf[0] = cborHdr(cborType1, cborInfo24)
			buf[1] = byte(-(item + 1)) // -255..-129
			return 2
		}
		buf[0] = cborHdr(cborType1, cborInfo25) // -32768..-256
		binary.BigEndian.PutUint16(buf[1:], uint16(-(item + 1)))
		return 3
	}
	return valint82cbor(int8(item), buf)
}

func valuint322cbor(item uint32, buf []byte) int {
	if item < 65536 {
		return valuint162cbor(uint16(item), buf) // 0..65535
	}
	buf[0] = cborHdr(cborType0, cborInfo26)
	binary.BigEndian.PutUint32(buf[1:], item) // 65536 to 4294967295
	return 5
}

func valint322cbor(item int32, buf []byte) int {
	if item > 32767 {
		if item < 65536 {
			buf[0] = cborHdr(cborType0, cborInfo25)
			binary.BigEndian.PutUint16(buf[1:], uint16(item)) // 32768..65535
			return 3
		}
		buf[0] = cborHdr(cborType0, cborInfo26) // 65536 to 2147483647
		binary.BigEndian.PutUint32(buf[1:], uint32(item))
		return 5

	} else if item < -32768 {
		if item > -65536 {
			buf[0] = cborHdr(cborType1, cborInfo25) // -65535..-32769
			binary.BigEndian.PutUint16(buf[1:], uint16(-(item + 1)))
			return 3
		}
		buf[0] = cborHdr(cborType1, cborInfo26) // -2147483648..-65536
		binary.BigEndian.PutUint32(buf[1:], uint32(-(item + 1)))
		return 5
	}
	return valint162cbor(int16(item), buf)
}

func valuint642cbor(item uint64, buf []byte) int {
	if item < 4294967296 {
		return valuint322cbor(uint32(item), buf) // 0..4294967295
	}
	// 4294967296 .. 18446744073709551615
	buf[0] = cborHdr(cborType0, cborInfo27)
	binary.BigEndian.PutUint64(buf[1:], item)
	return 9
}

func valint642cbor(item int64, buf []byte) int {
	if item > 2147483647 {
		if item < 4294967296 {
			buf[0] = cborHdr(cborType0, cborInfo26) // 2147483647..4294967296
			binary.BigEndian.PutUint32(buf[1:], uint32(item))
			return 5
		}
		// 4294967296..9223372036854775807
		buf[0] = cborHdr(cborType0, cborInfo27)
		binary.BigEndian.PutUint64(buf[1:], uint64(item))
		return 9

	} else if item < -2147483648 {
		if item > -4294967296 {
			// -4294967295..-2147483649
			buf[0] = cborHdr(cborType1, cborInfo26)
			binary.BigEndian.PutUint32(buf[1:], uint32(-(item + 1)))
			return 5
		}
		// -9223372036854775808..-4294967296
		buf[0] = cborHdr(cborType1, cborInfo27)
		binary.BigEndian.PutUint64(buf[1:], uint64(-(item + 1)))
		return 9
	}
	return valint322cbor(int32(item), buf)
}

// TODO: unused function, cleanup later.
//func length2cbor(item interface{}, buf []byte) int {
//	switch v := item.(type) {
//	case uint8:
//		buf[0] = cborHdr(cborType0, cborInfo24)
//		buf[1] = v
//		return 2
//	case uint16:
//		buf[0] = cborHdr(cborType0, cborInfo25)
//		binary.BigEndian.PutUint16(buf[1:], v)
//		return 3
//	case uint32:
//		buf[0] = cborHdr(cborType0, cborInfo26)
//		binary.BigEndian.PutUint32(buf[1:], v)
//		return 5
//	case uint64:
//		buf[0] = cborHdr(cborType0, cborInfo27)
//		binary.BigEndian.PutUint64(buf[1:], v)
//		return 9
//	}
//	v := item.(int)
//	buf[0] = cborHdr(cborType0, cborInfo27)
//	binary.BigEndian.PutUint64(buf[1:], uint64(v))
//	return 9
//}

func valfloat322cbor(item float32, buf []byte) int {
	buf[0] = cborHdr(cborType7, cborFlt32)
	binary.BigEndian.PutUint32(buf[1:], math.Float32bits(item))
	return 5
}

func valfloat642cbor(item float64, buf []byte) int {
	buf[0] = cborHdr(cborType7, cborFlt64)
	binary.BigEndian.PutUint64(buf[1:], math.Float64bits(item))
	return 9
}

func valbytes2cbor(item []byte, buf []byte) int {
	n := valuint642cbor(uint64(len(item)), buf)
	buf[0] = (buf[0] & 0x1f) | cborType2 // fix the type from type0->type2
	copy(buf[n:], item)
	return n + len(item)
}

func bytesStart(buf []byte) int {
	// indefinite chunks of byte string
	buf[0] = cborHdr(cborType2, byte(cborIndefiniteLength))
	return 1
}

func valtext2cbor(item string, buf []byte) int {
	n := valbytes2cbor(str2bytes(item), buf)
	buf[0] = (buf[0] & 0x1f) | cborType3 // fix the type from type2->type3
	return n
}

func textStart(buf []byte) int {
	// indefinite chunks of text
	buf[0] = cborHdr(cborType3, byte(cborIndefiniteLength))
	return 1
}

func valarray2cbor(items []interface{}, buf []byte) int {
	// Stream encoding
	n := arrayStart(buf)
	n += arrayitems2cbor(items, buf[n:])
	n += breakStop(buf[n:])
	return n
}

func arrayitems2cbor(items []interface{}, buf []byte) int {
	n := 0
	for _, item := range items {
		n += value2cbor(item, buf[n:])
	}
	return n
}

func arrayStart(buf []byte) int {
	// indefinite length array
	buf[0] = cborHdr(cborType4, byte(cborIndefiniteLength))
	return 1
}

func valmap2cbor(items [][2]interface{}, buf []byte) int {
	// Stream encoding
	n := mapStart(buf)
	n += mapl2cbor(items, buf[n:])
	n += breakStop(buf[n:])
	return n
}

func mapl2cbor(items [][2]interface{}, buf []byte) int {
	n := 0
	for _, item := range items {
		n += value2cbor(item[0], buf[n:])
		n += value2cbor(item[1], buf[n:])
	}
	return n
}

func mapitem2cbor(item [2]interface{}, buf []byte) int {
	n := 0
	n += value2cbor(item[0], buf[n:])
	n += value2cbor(item[1], buf[n:])
	return n
}

func mapStart(buf []byte) int {
	// indefinite length map
	buf[0] = cborHdr(cborType5, byte(cborIndefiniteLength))
	return 1
}

func breakStop(buf []byte) int {
	// break stop for indefinite array or map
	buf[0] = cborHdr(cborType7, byte(cborItemBreak))
	return 1
}

func valundefined2cbor(buf []byte) int {
	buf[0] = cborHdr(cborType7, cborSimpleUndefined)
	return 1
}

func simpletypeToCbor(typcode byte, buf []byte) int {
	if typcode < 32 {
		buf[0] = cborHdr(cborType7, typcode)
		return 1
	}
	buf[0] = cborHdr(cborType7, cborSimpleTypeByte)
	buf[1] = typcode
	return 2
}

//---- encode tags

func valtime2cbor(dt interface{}, buf []byte) int {
	n := 0
	switch v := dt.(type) {
	case time.Time: // rfc3339, as refined by section 3.3 rfc4287
		n += tag2cbor(tagDateTime, buf)
		// TODO: make rfc3339 as config.
		n += value2cbor(v.Format(time.RFC3339), buf[n:])
	case CborEpoch:
		n += tag2cbor(tagEpoch, buf)
		n += value2cbor(int64(v), buf[n:])
	case CborEpochMicro:
		n += tag2cbor(tagEpoch, buf)
		n += value2cbor(float64(v), buf[n:])
	}
	return n
}

func valbignum2cbor(num *big.Int, buf []byte) int {
	n := 0
	bytes := num.Bytes()
	if num.Sign() < 0 {
		n += tag2cbor(tagNegBignum, buf)
	} else {
		n += tag2cbor(tagPosBignum, buf)
	}
	n += value2cbor(bytes, buf[n:])
	return n
}

func valdecimal2cbor(item interface{}, buf []byte) int {
	n := tag2cbor(tagDecimalFraction, buf)
	x := item.(CborDecimalFraction)
	n += valint642cbor(x[0], buf[n:])
	n += valint642cbor(x[1], buf[n:])
	return n
}

func valbigfloat2cbor(item interface{}, buf []byte) int {
	n := tag2cbor(tagBigFloat, buf)
	x := item.(CborBigFloat)
	n += valint642cbor(x[0].(int64), buf[n:])
	n += valint642cbor(x[1].(int64), buf[n:])
	return n
}

func valcbor2cbor(item, buf []byte) int {
	n := tag2cbor(tagCborEnc, buf)
	n += valbytes2cbor(item, buf[n:])
	return n
}

func valregexp2cbor(item *regexp.Regexp, buf []byte) int {
	n := tag2cbor(tagRegexp, buf)
	n += valtext2cbor(item.String(), buf[n:])
	return n
}

func valcborprefix2cbor(item, buf []byte) int {
	n := tag2cbor(tagCborPrefix, buf)
	n += valbytes2cbor(item, buf[n:])
	return n
}

//---- decode basic data types

var cbor2valueM = make(map[byte]func([]byte) (interface{}, int))

func cbor2valnull(buf []byte) (interface{}, int) {
	return nil, 1
}

func cbor2valfalse(buf []byte) (interface{}, int) {
	return false, 1
}

func cbor2valtrue(buf []byte) (interface{}, int) {
	return true, 1
}

func cbor2stbyte(buf []byte) (interface{}, int) {
	return buf[1], 2
}

func cbor2valfloat16(buf []byte) (interface{}, int) {
	panic("cbor2valfloat16 not supported")
}

func cbor2valfloat32(buf []byte) (interface{}, int) {
	item, n := binary.BigEndian.Uint32(buf[1:]), 5
	return math.Float32frombits(item), n
}

func cbor2valfloat64(buf []byte) (interface{}, int) {
	item, n := binary.BigEndian.Uint64(buf[1:]), 9
	return math.Float64frombits(item), n
}

func cbor2valt0smallint(buf []byte) (interface{}, int) {
	return uint64(cborInfo(buf[0])), 1
}

func cbor2valt1smallint(buf []byte) (interface{}, int) {
	return -int64(cborInfo(buf[0]) + 1), 1
}

func cbor2valt0info24(buf []byte) (interface{}, int) {
	return uint64(buf[1]), 2
}

func cbor2valt1info24(buf []byte) (interface{}, int) {
	return -int64(buf[1] + 1), 2
}

func cbor2valt0info25(buf []byte) (interface{}, int) {
	return uint64(binary.BigEndian.Uint16(buf[1:])), 3
}

func cbor2valt1info25(buf []byte) (interface{}, int) {
	return -int64(binary.BigEndian.Uint16(buf[1:]) + 1), 3
}

func cbor2valt0info26(buf []byte) (interface{}, int) {
	return uint64(binary.BigEndian.Uint32(buf[1:])), 5
}

func cbor2valt1info26(buf []byte) (interface{}, int) {
	return -int64(binary.BigEndian.Uint32(buf[1:]) + 1), 5
}

func cbor2valt0info27(buf []byte) (interface{}, int) {
	return uint64(binary.BigEndian.Uint64(buf[1:])), 9
}

func cbor2valt1info27(buf []byte) (interface{}, int) {
	x := uint64(binary.BigEndian.Uint64(buf[1:]))
	if x > 9223372036854775807 {
		panic("cbor decoding integer exceeds int64")
	}
	return int64(-x) - 1, 9
}

func cborItemLength(buf []byte) (int, int) {
	if y := cborInfo(buf[0]); y < cborInfo24 {
		return int(y), 1
	} else if y == cborInfo24 {
		return int(buf[1]), 2
	} else if y == cborInfo25 {
		return int(binary.BigEndian.Uint16(buf[1:])), 3
	} else if y == cborInfo26 {
		return int(binary.BigEndian.Uint32(buf[1:])), 5
	}
	return int(binary.BigEndian.Uint64(buf[1:])), 9 // info27
}

func cbor2valt2(buf []byte) (interface{}, int) {
	ln, n := cborItemLength(buf)
	dst := make([]byte, ln)
	copy(dst, buf[n:n+ln])
	return dst, n + ln
}

func cbor2valt2indefinite(buf []byte) (interface{}, int) {
	return CborIndefinite(buf[0]), 1
}

func cbor2valt3(buf []byte) (interface{}, int) {
	ln, n := cborItemLength(buf)
	dst := make([]byte, ln)
	copy(dst, buf[n:n+ln])
	return bytes2str(dst), n + ln
}

func cbor2valt3indefinite(buf []byte) (interface{}, int) {
	return CborIndefinite(buf[0]), 1
}

func cbor2valt4(buf []byte) (interface{}, int) {
	ln, n := cborItemLength(buf)
	arr := make([]interface{}, ln)
	for i := 0; i < ln; i++ {
		item, n1 := cbor2value(buf[n:])
		arr[i], n = item, n+n1
	}
	return arr, n
}

func cbor2valt4indefinite(buf []byte) (interface{}, int) {
	return CborIndefinite(buf[0]), 1
}

func cbor2valt5(buf []byte) (interface{}, int) {
	ln, n := cborItemLength(buf)
	mapv := make(map[string]interface{})
	for i := 0; i < ln; i++ {
		key, n1 := cbor2value(buf[n:])
		value, n2 := cbor2value(buf[n+n1:])
		mapv[key.(string)] = value
		n = n + n1 + n2
	}
	return mapv, n
}

func cbor2valt5indefinite(buf []byte) (interface{}, int) {
	return CborIndefinite(buf[0]), 1
}

func cbor2valbreakcode(buf []byte) (interface{}, int) {
	return CborBreakStop(buf[0]), 1
}

func cbor2valundefined(buf []byte) (interface{}, int) {
	return CborUndefined(cborSimpleUndefined), 1
}

//---- decode tags

func cbor2dtval(buf []byte) (interface{}, int) {
	item, n := cbor2value(buf)
	item, err := time.Parse(time.RFC3339, item.(string))
	if err != nil {
		panic("cbor2dtval(): malformed time.RFC3339")
	}
	return item, n
}

func cbor2epochval(buf []byte) (interface{}, int) {
	item, n := cbor2value(buf)
	switch v := item.(type) {
	case int64:
		return CborEpoch(v), n
	case uint64:
		return CborEpoch(v), n
	case float64:
		return CborEpochMicro(v), n
	}
	fmsg := "cbor2bignumval(): neither int64 nor float64: %T"
	panic(fmt.Errorf(fmsg, item))
}

func cbor2bignumval(buf []byte) (interface{}, int) {
	item, n := cbor2value(buf)
	num := big.NewInt(0).SetBytes(item.([]byte))
	return num, n
}

func cbor2decimalval(buf []byte) (interface{}, int) {
	e, x := cbor2value(buf)
	m, y := cbor2value(buf[x:])
	if a, ok := e.(uint64); ok {
		if b, ok := m.(uint64); ok {
			return CborDecimalFraction([2]int64{int64(a), int64(b)}), x + y
		}
		return CborDecimalFraction([2]int64{int64(a), m.(int64)}), x + y

	} else if b, ok := m.(uint64); ok {
		return CborDecimalFraction([2]int64{e.(int64), int64(b)}), x + y
	}
	return CborDecimalFraction([2]int64{e.(int64), m.(int64)}), x + y
}

func cbor2bigfloatval(buf []byte) (interface{}, int) {
	e, x := cbor2value(buf)
	m, y := cbor2value(buf[x:])
	if a, ok := e.(uint64); ok {
		if b, ok := m.(uint64); ok {
			return CborBigFloat([2]interface{}{int64(a), int64(b)}), x + y
		}
		return CborBigFloat([2]interface{}{int64(a), m.(int64)}), x + y

	} else if b, ok := m.(uint64); ok {
		return CborBigFloat([2]interface{}{e.(int64), int64(b)}), x + y
	}
	return CborBigFloat([2]interface{}{e.(int64), m.(int64)}), x + y
}

func cbor2cborval(buf []byte) (interface{}, int) {
	item, n := cbor2value(buf)
	return Cbor(item.([]uint8)), n
}

func cbor2regexpval(buf []byte) (interface{}, int) {
	item, n := cbor2value(buf)
	s := item.(string)
	re, err := regexp.Compile(s)
	if err != nil {
		panic(fmt.Errorf("compiling regexp %q: %v", s, err))
	}
	return re, n
}

func cbor2cborprefixval(buf []byte) (interface{}, int) {
	item, n := cbor2value(buf)
	return CborPrefix(item.([]byte)), n
}

func init() {
	makePanic := func(msg string) func([]byte) (interface{}, int) {
		return func(_ []byte) (interface{}, int) { panic(msg) }
	}
	//-- type0                  (unsigned integer)
	// 1st-byte 0..23
	for i := byte(0); i < cborInfo24; i++ {
		cbor2valueM[cborHdr(cborType0, i)] = cbor2valt0smallint
	}
	// 1st-byte 24..27
	cbor2valueM[cborHdr(cborType0, cborInfo24)] = cbor2valt0info24
	cbor2valueM[cborHdr(cborType0, cborInfo25)] = cbor2valt0info25
	cbor2valueM[cborHdr(cborType0, cborInfo26)] = cbor2valt0info26
	cbor2valueM[cborHdr(cborType0, cborInfo27)] = cbor2valt0info27
	// 1st-byte 28..31
	msg := "cbor decode value type0 reserved info"
	cbor2valueM[cborHdr(cborType0, 28)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType0, 29)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType0, 30)] = makePanic(msg)
	msg = "cbor decode value type0 indefnite"
	cbor2valueM[cborHdr(cborType0, cborIndefiniteLength)] = makePanic(msg)

	//-- type1                  (signed integer)
	// 1st-byte 0..23
	for i := byte(0); i < cborInfo24; i++ {
		cbor2valueM[cborHdr(cborType1, i)] = cbor2valt1smallint
	}
	// 1st-byte 24..27
	cbor2valueM[cborHdr(cborType1, cborInfo24)] = cbor2valt1info24
	cbor2valueM[cborHdr(cborType1, cborInfo25)] = cbor2valt1info25
	cbor2valueM[cborHdr(cborType1, cborInfo26)] = cbor2valt1info26
	cbor2valueM[cborHdr(cborType1, cborInfo27)] = cbor2valt1info27
	// 1st-byte 28..31
	msg = "cbor decode value type1 reserved info"
	cbor2valueM[cborHdr(cborType1, 28)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType1, 29)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType1, 30)] = makePanic(msg)
	msg = "cbor decode value type1 indefnite"
	cbor2valueM[cborHdr(cborType1, cborIndefiniteLength)] = makePanic(msg)

	//-- type2                  (byte string)
	// 1st-byte 0..27
	for i := 0; i < 28; i++ {
		cbor2valueM[cborHdr(cborType2, byte(i))] = cbor2valt2
	}
	// 1st-byte 28..31
	msg = "cbor decode value type2 reserved info"
	cbor2valueM[cborHdr(cborType2, 28)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType2, 29)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType2, 30)] = makePanic(msg)
	msg = "cbor decode value type2 indefnite"
	cbor2valueM[cborHdr(cborType2, cborIndefiniteLength)] = cbor2valt2indefinite

	//-- type3                  (string)
	// 1st-byte 0..27
	for i := 0; i < 28; i++ {
		cbor2valueM[cborHdr(cborType3, byte(i))] = cbor2valt3
	}
	// 1st-byte 28..31
	msg = "cbor decode value type3 reserved info"
	cbor2valueM[cborHdr(cborType3, 28)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType3, 29)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType3, 30)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType3, cborIndefiniteLength)] = cbor2valt3indefinite

	//-- type4                  (array)
	// 1st-byte 0..27
	for i := 0; i < 28; i++ {
		cbor2valueM[cborHdr(cborType4, byte(i))] = cbor2valt4
	}
	// 1st-byte 28..31
	msg = "cbor decode value type4 reserved info"
	cbor2valueM[cborHdr(cborType4, 28)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType4, 29)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType4, 30)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType4, cborIndefiniteLength)] = cbor2valt4indefinite

	//-- type5                  (map)
	// 1st-byte 0..27
	for i := 0; i < 28; i++ {
		cbor2valueM[cborHdr(cborType5, byte(i))] = cbor2valt5
	}
	// 1st-byte 28..31
	msg = "cbor decode value type5 reserved info"
	cbor2valueM[cborHdr(cborType5, 28)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType5, 29)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType5, 30)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType5, cborIndefiniteLength)] = cbor2valt5indefinite

	//-- type6
	// 1st-byte 0..23
	for i := byte(0); i < cborInfo24; i++ {
		cbor2valueM[cborHdr(cborType6, i)] = cbor2tag
	}
	// 1st-byte 24..27
	cbor2valueM[cborHdr(cborType6, cborInfo24)] = cbor2tag
	cbor2valueM[cborHdr(cborType6, cborInfo25)] = cbor2tag
	cbor2valueM[cborHdr(cborType6, cborInfo26)] = cbor2tag
	cbor2valueM[cborHdr(cborType6, cborInfo27)] = cbor2tag
	// 1st-byte 28..31
	msg = "cbor decode value type6 reserved info"
	cbor2valueM[cborHdr(cborType6, 28)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType6, 29)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType6, 30)] = makePanic(msg)
	msg = "cbor decode value type6 indefnite"
	cbor2valueM[cborHdr(cborType6, cborIndefiniteLength)] = makePanic(msg)

	//-- type7                  (simple types / floats / break-stop)
	// 1st-byte 0..19
	for i := byte(0); i < 20; i++ {
		cbor2valueM[cborHdr(cborType7, i)] =
			func(i byte) func([]byte) (interface{}, int) {
				return func(buf []byte) (interface{}, int) {
					return i, 1
				}
			}(i)
	}
	// 1st-byte 20..23
	cbor2valueM[cborHdr(cborType7, cborSimpleTypeFalse)] = cbor2valfalse
	cbor2valueM[cborHdr(cborType7, cborSimpleTypeTrue)] = cbor2valtrue
	cbor2valueM[cborHdr(cborType7, cborSimpleTypeNil)] = cbor2valnull
	cbor2valueM[cborHdr(cborType7, cborSimpleUndefined)] = cbor2valundefined

	cbor2valueM[cborHdr(cborType7, cborSimpleTypeByte)] = cbor2stbyte
	cbor2valueM[cborHdr(cborType7, cborFlt16)] = cbor2valfloat16
	cbor2valueM[cborHdr(cborType7, cborFlt32)] = cbor2valfloat32
	cbor2valueM[cborHdr(cborType7, cborFlt64)] = cbor2valfloat64
	// 1st-byte 28..31
	msg = "cbor decode value type7 simple type"
	cbor2valueM[cborHdr(cborType7, 28)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType7, 29)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType7, 30)] = makePanic(msg)
	cbor2valueM[cborHdr(cborType7, cborItemBreak)] = cbor2valbreakcode
}
