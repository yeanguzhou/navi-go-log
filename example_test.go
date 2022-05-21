package navi_go_log

import (
	"errors"
	"io"
	"io/ioutil"
	_ "net/http/pprof"
	"strconv"
	"testing"
	"time"
)

func TestGetLoggerTmp(t *testing.T) {
	Logger.InitLogger(&LoggerConfig{
		ToStdout:      true,
		ToElastic:     true,
		LogLevel:      "DEBUG",
		LogServerIp:   "192.168.26.100",
		LogServerPort: "514",
		LoggerName:    "log_test",
	})
	Logger.Info(&LogRecord{
		Message: "GetLogger",
		Tag:     "test-tag",
	})

	Logger := GetLogger("test", "")
	Logger.Info(&LogRecord{
		Message: "GetLogger",
	})
}

func TestGetLogger(t *testing.T) {
	logger := GetLogger("test", "")
	logger.InitLogger(&LoggerConfig{
		ToStdout:      true,
		ToElastic:     true,
		LogLevel:      "DEBUG",
		LogServerIp:   "192.168.26.100",
		LogServerPort: "514",
		LoggerName:    "log_test",
	})

	logger.Info(&LogRecord{
		Message: "GetLogger",
		Tag:     "test-tag",
	})

	logger2 := GetLogger("test", "")
	logger2.Info(&LogRecord{
		Message: "GetLogger",
	})

}

func TestInfo(t *testing.T) {
	logger := GetLogger("test", "")
	logger.InitLogger(&LoggerConfig{
		ToStdout:      true,
		ToElastic:     true,
		LogLevel:      "DEBUG",
		LogServerIp:   "192.168.26.100",
		LogServerPort: "514",
		LoggerName:    "log_test",
	})
	t.Parallel()
	// method 1
	Logger.Debug(&LogRecord{
		Message: "message-debug"})
	Logger.Debug(&LogRecord{
		Message: "中文Debug",
		Tag:     "test-debug-2",
		TraceId: "233"})
	// method 2
	Logger.Log(INFO, &LogRecord{
		Message: `"message-log"'`})
	Logger.Log(INFO, &LogRecord{
		Message: "message-log-2",
		Tag:     "test-info-2",
		TraceId: "666"})

	Logger.Info(&LogRecord{
		Message: "中文message-log-3",
		Tag:     "test-info-2",
		TraceId: "666"})
	Logger.Info(&LogRecord{
		Message: "{\"data\":\"中文\"}",
		Tag:     "test-info-2",
		TraceId: "666"})
	var s string = "9223372036854775807"
	i, _ := strconv.ParseInt(s, 10, 64)

	Logger.Info(&LogRecord{
		Message: "message-log-3",
		Tag:     "test-info-2",
		TraceId: "666",
		Extra: &ExtField{
			"extra-test":         "extra",
			"extra-array":        []int{1, 2, 3},
			"extra-float":        1.3,
			"extra-array-string": []string{"a", "b"},
			"extra-int64":        i,
			"extra-MAP": map[string]interface{}{
				"key1": "value",
			},
		}})
	// method 3
	Info(&LogRecord{
		Message: "message-log-4",
		Tag:     "test-info-2",
		TraceId: "666",
		Extra: &ExtField{
			"extra-test": "extra"}})
}

func TestOutputFormat(t *testing.T) {
	logger := GetLogger("test", "")
	logger.InitLogger(&LoggerConfig{
		ToStdout:      true,
		ToElastic:     true,
		LogLevel:      "DEBUG",
		LogServerIp:   "192.168.26.100",
		LogServerPort: "514",
		LoggerName:    "log_test",
	})
	var s string = "9223372036854775807"
	i, _ := strconv.ParseInt(s, 10, 64)

	logger.Critical(&LogRecord{
		Message: `"message'-';:log-3"`,
		Tag:     "test-info-2",
		TraceId: "666",
		ExcInfo: "exc info",
		Extra: &ExtField{
			"log_time":           "2018-1-2 14:03:05.000",
			"exc_info":           "re write exec_info",
			"extra-test":         "extra",
			"extra-array":        []int{1, 2, 3},
			"extra-float":        1.3,
			"extra-array-string": []string{"a", "b"},
			"extra-int64":        i,
			"extra-MAP": map[string]interface{}{
				"key1": "value",
				"key2": 3.14,
			},
		}})
}

