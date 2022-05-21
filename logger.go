package navi_go_log

import (
	"fmt"
	"os"
	"time"
)

func Debug(v interface{}, args ...interface{}) {
	switch v.(type) {
	case *LogRecord:
		Logger.Log(DEBUG, v.(*LogRecord), DefaultLogCallDepth)
	case string:
		logMsg(DEBUG, v.(string), args...)
	default:
		Logger.SimpleLog(DEBUG, fmt.Sprintf("%v", v), DefaultLogCallDepth)
	}
}

func Debugf(format string, args ...interface{}) {
	Logger.SimpleLog(DEBUG, fmt.Sprintf(format, args...), DefaultLogCallDepth)
}

func Info(v interface{}, args ...interface{}) {
	switch v.(type) {
	case *LogRecord:
		Logger.Log(INFO, v.(*LogRecord), DefaultLogCallDepth)
	case string:
		logMsg(INFO, v.(string), args...)
	default:
		Logger.SimpleLog(INFO, fmt.Sprintf("%v", v), DefaultLogCallDepth)
	}
}

func Infof(format string, args ...interface{}) {
	if args != nil && len(args) > 0 {
		Logger.SimpleLog(INFO, fmt.Sprintf(format, args...), DefaultLogCallDepth)
	} else {
		Logger.SimpleLog(INFO, fmt.Sprintf(format), DefaultLogCallDepth)
	}
}

func Warning(v interface{}, args ...interface{}) {
	switch v.(type) {
	case *LogRecord:
		Logger.Log(WARNING, v.(*LogRecord), DefaultLogCallDepth)
	case string:
		logMsg(WARNING, v.(string), args...)
	default:
		Logger.SimpleLog(WARNING, fmt.Sprintf("%v", v), DefaultLogCallDepth)
	}
}

func Warningf(format string, args ...interface{}) {
	Logger.SimpleLog(WARNING, fmt.Sprintf(format, args...), DefaultLogCallDepth)
}

func Error(v interface{}, args ...interface{}) {
	switch v.(type) {
	case *LogRecord:
		Logger.Log(ERROR, v.(*LogRecord), DefaultLogCallDepth)
	case string:
		logMsg(ERROR, v.(string), args...)
	default:
		Logger.SimpleLog(ERROR, fmt.Sprintf("%v", v), DefaultLogCallDepth)
	}
}

func Errorf(format string, args ...interface{}) {
	Logger.SimpleLog(ERROR, fmt.Sprintf(format, args...), DefaultLogCallDepth)
}

func Critical(v interface{}, args ...interface{}) {
	switch v.(type) {
	case *LogRecord:
		Logger.Log(CRITICAL, v.(*LogRecord), DefaultLogCallDepth)
	case string:
		logMsg(CRITICAL, v.(string), args...)
	default:
		Logger.SimpleLog(CRITICAL, fmt.Sprintf("%v", v), DefaultLogCallDepth)
	}
}

func Criticalf(format string, args ...interface{}) {
	Logger.SimpleLog(CRITICAL, fmt.Sprintf(format, args...), DefaultLogCallDepth)
}

func Fatal(v interface{}, args ...interface{}) {
	switch v.(type) {
	case *LogRecord:
		Logger.Log(FATAL, v.(*LogRecord), DefaultLogCallDepth)
	case string:
		logMsg(FATAL, v.(string), args...)
	default:
		Logger.SimpleLog(FATAL, fmt.Sprintf("%v", v), DefaultLogCallDepth)
	}
	time.Sleep(time.Second * 4)
	os.Exit(1)
}

func Fatalf(format string, args ...interface{}) {
	Logger.SimpleLog(FATAL, fmt.Sprintf(format, args...), DefaultLogCallDepth)
	time.Sleep(time.Second * 4)
	os.Exit(1)
}

func Fixed(v interface{}, args ...interface{}) {
	switch v.(type) {
	case *LogRecord:
		Logger.Log(FIXED, v.(*LogRecord), DefaultLogCallDepth)
	case string:
		logMsg(FIXED, v.(string), args...)
	default:
		Logger.SimpleLog(FIXED, fmt.Sprintf("%v", v), DefaultLogCallDepth)
	}
}

func Fixedf(format string, args ...interface{}) {
	Logger.SimpleLog(FIXED, fmt.Sprintf(format, args...), DefaultLogCallDepth)
}

func logMsg(level int, msg string, args ...interface{}) {
	lr := &LogRecord{
		Message: msg,
	}

	if len(args) == 0 {
		Logger.Log(level, lr, DefaultLogCallDepth+1)
		return
	}
	switch len(args) {
	case 3:
		lr.TraceId = fmt.Sprintf("%v", args[2])
		fallthrough
	case 2:
		lr.Tag = fmt.Sprintf("%v", args[1])
		fallthrough
	case 1:
		lr.ExcInfo = fmt.Sprintf("%v", args[0])
	}
	Logger.Log(level, lr, DefaultLogCallDepth+1)
}
