package navi_go_log

import (
	"fmt"
	"os"

	"encoding/json"
	"errors"
	"io"
	"strconv"
	"sync"
	"time"
	// json "github.com/json-iterator/go"
)

//==================syslog==========================
var syslogWriter SysLogHandle

//==================================================
/*
FIXED 等级用于记录固定的信息，比如程序启动，关闭等等。
CRITICAL 级别以上会输出栈信息
*/
const (
	DEBUG    = 10
	INFO     = 20
	WARNING  = 30
	ERROR    = 40
	CRITICAL = 50
	FATAL    = 60
	FIXED    = 100

	openBrace   = '{'
	closeBrace  = '}'
	comma       = ','
	doubleQuote = '"'
	singleQuote = '\''
	colon       = ':'
	newLine     = '\n'
)
const RFC3339 = "2006-01-02T15:04:05.999999+08:00"
const RootLoggerName = "root_logger"
const DefaultTag = "root"

var GlobleStdLock = &sync.Mutex{}

var NameToLevel = map[string]int{
	"DEBUG":    DEBUG,
	"INFO":     INFO,
	"WARNING":  WARNING,
	"ERROR":    ERROR,
	"CRITICAL": CRITICAL,
	"FATAL":    FATAL,
	"FIXED":    FIXED,
}

var LevelToName = map[int]string{
	DEBUG:    "DEBUG",
	INFO:     "INFO",
	WARNING:  "WARNING",
	ERROR:    "ERROR",
	CRITICAL: "CRITICAL",
	FATAL:    "FATAL",
	FIXED:    "FIXED",
}

var CustomLevelToName = map[int]string{
	DEBUG:    "D",
	INFO:     "I",
	WARNING:  "W",
	ERROR:    "E",
	CRITICAL: "C",
	FATAL:    "F",
	FIXED:    "F",
}

// 前景 背景 颜色
// ---------------------------------------
// 30  40  黑色
// 31  41  红色
// 32  42  绿色
// 33  43  黄色
// 34  44  蓝色
// 35  45  紫红色
// 36  46  青蓝色
// 37  47  白色
//
// 代码 意义
// -------------------------
//  0  终端默认设置
//  1  高亮显示
//  4  使用下划线
//  5  闪烁
//  7  反白显示
//  8  不可见

var LevelBackgroundColor = map[int]int{
	DEBUG:    40,
	INFO:     40,
	WARNING:  40,
	ERROR:    40,
	CRITICAL: 40,
	FATAL:    40,
	FIXED:    40,
}

var LevelFrontColor = map[int]int{
	DEBUG:    37,
	INFO:     36,
	WARNING:  33,
	ERROR:    31,
	CRITICAL: 31,
	FATAL:    35,
	FIXED:    37,
}

var NoMatchLogLevel = errors.New("can't match log level")

// log 主体
type CustomLogger struct {
	Name            string
	Level           int
	FixedFlag       bool // 是否启用 FIXED 日志输出,默认是true
	mu              *sync.Mutex
	out             io.Writer
	customStdout    io.Writer
	Tag             []byte
	CloserWriter    *SysLogHandle
	GlobalTag       string
	StdoutFormat    string
	SimpleLogStatus bool
}

// 日志输出的字段，true表示可以在拓展字段中覆盖他
var recordField = map[string]bool{
	"level":      false,
	"log_time":   true,
	"filename":   false,
	"moudle":     false,
	"line_no":    false,
	"func_name":  false,
	"message":    false,
	"tag":        false,
	"trace_id":   false,
	"exc_info":   false,
	"stack_info": false,
}

// 3 allocs/op
type ExtField map[string]interface{}

// 允许设置的log内容
type LogRecord struct {
	Message string    `json:"message,omitempty"`
	Tag     string    `json:"tag"`
	TraceId string    `json:"trace_id,omitempty"`
	ExcInfo string    `json:"exc_info,omitempty,string"`
	Extra   *ExtField `json:"extra,omitempty"`
}

// 是否直接调用log的标志,用来设置stack_skip层数的,自定义以便识别.
type LogCallDepth int

var DefaultLogCallDepth LogCallDepth = 3

