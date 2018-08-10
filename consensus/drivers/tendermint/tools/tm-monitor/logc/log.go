package logc

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	log "github.com/inconshreveable/log15"
)

type LogMonitor struct {
	keyWord string        // 查找日志时的关键字
	logPath string        // 日志存放路径
	channel chan *LogInfo // 用于传出匹配结果的channel
	bRegex  bool          // 是否使用正则表达式匹配
}

type LogInfo struct {
	text string
	time time.Time
}

func NewLogMonitor(k string, b bool) *LogMonitor {
	path := "/usr/local/gopath/src/gitlab.33.cn/chain33/chain33/build/logs/chain33.log"
	return &LogMonitor{k, path, make(chan *LogInfo), b}
}

func (l *LogMonitor) GetCurrentLog() (chan<- *tail.Line, error) {
	d, err := tail.TailFile(l.logPath, tail.Config{Follow: true})
	if err != nil {
		fmt.Println("Get current string err,", err)
		return nil, err
	}

	return d.Lines, nil
}

func (l *LogMonitor) GetCurrentKeyInfo() (chan *LogInfo, error) {
	d, err := tail.TailFile(l.logPath, tail.Config{Follow: true})
	if err != nil {
		fmt.Println("Get current string err,", err)
		return nil, err
	}

	go func() {
		for line := range d.Lines {
			if l.bRegex {
				if ok, _ := regexp.MatchString(l.keyWord, line.Text); ok {
					l.channel <- &LogInfo{line.Text, line.Time}
				}
			} else {
				if strings.Contains(line.Text, l.keyWord) {
					l.channel <- &LogInfo{line.Text, line.Time}
				}
			}
		}
	}()
	return l.channel, nil
}

func ParseFromConfigFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Error("File %s is not existed...", path)
	}
	defer file.Close()
	data, _ := ioutil.ReadAll(file)

	//words := strings.FieldsFunc(string(data), func(ch rune) bool {
	//	return !unicode.IsLetter(ch) //任何不是字母的字符都是分隔符
	//})

	fmt.Println(string(data))

}

