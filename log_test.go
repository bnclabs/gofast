package gofast

import "testing"
import "os"
import "strings"
import "fmt"
import "io/ioutil"

var _ = fmt.Sprintf("dummy")

func TestSetLogger(t *testing.T) {
	logfile := "setlogger_test.log.file"
	logline := "hello world"
	defer os.Remove(logfile)

	ref := &DefaultLogger{level: logLevelIgnore, output: nil}
	log := setLogger(ref, nil).(*DefaultLogger)
	if log.level != logLevelIgnore || log.output != nil {
		t.Errorf("expected %v, got %v", ref, log)
	}

	// test a custom logger
	config := map[string]interface{}{
		"log.level": "info",
		"log.file":  logfile,
	}
	clog := setLogger(nil, config)
	clog.Infof(logline)
	clog.Verbosef(logline)
	if data, err := ioutil.ReadFile(logfile); err != nil {
		t.Error(err)
	} else if s := string(data); !strings.Contains(s, "hello world") {
		t.Errorf("expected %v, got %v", logline, s)
	} else if len(strings.Split(s, "\n")) != 1 {
		t.Errorf("expected %v, got %v", logline, s)
	}
}