func SetLogCallDepth(depth int) {
	DefaultLogCallDepth = LogCallDepth(depth)
}

// LogLevel 设置日志等级
func LogLevel(level int) error {
	_, ok := LevelToName[level]
	if !ok {
		return NoMatchLogLevel
	}
	if Logger.Level > 0 {
		Logger.Level = level
	}
	return nil
}

// GetTime 获取时间（毫秒级）
func GetTime() string {
	now := time.Now()
	// %Y-%m-%d %H:%M:%S
	// 2016 -> Y 年
	// 01   -> m 月
	// 02   -> d 日
	// 15   -> H 小时
	// 04   -> M 分钟
	// 05   -> S 秒
	// .000 ->   毫秒
	return now.Format("2006-01-02 15:04:05.000")
}

// IsEnableLog 是否允许打印日志
func (logger *CustomLogger) isEnableLog(level int) bool {
	//logRecordNew := setFuncInfo(&logRecord,2)
	return (level >= logger.Level) && (logger.FixedFlag || level < FIXED)
	//return level >= logger.Level
}

// SetWriter 设置日志记录容器，默认是os.stdout
func (logger *CustomLogger) SetWriter(writer []io.Writer) {
	// 允许配置多个writer
	if writer != nil {
		logger.out = io.MultiWriter(writer...)
	}
}

func (logger *CustomLogger) SetCustomWriter(writer io.Writer) {
	// 允许配置多个writer
	if writer != nil {
		logger.customStdout = writer
	}
}

func (logger *CustomLogger) SetStdoutFormat(stdoutFormat string) {
	if stdoutFormat == "custom" {
		logger.StdoutFormat = "custom"
	} else {
		logger.StdoutFormat = "json"
	}
}

func (logger *CustomLogger) SetSimpleLogStatus(status bool) {
	logger.SimpleLogStatus = status
}

// InitLogger 设置日志输出到标志输出
func (logger *CustomLogger) InitLogger(loggerConfig *LoggerConfig) (err error) {
	// loggerName is global_tag
	// 环境变量优先级最高
	envToStdout := os.Getenv("LOG_TO_STDOUT")
	envToSyslog := os.Getenv("LOG_TO_SYSLOG")
	envIP := os.Getenv("LOG_SERVER_IP")
	envPort := os.Getenv("LOG_SERVER_PORT")
	envLevel := os.Getenv("LOG_OUT_LEVEL")
	envStdoutFormat := os.Getenv("STDOUT_FORMAT")
	envSimpleLogOn := os.Getenv("SIMPLE_LOG_ON")

	// 检查环境变量
	if envToStdout == "YES" {
		loggerConfig.ToStdout = true
	} else if envToStdout == "NO" {
		loggerConfig.ToStdout = false
	}

	if envToSyslog == "YES" {
		loggerConfig.ToElastic = true
	} else if envToSyslog == "NO" {
		loggerConfig.ToElastic = false
	}

	if envIP != "" {
		loggerConfig.LogServerIp = envIP
	}

	if envPort != "" {
		loggerConfig.LogServerPort = envPort
	}

	if envLevel != "" {
		loggerConfig.LogLevel = envLevel
	}

	if envStdoutFormat != "" {
		loggerConfig.StdoutFormat = envStdoutFormat
	}

	if envSimpleLogOn == "YES" {
		loggerConfig.SimpleLogStatus = true
	} else if envSimpleLogOn == "NO" {
		loggerConfig.SimpleLogStatus = false
	}

	if logger.Name == RootLoggerName {
		GlobalConf = *loggerConfig
	}

	if loggerConfig.LoggerName == "" && GlobalConf.LoggerName != "" {
		loggerConfig.LoggerName = GlobalConf.LoggerName
	}
	logger.GlobalTag = loggerConfig.LoggerName

	var writers []io.Writer
	var oldSyslog *SysLogHandle
	if loggerConfig.ToElastic {
		mySysHandler, err = Dial("tcp", loggerConfig.LogServerIp+":"+loggerConfig.LogServerPort, syslogLevM[loggerConfig.LogLevel])
		if err != nil {
			panic(err)
		}
		if logger.CloserWriter != nil {
			oldSyslog = logger.CloserWriter
		}
		logger.CloserWriter = mySysHandler
		writers = append(writers, mySysHandler)
	}
	if loggerConfig.ToStdout {
		// writers = append(writers, GetLockWriter(os.Stdout, GlobleStdLock))
		logger.SetStdoutFormat(loggerConfig.StdoutFormat)
		logger.SetSimpleLogStatus(loggerConfig.SimpleLogStatus)
		if logger.StdoutFormat == "custom" {
			logger.SetCustomWriter(os.Stdout)
		} else {
			writers = append(writers, os.Stdout)
		}
	}
	logger.SetLevel(defaultLevM[loggerConfig.LogLevel])
	logger.SetWriter(writers)
	if oldSyslog != nil {
		oldSyslog.Close()
	}
	return nil
}

