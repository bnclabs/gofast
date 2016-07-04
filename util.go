//  Copyright (c) 2015 Couchbase, Inc.

package gofast

import "reflect"
import "unsafe"
import "strings"
import "strconv"
import "bytes"
import "fmt"

func bytes2str(bytes []byte) string {
	if bytes == nil {
		return ""
	}
	sl := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	st := &reflect.StringHeader{Data: sl.Data, Len: sl.Len}
	return *(*string)(unsafe.Pointer(st))
}

func str2bytes(str string) []byte {
	if str == "" {
		return nil
	}
	st := (*reflect.StringHeader)(unsafe.Pointer(&str))
	sl := &reflect.SliceHeader{Data: st.Data, Len: st.Len, Cap: st.Len}
	return *(*[]byte)(unsafe.Pointer(sl))
}

func hasString(str string, strs []string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}
	return false
}

func csv2strings(line string, out []string) []string {
	for _, str := range strings.Split(line, ",") {
		if s := strings.Trim(str, " \n\t\r"); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func getStackTrace(skip int, stack []byte) string {
	var buf bytes.Buffer
	lines := strings.Split(string(stack), "\n")
	for _, call := range lines[skip*2:] {
		buf.WriteString(fmt.Sprintf("%s\n", call))
	}
	return buf.String()
}

func hexstring(data []byte) string {
	out := "["
	for _, b := range data {
		out += strconv.FormatInt(int64(b), 16) + ", "
	}
	out += "]"
	return out
}
