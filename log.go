//  Copyright (c) 2014 Couchbase, Inc.

package gofast

import "io"
import "os"
import "fmt"
import "time"
import "strings"

// Logger interface for gofast logging, applications can
// supply a logger object implementing this interface or
// gofast will fall back to the defaultLogger{}.
type Logger interface {
	Fatalf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Verbosef(format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Tracef(format string, v ...interface{})
}

// defaultLogger with default log-file as os.Stdout and,
// default log-level as LogLevelInfo. Applications can
// supply a Logger{} object when instantiating the
// Transport.
type defaultLogger struct {
	level  logLevel
	output io.Writer
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

func init() {
	level, logfd := logLevelInfo, os.Stdout
	log = &defaultLogger{level: level, output: logfd}
}

// SetLogger to integrate gofast logging with application logging.
func SetLogger(logger Logger, setts map[string]interface{}) Logger {
	if logger != nil {
		log = logger
		return log
	}

	var err error
	level := logLevelInfo
	vallevel, ok1 := setts["log.level"]
	if ok1 {
		level = string2logLevel(vallevel.(string))
	}
	logfd := os.Stdout
	valfile, ok2 := setts["log.file"]
	if ok2 {
		logfile := valfile.(string)
		logfd, err = os.OpenFile(logfile, os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			if logfd, err = os.Create(logfile); err != nil {
				panic(fmt.Errorf("unable to create log file %v: %v", err))
			}
		}
	}
	if ok1 || ok2 {
		log = &defaultLogger{level: level, output: logfd}
	}
	return log
}

func (l *defaultLogger) Fatalf(format string, v ...interface{}) {
	l.printf(logLevelFatal, format, v...)
}

func (l *defaultLogger) Errorf(format string, v ...interface{}) {
	l.printf(logLevelError, format, v...)
}

func (l *defaultLogger) Warnf(format string, v ...interface{}) {
	l.printf(logLevelWarn, format, v...)
}

func (l *defaultLogger) Infof(format string, v ...interface{}) {
	l.printf(logLevelInfo, format, v...)
}

func (l *defaultLogger) Verbosef(format string, v ...interface{}) {
	l.printf(logLevelVerbose, format, v...)
}

func (l *defaultLogger) Debugf(format string, v ...interface{}) {
	l.printf(logLevelDebug, format, v...)
}

func (l *defaultLogger) Tracef(format string, v ...interface{}) {
	l.printf(logLevelTrace, format, v...)
}

func (l *defaultLogger) printf(level logLevel, format string, v ...interface{}) {
	if l.canlog(level) {
		ts := time.Now().Format("2006-01-02T15:04:05.999Z-07:00")
		fmt.Fprintf(l.output, ts+" ["+level.String()+"] "+format, v...)
	}
}

func (l *defaultLogger) canlog(level logLevel) bool {
	if level <= l.level {
		return true
	}
	return false
}

func (l logLevel) String() string {
	switch l {
	case logLevelIgnore:
		return "Ignor"
	case logLevelFatal:
		return "Fatal"
	case logLevelError:
		return "Error"
	case logLevelWarn:
		return "Warng"
	case logLevelInfo:
		return "Infom"
	case logLevelVerbose:
		return "Verbs"
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