// SetLevel 设置日志输出等级
func (logger *CustomLogger) SetLevel(Level int) {
	if _, ok := LevelToName[Level]; ok {
		logger.Level = Level
	} else {
		logger.Level = INFO
	}
}

// SetDefaultTag 设置默认TAG
func (logger *CustomLogger) SetDefaultTag(tag string) {
	if tag != "" {
		logger.Tag = EncodeString(tag, false)
	} else {
		logger.Tag = nil
	}
}

// SetGlobalTag 设置 Global Tag
func (logger *CustomLogger) SetGlobalTag(globalTag string) {
	logger.GlobalTag = globalTag
}

// SetFixedFlag 是否启用 FIXED 日志输出,默认是true
func (logger *CustomLogger) SetFixedFlag(flag bool) {
	logger.FixedFlag = flag
}

// WriterClose  关闭Writer
func (logger *CustomLogger) WriterClose() {
	if logger.CloserWriter != nil {
		logger.CloserWriter.Close()
	}
}

func (logger *CustomLogger) SimpleLog(level int, msg string, extend ...interface{}) {
	if !logger.isEnableLog(level) {
		return
	}
	if !logger.SimpleLogStatus {
		return
	}

	stackSkip := DefaultLogCallDepth
	//stackSkip := LogCallDepth(4)
	// 判断是否是直接调用log，非直接调用log的，需要设置一下skip参数，用于栈信息的获取
	for _, v := range extend {
		switch v.(type) {
		case LogCallDepth:
			stackSkip = v.(LogCallDepth)
		}
	}
	// using map
	//设置函数调用信息
	var filename string
	var lineNo int

	// 设置错误栈信息,level 为 FIXED 时，也不记录
	// 300000	      4680 ns/op	    1200 B/op	       9 allocs/op
	if level >= CRITICAL && level != FIXED {
		// 3600 ns/op 10 allocs/op
		// 1000000	      2583 ns/op	     208 B/op	       1 allocs/op
		_, filename, _, _, lineNo = CallersWithFirstCallInfo(int(stackSkip) - 1)
	} else {
		filename, _, _, lineNo = setFuncInfo(int(stackSkip))
	}
	data := GetBytesBuffer()
	simpleLogTime := fmt.Sprintf("%s", GetTime())
	simpleLevel := fmt.Sprintf("%c[%d;%d;%dm%s%s%s%c[0m", 0x1B, 1, LevelBackgroundColor[level],
		LevelFrontColor[level], "[", CustomLevelToName[level], "]", 0x1B)
	simpleMessage := fmt.Sprintf("%c[%d;%d;%dm%s%c[0m", 0x1B, 1, 0,
		LevelFrontColor[level], msg, 0x1B)
	filenameAndLineNo := fmt.Sprintf("%c[%d;%d;%dm%s%s:%d%s%c[0m", 0x1B, 1, 0,
		LevelFrontColor[level], "[", filename, lineNo,"]",  0x1B)

	simpleLog := fmt.Sprintf("%s %s %s ▶ %s\n",
		simpleLogTime,
		simpleLevel,
		filenameAndLineNo,
		simpleMessage,
	)

	data.WriteString(simpleLog)
	go func() {
		logger.customStdoutWrite(data.Bytes())
		PutBytesBuffer(data)
	}()
}

