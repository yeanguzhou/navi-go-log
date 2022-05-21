package navi_go_log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LogHandle interface {
	io.WriteCloser
	WriteString(s string) (n int, err error)
}

type Priority int

const (
	// Severity.

	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

const (
	// Facility.

	// From /usr/include/sys/syslog.h.
	// These are the same up to LOG_FTP on Linux, BSD, and OS X.
	LOG_KERN Priority = iota << 3
	LOG_USER
	LOG_MAIL
	LOG_DAEMON
	LOG_AUTH
	LOG_SYSLOG
	LOG_LPR
	LOG_NEWS
	LOG_UUCP
	LOG_CRON
	LOG_AUTHPRIV
	LOG_FTP
	_ // unused
	_ // unused
	_ // unused
	_ // unused
	LOG_LOCAL0
	LOG_LOCAL1
	LOG_LOCAL2
	LOG_LOCAL3
	LOG_LOCAL4
	LOG_LOCAL5
	LOG_LOCAL6
	LOG_LOCAL7
)

type sysConn struct {
	conn       net.Conn
	createTime int64
	lifeTime   int64
	timeOut    time.Duration
}

func (s *sysConn) timeout() {
	s.conn.SetDeadline(time.Now().Add(s.timeOut * time.Millisecond))
}

func (s *sysConn) isOld() bool {
	return time.Now().Unix()-s.createTime > s.lifeTime
}

type SysLogHandle struct {
	priority Priority //等级
	addr     string   //连接地址
	daemon   bool     //后台
	stopTag  chan int //发送协程

	netPool *queue // 连接池
	buff    *queue //缓存队列

	limit     chan int
	waitGroup sync.WaitGroup //并发控制
	filePath  string         //缓存文件
	batchSize int            // 批发条数
	linger    int64          //延时等待时间
	timeout   time.Duration  //发送超时时间
	lifeTime  int64          //连接最大生存时间
}

func (S *SysLogHandle) Write(b []byte) (n int, err error) {
	return S.WriteString(string(b))
}

func (S *SysLogHandle) WriteString(msg string) (n int, err error) {
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	pri := LOG_LOCAL0 + S.priority
	message := fmt.Sprintf("<%d>%s", pri, msg)
	S.buff.Put(message)
	return len(msg), nil
}

func (S *SysLogHandle) Close() error {
	S.daemon = false
	<-S.stopTag
	count := 0
	buff := new(bytes.Buffer)
	for !S.buff.Empty() {
		content, ok := S.buff.Get()
		if ok {
			buff.WriteString(content.(string))
			count++
		}
		if count >= S.batchSize {
			S.waitGroup.Add(1)
			S.emit(buff.Bytes())
			buff.Reset()
			count = 0
		}
	}
	if buff.Len() > 0 {
		S.waitGroup.Add(1)
		S.emit(buff.Bytes())
	}

	S.waitGroup.Wait() //等待所有发送结束
	for !S.netPool.Empty() {
		conn, ok := S.netPool.Get()
		if !ok { // have no connect to use
			continue
		}
		connect, ok := conn.(*sysConn)
		if !ok { //conn is not sysconn
			continue
		}
		connect.conn.Close()
	}
	S.buff.Close()
	S.netPool.Close()
	return nil
}

func (S *SysLogHandle) scanBuffer() {
	defer func() {
		S.stopTag <- 1
	}()
	start := time.Now().Unix()
	count := 0
	for S.daemon {
		buff := new(bytes.Buffer)
		for count < S.batchSize && time.Now().Unix()-start < S.linger {
			content, ok := S.buff.Get()
			if ok {
				buff.WriteString(content.(string))
				count++
			}
		}
		if count > 0 {
			S.waitGroup.Add(1)
			go S.emit(buff.Bytes())
		}
		if count < S.batchSize {
			S.scanFile()
		}
		count = 0
		start = time.Now().Unix()
	}
}

func (S *SysLogHandle) getConn() *sysConn {
	var c *sysConn
	length := S.netPool.Size()
	if length == 0 {
		S.createConn()
		length = 1
	}
	for i := 0; i < length; i++ {
		conn, ok := S.netPool.Get()
		if !ok { // have no connect to use
			continue
		}
		connect, ok := conn.(*sysConn)
		if !ok { //conn is not sysconn
			continue
		}
		if connect.isOld() {
			connect.conn.Close()
			continue
		}
		c = connect
		break
	}
	return c
}

func (S *SysLogHandle) emit(b []byte) {
	defer S.waitGroup.Add(-1)
	select {
	case S.limit <- 1:
		defer func() {
			<-S.limit
		}()
		conn := S.getConn()
		if conn == nil {
			S.writeFile(b)
			return
		}
		conn.timeout()
		_, err := conn.conn.Write(b)

		if err != nil {
			fmt.Fprintln(os.Stderr, "syslog send fail, write file", err)
			S.writeFile(b)
			conn.conn.Close()
			return
		}
		S.netPool.Put(conn)

	case <-time.After(time.Millisecond * 10):
		fmt.Fprintln(os.Stderr, "flow control, write file")
		S.writeFile(b)
	}
	return
}

func (S *SysLogHandle) init() {
	filePath, ok := os.LookupEnv("SYSLOG_BUFFER")
	if !ok {
		filePath = "/data/syslog_buffer"
	}
	S.filePath = strings.TrimSuffix(filePath, "/")

	batchSize, ok := os.LookupEnv("BATCH_SIZE")
	if !ok {
		batchSize = "1000"
	}
	S.batchSize, _ = strconv.Atoi(batchSize)

	Linger, ok := os.LookupEnv("Linger")
	if !ok {
		Linger = "3"
	}
	S.linger, _ = strconv.ParseInt(Linger, 10, 64)

	timeOut, ok := os.LookupEnv("SYSLOG_TIMEOUT")
	if !ok {
		timeOut = "3000"
	}
	timeout, _ := strconv.Atoi(timeOut)
	S.timeout = time.Duration(timeout)

	lifeTime, ok := os.LookupEnv("SYSLOG_CONN_LIFE_TIME")
	if !ok {
		lifeTime = "100"
	}
	S.lifeTime, _ = strconv.ParseInt(lifeTime, 10, 64)
	err := os.MkdirAll(S.filePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	S.createConn()
	go S.scanBuffer()
}

func (S *SysLogHandle) createConn() {
	conn, err := net.DialTimeout("tcp", S.addr, time.Millisecond*S.timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		S.netPool.Put("")
		return
	}
	S.netPool.Put(&sysConn{conn: conn, createTime: time.Now().Unix(), lifeTime: S.lifeTime, timeOut: S.timeout})
}

func Dial(network, addr string, priority Priority) (*SysLogHandle, error) {
	if priority < LOG_EMERG || priority > LOG_DEBUG {
		return nil, errors.New("log/syslog: invalid priority")
	}

	if network != "tcp" {
		return nil, errors.New("syslog only support tcp")
	}

	w := &SysLogHandle{
		priority: priority,
		addr:     addr,
		netPool:  NewQueue(30, time.Millisecond*10),
		buff:     NewQueue(100000, time.Millisecond*10),
		daemon:   true,
		stopTag:  make(chan int),
		limit:    make(chan int, 30),
	}
	w.init()
	return w, nil
}

func (S *SysLogHandle) writeFile(data []byte) {
	//f := getRandomString(32)
	fileName := fmt.Sprintf("%s/%d", S.filePath, time.Now().UnixNano())
	err := ioutil.WriteFile(fileName, data, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return
}

func (S *SysLogHandle) scanFile() {
	filePath := S.filePath
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if len(files) == 0 {
		return
	}
	n := rand.Intn(len(files))
	name := files[n].Name()
	fileName := filePath + "/" + name
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if len(content) > 0 {
		S.waitGroup.Add(1)
		go S.emit(content)
	}
	os.Remove(fileName)
}

//func getRandomString(length int) string {
//	str := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
//	result := make([]byte, 32)
//	r := rand.New(rand.NewSource(time.Now().UnixNano()))
//	for i := 0; i < length; i++ {
//		result = append(result, str[r.Intn(len(str))])
//	}
//	return string(result)
//}
