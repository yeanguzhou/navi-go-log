package navi_go_log

import (
	"fmt"
	"sync"
)

var (
	Logger       *CustomLogger // 控制台日志
	GlobalConf   LoggerConfig  // 全局配置
	mySysHandler *SysLogHandle // syslog.writer
)

type LoggerConfig struct {
	ToStdout        bool   // 是否输出到控制台
	StdoutFormat    string // 控制台输出格式（json或custom）
	SimpleLogStatus bool   // 是否开启简易日志
	ToElastic       bool   // 是否输出到syslog服务器
	LogLevel        string // 日志输出等级
	LogServerIp     string // syslog服务器IP
	LogServerPort   string // syslog服务器端口
	LoggerName      string // logger名称，也即服务标签名，如data_transfer
}

var syslogLevM = map[string]Priority{
	"DEBUG":    LOG_DEBUG,
	"INFO":     LOG_INFO,
	"ERROR":    LOG_ERR,
	"WARNING":  LOG_WARNING,
	"CRITICAL": LOG_CRIT,
	"FIXED":    LOG_ALERT,
}

var defaultLevM = map[string]int{
	"DEBUG":    DEBUG,
	"INFO":     INFO,
	"WARNING":  WARNING,
	"ERROR":    ERROR,
	"CRITICAL": CRITICAL,
	"FIXED":    FIXED,
}

//============================
// 初始化
func init() {
	// 默认参数
	GlobalConf.ToStdout = true
	GlobalConf.ToElastic = false
	GlobalConf.LogLevel = "INFO"
	GlobalConf.LoggerName = "log_test"
	GlobalConf.StdoutFormat = "json"
	GlobalConf.SimpleLogStatus = false

	Logger = GetLogger(RootLoggerName, DefaultTag)
	//Logger.SetWriter([]io.Writer{os.Stdout}) // 初始化时默认添加os.Stdout
}

var loggerManager = make(map[string]*CustomLogger, 1)

var lock = sync.RWMutex{}

// GetLogger 获取 Logger
func GetLogger(name string, tag string) *CustomLogger {
	lock.RLock()
	logger, ok := loggerManager[name]
	lock.RUnlock()
	if !ok {
		lock.Lock()
		defer lock.Unlock()
		logger = &CustomLogger{
			Level:     defaultLevM[GlobalConf.LogLevel],
			FixedFlag: true,
			mu:        &sync.Mutex{},
			GlobalTag: GlobalConf.LoggerName,
		}
		logger.Name = name
		if logger.Name != RootLoggerName {
			err := logger.InitLogger(&GlobalConf)
			if err != nil {
				fmt.Println("InitLogger err:", err.Error())
			}
		}

		loggerManager[name] = logger
	}
	if tag != "" {
		logger.SetDefaultTag(tag)
	}
	return logger
}