// Log 日志记录，手动写入bytes,效率更快，有待完整测试
func (logger *CustomLogger) Log(level int, logRecord *LogRecord, extend ...interface{}) {
	if !logger.isEnableLog(level) {
		return
	}
	// 输出数据缓冲区
	// lgr := GetLogRecodeHandle()

	// // 1 allocs/op
	//stackSkip := LogCallDepth(4)
	stackSkip := DefaultLogCallDepth
	// 判断是否是直接调用log，非直接调用log的，需要设置一下skip参数，用于栈信息的获取
	for _, v := range extend {
		switch v.(type) {
		case LogCallDepth:
			stackSkip = v.(LogCallDepth)
		}
	}
	// using map
	//设置函数调用信息
	var filename, module, funcName, stackInfo string
	var lineNo int

	// 设置错误栈信息,level 为 FIXED 时，也不记录
	// 300000	      4680 ns/op	    1200 B/op	       9 allocs/op
	if level >= CRITICAL && level != FIXED {
		// 3600 ns/op 10 allocs/op
		// 1000000	      2583 ns/op	     208 B/op	       1 allocs/op
		stackInfo, filename, module, funcName, lineNo = CallersWithFirstCallInfo(int(stackSkip) - 1)
	} else {
		filename, module, funcName, lineNo = setFuncInfo(int(stackSkip))
	}

	// 控制台日志定制化输出
	if logger.StdoutFormat == "custom"{
		customData := GetBytesBuffer()
		customLogTime := fmt.Sprintf("%s", GetTime())
		customLevel := fmt.Sprintf("%c[%d;%d;%dm%s%s%s%c[0m", 0x1B, 1, LevelBackgroundColor[level],
			LevelFrontColor[level], "[", CustomLevelToName[level], "]", 0x1B)
		//customMessage := fmt.Sprintf("%v", logRecord.Message)
		customMessage := fmt.Sprintf("%c[%d;%d;%dm%s%c[0m", 0x1B, 1, 0,
			LevelFrontColor[level], logRecord.Message, 0x1B)
		filenameAndLineNo := fmt.Sprintf("%c[%d;%d;%dm%s%s:%d%s%c[0m", 0x1B, 1, 0,
			LevelFrontColor[level], "[", filename, lineNo,"]", 0x1B)

		if stackInfo != "" {
			stackInfo = stackInfo + "\n"
		}

		customLog := fmt.Sprintf("%s %s %s ▶ %s %s\n%s",
			customLogTime,
			customLevel,
			filenameAndLineNo,
			customMessage,
			logRecord.ExcInfo,
			stackInfo)

		customData.WriteString(customLog)
		go func() {
			logger.customStdoutWrite(customData.Bytes())
			PutBytesBuffer(customData)
		}()
		// json格式输出的write（无论是stdout还是rsyslog）如果为空，则不应该再往下走
		if logger.out == nil{
			return
		}
	}

	// json格式输出
	data := GetBytesBuffer()
	// defer PutBytesBuffer(data)

	data.WriteByte('{')
	// 设置global_tag
	data.WriteByte('"')
	data.WriteString("@global_tag")
	data.WriteString(`":"`)

	data.WriteString(logger.GlobalTag)
	data.WriteByte('"')

	// 设置log级别 level_name
	data.WriteByte(',')
	data.WriteByte('"')
	data.WriteString("level_name")
	data.WriteString(`":"`)
	data.WriteString(LevelToName[level])
	data.WriteByte('"')

	// 日志记录时间 log_time
	data.WriteByte(',')
	data.WriteByte('"')
	data.WriteString("log_time")
	data.WriteString(`":`)
	//logTime := []byte(fmt.Sprintf("\"%s\"", GetTime()))
	logTime := []byte(fmt.Sprintf("\"%s\"", time.Now().Format(RFC3339)))
	// 日志时间允许被重写
	if logRecord.Extra != nil {
		if v, ok := (*logRecord.Extra)["log_time"]; ok {
			logTime = EncodeString(v.(string), false)
		}
	}
	data.Write(logTime)

	// filename 文件名
	data.WriteByte(',')
	data.WriteByte('"')
	data.WriteString("filename")
	data.WriteString(`":"`)
	data.WriteString(filename)
	data.WriteByte('"')

	// 包名 module
	data.WriteByte(',')
	data.WriteByte('"')
	data.WriteString("module")
	data.WriteString(`":"`)
	data.WriteString(module)
	data.WriteByte('"')

	// 函数名 funcName
	data.WriteByte(',')
	data.WriteByte('"')
	data.WriteString("func_name")
	data.WriteString(`":"`)
	data.WriteString(funcName)
	data.WriteByte('"')

	// 行号 lineNo
	data.WriteByte(',')
	data.WriteByte('"')
	data.WriteString("line_no")
	data.WriteString(`":`)
	data.WriteString(strconv.Itoa(lineNo))

	// 日志信息 message
	data.WriteByte(',')
	data.WriteByte('"')
	data.WriteString("message")
	data.WriteString(`":`)
	data.Write(EncodeString(logRecord.Message, false))

	// 栈信息 stackInfo
	if stackInfo != "" {
		data.WriteByte(',')
		data.WriteByte('"')
		data.WriteString("stack_info")
		data.WriteString(`":`)
		// 300000	      4236 ns/op	    1288 B/op	      10 allocs/op
		data.Write(EncodeString(stackInfo, false))
	}
	// 错误信息 exc_info
	if logRecord.ExcInfo != "" {
		data.WriteByte(',')
		data.WriteByte('"')
		data.WriteString("exc_info")
		data.WriteString(`":`)
		data.Write(EncodeString(logRecord.ExcInfo, false))
	}

	// 写入trace_id
	if logRecord.TraceId != "" {
		data.WriteByte(',')
		data.WriteByte('"')
		data.WriteString("trace_id")
		data.WriteString(`":"`)
		data.WriteString(logRecord.TraceId)
		data.WriteByte('"')
	}
	// 写入tag
	if logRecord.Tag != "" || logger.Tag != nil {
		data.WriteByte(',')
		data.WriteByte('"')
		data.WriteString("tag")
		data.WriteString(`":`)
		if logRecord.Tag != "" {
			data.Write(EncodeString(logRecord.Tag, false))
		} else if logger.Tag != nil {
			data.Write(logger.Tag)
		}
	}

	// 添加拓展字段的信息
	if logRecord.Extra != nil {
		for k, v := range *logRecord.Extra {
			if _, ok := recordField[k]; ok {
				continue
			}
			data.WriteByte(',')
			switch v.(type) {
			case string:
				data.Write(EncodeString(k, false))
				data.WriteString(`:`)
				data.Write(EncodeString(v.(string), false))
			default:
				data.Write(EncodeString(k, false))
				data.WriteString(`:`)
				tmp, _ := json.Marshal(v)
				data.Write(tmp)
			}
		}
	}

	data.WriteByte('}')
	data.WriteByte('\n')

	go func() {
		logger.write(data.Bytes())
		PutBytesBuffer(data)
	}()

}