//send alarm  to monitor
func httpPost(pUrl string, pBody string) {
	resp, err := http.Post(pUrl,
		"application/x-www-form-urlencoded",
		strings.NewReader(pBody))
	if err != nil {
		log.Error("httpPost  error:" + err.Error())
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
		log.Error("no resp  error:" + err.Error())
		return
	}
	fmt.Println(string(body))
	log.Debug("ugslb resp with body:%s", string(body))
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Error("in getCurrentDirectory get path error:" + err.Error())
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//parse the content like  st:20170106180425
func parseTimeVal(dateContent string) (dateTime string) {
	//fmt.Println(dateContent)
	log.Debug("date to parse:%s", dateContent)
	content := strings.Split(dateContent, ":")

	timeVal := ""

	if 0 != len(content[1]) {
		timeformatdate, _ := time.Parse("20060102150405", content[1])
		parsedTime := timeformatdate.Format("2006-01-02 15:04:05")
		log.Debug("timeVal:%s", parsedTime)
		timeVal = parsedTime
	}
	return timeVal
}

//parse line content like st:20171221150609 et:20171221150614 http://guo1guang.live.OOotvcloud.com/otv/xjgg/livess/channel44/700.m3u8 part
func parseLine(line string) (st, et, url string) {
	content := strings.Split(line, " ")
	fmt.Println(content, len(content))
	stVal := ""
	etVal := ""
	urlVal := ""
	for _, a := range content {
		if 0 != len(a) {
			log.Debug("to parse %s, length %d", a, len(a))
			if strings.Contains(a, "st") {
				stVal = parseTimeVal(a)
				continue
			}
			if strings.Contains(a, "et") {
				etVal = parseTimeVal(a)
				continue
			}
			if strings.Contains(a, "http") {
				urlVal = a
			}
		}
	}
	return stVal, etVal, urlVal
}

//func alarmProcess(fileName string) {
//	//analyze the log
//	file, err := os.Open(fileName)
//	check(err)
//	defer file.Close()
//
//	var alarm_sliceSend []outAlarmPara
//	const blockSize = 1024 * 32
//	const timeoutPattern = "st.*part$"
//	const lostPattern = "st.*404$"
//	//log.Debug("cur pos:%d, read block size:%d", pos, blockSize)
//
//	for {
//		o2, err := file.Seek(pos, 0)
//		check(err)
//
//		b2 := make([]byte, blockSize)
//		n2, err := file.Read(b2)
//		//fmt.Printf("%d bytes @ %d: %s\n", n2, o2, string(b2))
//
//		lines := strings.Split(string(b2), "\n")
//		for i, a := range lines {
//			//fmt.Printf("line:%d, value:%q\n", i, a)
//			//check if it is timeout
//			reg, err := regexp.Compile(timeoutPattern)
//			if err != nil {
//				log.Warn(err.Error())
//				return
//			}
//
//			if true == reg.MatchString(a) {
//				//parse the line
//				st, et, url := parseLine(a)
//				log.Debug("parse Res: st %s, et %s, url %s", st, et, url)
//				var outAlarm outAlarmPara
//				outAlarm.Alarmtype = AlarmTypeTimeout
//				outAlarm.Alarmlevel = 10
//				outAlarm.Url = url
//				outAlarm.Alarmip = g_ip
//				outAlarm.Alarmtimer = st
//				outAlarm.Alarmcontext = "超时"
//				alarm_sliceSend = append(alarm_sliceSend, outAlarm)
//				log.Debug("part detected line:%d,%q,%q", i, a, alarm_sliceSend)
//				continue
//			}
//
//			//check if  it is not found
//			reg, err = regexp.Compile(lostPattern)
//			if err != nil {
//				fmt.Println(err)
//				return
//			}
//			//fmt.Printf("lost %q\n", reg.FindAllString(a, -1))
//			if true == reg.MatchString(a) {
//				st, et, url := parseLine(a)
//				log.Debug("parse Res: st %s, et %s, url %s", st, et, url)
//				var outAlarm outAlarmPara
//				outAlarm.Alarmtype = AlarmTypeLost
//				outAlarm.Alarmlevel = 10
//				outAlarm.Url = url
//				outAlarm.Alarmip = g_ip
//				outAlarm.Alarmtimer = st
//				outAlarm.Alarmcontext = "缺片"
//				alarm_sliceSend = append(alarm_sliceSend, outAlarm)
//			}
//		}
//		pos += int64(n2)
//		log.Debug("offset %d, read %d, pos %d", o2, n2, pos)
//		if n2 != blockSize {
//			log.Debug("reach end of file")
//			break
//		}
//	}
//	//notify monitor when the timeout happened
//	outM := &outAlarmBody{
//		g_cpid,
//		alarm_sliceSend,
//	}
//	b, err := json.Marshal(outM)
//	if err != nil {
//		log.Error("json encode message failed, %s", string(b))
//	} else {
//		log.Debug("post content:%s", string(b))
//		httpPost(g_remoteInterface, string(b))
//	}
//}

//func tail(filename string, n int) (lines []string, err error) {
//	f, e := os.Stat(filename)
//	if e == nil {
//		size := f.Size()
//		var fi *os.File
//		fi, err = os.Open(filename)
//		if err == nil {
//			b := make([]byte, defaultBufSize)
//			sz := int64(defaultBufSize)
//			nn := n
//			bTail := bytes.NewBuffer([]byte{})
//			istart := size
//			flag := true
//			for flag {
//				if istart < defaultBufSize {
//					sz = istart
//					istart = 0
//					//flag = false
//				} else {
//					istart -= sz
//				}
//				_, err = fi.Seek(istart, os.SEEK_SET)
//				if err == nil {
//					mm, e := fi.Read(b)
//					if e == nil && mm > 0 {
//						j := mm
//						for i := mm - 1; i >= 0; i-- {
//							if b[i] == '\n' {
//								bLine := bytes.NewBuffer([]byte{})
//								bLine.Write(b[i+1 : j])
//								j = i
//								if bTail.Len() > 0 {
//									bLine.Write(bTail.Bytes())
//									bTail.Reset()
//								}
//
//								if (nn == n && bLine.Len() > 0) || nn < n { //skip last "\n"
//									lines = append(lines, bLine.String())
//									nn --
//								}
//								if nn == 0 {
//									flag = false
//									break
//								}
//							}
//						}
//						if flag && j > 0 {
//							if istart == 0 {
//								bLine := bytes.NewBuffer([]byte{})
//								bLine.Write(b[:j])
//								if bTail.Len() > 0 {
//									bLine.Write(bTail.Bytes())
//									bTail.Reset()
//
//								}
//								lines = append(lines, bLine.String())
//								flag = false
//							} else {
//								bb := make([]byte, bTail.Len())
//								copy(bb, bTail.Bytes())
//								bTail.Reset()
//								bTail.Write(b[:j])
//								bTail.Write(bb)
//							}
//						}
//					}
//				}
//			}
//			//func (f *File) Seek(offset int64, whence int) (ret int64, err error)
//			//func (f *File) Read(b []byte) (n int, err error) {
//		}
//		defer fi.Close()
//	}
//	return
//}
