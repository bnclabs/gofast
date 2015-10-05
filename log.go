//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "io"
import "io/ioutil"
import "log"
import "os"
import "time"
import "strings"

// Logger interface for gofast logging, applications can
// supply a logger object implementing this interface or
// gofast will fall back to the defaul logger.
//
// * default log file is os.Stdout
// * default level is LogLevelInfo
type Logger interface {
	Fatalf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Verbosef(format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Tracef(format string, v ...interface{})
}

type DefaultLogger struct {
	level logLevelInfo
	file  io.Writer
}

type logLevel int

const (
	logLevelIgnore logLevel = iota + 1
	logLevelFatal
	logLevelError
	logLevelWarn
	logLevelInfo
	logLevelVerbose
	logLevelDebug
	logLevelTrace
)

var log Logger // object used by gofast component for logging.

func setLogger(logger Logger, config map[string]interface{}) Logger {
	if logger != nil {
		log = logger
		return
	}

	fd := os.OpenFile(config["log.file"].(string), os.O_RDWR|os.O_APPEND, 0660)
	log = DefaultLogger{
		level: string2logLevel(config["log.level"].(string)),
		file:  fd,
	}
	return log
}

func (l *DefaultLogger) Fatalf(format string, v ...interface{}) {
	l.printf(logLevelFatal, format, v...)
}

func (l *DefaultLogger) Errorf(format string, v ...interface{}) {
	l.printf(logLevelError, format, v...)
}

func (l *DefaultLogger) Warnf(format string, v ...interface{}) {
	l.printf(logLevelWarn, format, v...)
}

func (l *DefaultLogger) Infof(format string, v ...interface{}) {
	l.printf(logLevelInfo, format, v...)
}

func (l *DefaultLogger) Verbosef(format string, v ...interface{}) {
	l.printf(logLevelVerbose, format, v...)
}

func (l *DefaultLogger) Debugf(format string, v ...interface{}) {
	l.printf(logLevelDebug, format, v...)
}

func (l *DefaultLogger) Tracef(format string, v ...interface{}) {
	l.printf(logLevelTrace, format, v...)
}

func (l *DefaultLogger) printf(level logLevel, format string, v ...interface{}) {
	if l.canlog(level) {
		ts := time.Now().Format("2006-01-02T15:04:05.999Z-07:00")
		fmt.Fprintf(l.w, ts+" ["+l.level.String()+"] "+format, v...)
	}
}

func (l *DefaultLogger) canlog(level logLevel) bool {
	if level <= l.level {
		return true
	}
	return false
}

func (l logLevel) String() string {
	switch l {
	case logLevelFatal:
		return "Fatal"
	case logLevelError:
		return "Error"
	case logLevelWarn:
		return "Warn"
	case logLevelInfo:
		return "Info"
	case logLevelVerbose:
		return "Verbose"
	case logLevelDebug:
		return "Debug"
	case logLevelTrace:
		return "Trace"
	}
	panic("unexpected log level") // should never reach here
}

func string2logLevel(s string) logLevel {
	s = strings.ToLower(s)
	switch s {
	case "ignore":
		return logLevelIgnore
	case "fatal":
		return logLevelFatal
	case "error":
		return logLevelError
	case "warn":
		return logLevelWarn
	case "info":
		return logLevelInfo
	case "verbose":
		return logLevelVerbose
	case "debug":
		return logLevelDebug
	case "trace":
		return logLevelTrace
	}
	panic("unexpected log level") // should never reach here
}