func TestSimpleLog(t *testing.T)  {
	Logger.InitLogger(&LoggerConfig{
		ToStdout:        true,
		ToElastic:       true,
		LogLevel:        "DEBUG",
		LogServerIp:     "192.168.1.18",
		LogServerPort:   "514",
		LoggerName:      "tiku_test",
		StdoutFormat:    "custom",
		SimpleLogStatus: true,
	})
	Debug("haha")
	Info("haha")
	Warning("haha")
	Error("haha")
	Critical("haha")
	Fatal("haha")
	Fixed("haha")
	time.Sleep(time.Second *3)
}

func TestError(t *testing.T) {
	logger := GetLogger("data_transfer", "")
	logger.InitLogger(&LoggerConfig{
		ToStdout:      true,
		ToElastic:     true,
		LogLevel:      "DEBUG",
		LogServerIp:   "192.168.26.100",
		LogServerPort: "514",
		LoggerName:    "data_transfer",
	})
	t.Parallel()
	// 自定义配置
	// init log,if you need
	//logger, _ := New("test", "12345")
	defer func() {
		if err := recover(); err != nil {
			logger.Error(&LogRecord{
				Message: "message-debug",
				ExcInfo: err.(error).Error()})
		}
	}()

	// method 1
	logger.Warning(&LogRecord{
		Message: "message-warning",
		Tag:     "test-warning",
		TraceId: "555",
		ExcInfo: errors.New("test warning").Error()})

	logger.Critical(&LogRecord{
		Message: "message-critical",
		Tag:     "test-critical",
		TraceId: "666",
		ExcInfo: errors.New("test critical").Error()})
	// method 2
	logger.Log(ERROR, &LogRecord{
		Message: "message-error-3",
		Tag:     "test-error-3",
		TraceId: "666"})
	// method 3
	logger.Error(&LogRecord{
		Message: "message-error-4",
		Tag:     "test-error-4",
		TraceId: "777"})

	// test panic
	panic(errors.New("test error"))
}

// go test -bench="."
func BenchmarkLogInfo(b *testing.B) {
	Logger.SetWriter([]io.Writer{ioutil.Discard})
	// 必须循环 b.N 次 。 这个数字 b.N 会在运行中调整，以便最终达到合适的时间消耗。方便计算出合理的数据。 （ 免得数据全部是 0 ）
	record := &LogRecord{
		Message: "BenchmarkLogInfo",
		Tag:     "BenchmarkLogInfo",
		TraceId: "666",
		Extra: &ExtField{
			"extra-test": "extra"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Logger.Log(INFO, record)
	}
}

func BenchmarkLogError(b *testing.B) {
	Logger.SetWriter([]io.Writer{ioutil.Discard})
	// 必须循环 b.N 次 。 这个数字 b.N 会在运行中调整，以便最终达到合适的时间消耗。方便计算出合理的数据。 （ 免得数据全部是 0 ）
	var s string = "9223372036854775807"
	j, _ := strconv.ParseInt(s, 10, 64)
	record := &LogRecord{
		Message: "BenchmarkLogError",
		Tag:     "BenchmarkLogError",
		TraceId: "666",
		ExcInfo: "exc info",
		Extra: &ExtField{
			"extra-test":         "extra",
			"extra-array":        []int{1, 2, 3},
			"extra-float":        1.3,
			"extra-array-string": []string{"a", "b"},
			"extra-int64":        j,
			"extra-MAP": map[string]interface{}{
				"key1": "value",
				"key2": 3.14,
			},
		}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Critical(record)
	}
}

// 测试并发效率
func BenchmarkLogParallel(b *testing.B) {
	Logger.SetWriter([]io.Writer{ioutil.Discard})
	record := &LogRecord{
		Message: "BenchmarkLogParallel",
		Tag:     "BenchmarkLogParallel",
		TraceId: "666",
		Extra: &ExtField{
			"extra-test":  "extra",
			"extra-float": 1.3,
			"extra-int":   1,
		}}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// count++
			// if count%100 == 0 {
			// 	InitWorkPool(300)
			// }
			Info(record)
		}
	})

}

// 测试并发效率 2
func BenchmarkLogParallel2(b *testing.B) {
	Logger.SetWriter([]io.Writer{ioutil.Discard})
	record := &LogRecord{
		Message: "BenchmarkLogParallel",
		Tag:     "BenchmarkLogParallel",
		TraceId: "666",
		ExcInfo: "error ",
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {

		for pb.Next() {
			Info(record)
		}
	})

}

func TestPrint(t *testing.T) {

}