func (logger *CustomLogger) write(data []byte) {
	// logger.mu.Lock()
	logger.out.Write(data)
	// logger.mu.Unlock()
}

func (logger *CustomLogger) customStdoutWrite(data []byte) {
	// logger.mu.Lock()
	logger.customStdout.Write(data)
	// logger.mu.Unlock()
}

func (logger *CustomLogger) Debug(logRecord *LogRecord) {
	logger.Log(DEBUG, logRecord, DefaultLogCallDepth)
}

func (logger *CustomLogger) Info(logRecord *LogRecord) {
	logger.Log(INFO, logRecord, DefaultLogCallDepth)
}

func (logger *CustomLogger) Warning(logRecord *LogRecord) {
	logger.Log(WARNING, logRecord, DefaultLogCallDepth)
}

func (logger *CustomLogger) Error(logRecord *LogRecord) {
	logger.Log(ERROR, logRecord, DefaultLogCallDepth)
}

func (logger *CustomLogger) Critical(logRecord *LogRecord) {
	logger.Log(CRITICAL, logRecord, DefaultLogCallDepth)
}

func (logger *CustomLogger) Fatal(logRecord *LogRecord) {
	logger.Log(FATAL, logRecord, DefaultLogCallDepth)
}

func (logger *CustomLogger) Fixed(logRecord *LogRecord) {
	logger.Log(FIXED, logRecord, DefaultLogCallDepth)
}
