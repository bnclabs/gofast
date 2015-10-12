package gofast

import "testing"
import "fmt"
import "bytes"

var _ = fmt.Sprintf("dummy")

func TestPost(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	ref := []byte{
		217, 217, 247, 88, 38, 217, 1, 0, 88, 33, 216, 37, 191, 24, 39,
		2, 24, 40, 87, 159, 109, 116, 101, 115, 116, 116, 114, 97, 110,
		115, 112, 111, 114, 116, 1, 26, 0, 160, 0, 0, 96, 255, 255}
	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(trans)
	n := trans.post(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func BenchmarkPostPkt(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.post(wai, stream, out)
	}
}
func TestRequest(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	ref := []byte{
		217, 217, 247, 145, 88, 38, 217, 1, 0, 88, 33, 216, 37, 191, 24,
		39, 2, 24, 40, 87, 159, 109, 116, 101, 115, 116, 116, 114, 97,
		110, 115, 112, 111, 114, 116, 1, 26, 0, 160, 0, 0, 96, 255, 255}
	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(trans)
	n := trans.request(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func BenchmarkRequestPkt(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.request(wai, stream, out)
	}
}

func TestResponse(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	ref := []byte{
		217, 217, 247, 145, 88, 38, 217, 1, 0, 88, 33, 216, 37, 191, 24,
		39, 2, 24, 40, 87, 159, 109, 116, 101, 115, 116, 116, 114, 97,
		110, 115, 112, 111, 114, 116, 1, 26, 0, 160, 0, 0, 96, 255, 255}
	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(trans)
	n := trans.response(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func BenchmarkResponsePkt(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.response(wai, stream, out)
	}
}

func TestStart(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	ref := []byte{
		217, 217, 247, 159, 88, 38, 217, 1, 0, 88, 33, 216, 37, 191,
		24, 39, 2, 24, 40, 87, 159, 109, 116, 101, 115, 116, 116, 114, 97,
		110, 115, 112, 111, 114, 116, 1, 26, 0, 160, 0, 0, 96, 255, 255}
	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(trans)
	n := trans.start(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func BenchmarkStartPkt(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.start(wai, stream, out)
	}
}

func TestStream(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	ref := []byte{
		217, 217, 247, 88, 38, 217, 1, 0, 88, 33, 216, 37, 191, 24, 39, 2,
		24, 40, 87, 159, 109, 116, 101, 115, 116, 116, 114, 97, 110, 115,
		112, 111, 114, 116, 1, 26, 0, 160, 0, 0, 96, 255, 255}
	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(trans)
	n := trans.stream(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func BenchmarkStreamPkt(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.stream(wai, stream, out)
	}
}

func TestFinish(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	ref := []byte{217, 217, 247, 68, 217, 1, 0, 255}
	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	n := trans.finish(stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func BenchmarkFinishPkt(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.finish(stream, out)
	}
}

func TestFramePkt(t *testing.T) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		t.Error(err)
	}

	ref := []byte{
		88, 38, 217, 1, 0, 88, 33, 216, 37, 191, 24, 39, 2, 24, 40, 87, 159, 109,
		116, 101, 115, 116, 116, 114, 97, 110, 115, 112, 111, 114, 116, 1, 26, 0,
		160, 0, 0, 96, 255, 255}
	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewWhoami(trans)
	n := trans.framepkt(wai, stream, out)
	if bytes.Compare(out[:n], ref) != 0 {
		t.Errorf("expected %v, got %v", ref, out[:n])
	}
}

func BenchmarkFramePkt(b *testing.B) {
	st, end := tagOpaqueStart, tagOpaqueStart+10
	config := newconfig("testtransport", st, end)
	tconn := newTestConnection(nil, false)
	config["tags"], config["log.level"] = "", "error"
	trans, err := NewTransport(tconn, testVersion(1), nil, config)
	if err != nil {
		b.Error(err)
	}

	stream := trans.getstream(nil)
	out := make([]byte, 1024)
	wai := NewPing("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trans.framepkt(wai, stream, out)
	}
}