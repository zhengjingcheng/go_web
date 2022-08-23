package log

import (
	"fmt"
	"github.com/zhengjingcheng/zjcgo/msstrings"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

type LoggerLevel int

//添加其他字段
type Fields map[string]any

//log级别
const (
	LevelDebug LoggerLevel = iota
	LevelInfo
	LevelError
)

type Logger struct {
	Level        LoggerLevel    //日志级别
	Outs         []LoggerWriter //输出流数组，因为可能不只是输出到控制台，所以开一个数组
	Formatter    LoggingFormatter
	LoggerFields Fields //其他字段
	logPath      string //日志文件保存路径
	LogFileSize  int64  //M为单位
}

//日志输出方式
type LoggerWriter struct {
	Level LoggerLevel
	Out   io.Writer
}

//格式化日志文件输出
type LoggerFormatter struct {
	Color        bool //是不是带颜色
	Level        LoggerLevel
	loggerFields Fields
}

type LoggingFormatter interface {
	Formatter(param *LoggingFormatterParam) string
}

type LoggingFormatterParam struct {
	Color        bool
	Level        LoggerLevel
	Msg          any
	LoggerFields Fields
}

//新建一个日志
func New() *Logger {
	return &Logger{}
}

//初始化
func Default() *Logger {
	logger := New()
	out := LoggerWriter{Out: os.Stdout}    //标准输出
	logger.Outs = append(logger.Outs, out) //添加输出方式
	logger.Level = LevelDebug
	logger.Formatter = &TextFormatter{}
	return logger
}

//三级日志 参数:任意格式数据
func (l *Logger) Info(msg any) {
	l.Print(LevelInfo, msg)
	//l.Error(msg)
}

func (l *Logger) Debug(msg any) {
	l.Print(LevelDebug, msg)
	//l.Info(msg)
	//l.Error(msg)
}

func (l *Logger) Error(msg any) {
	l.Print(LevelError, msg)
}

func (l *Logger) WithFields(fields Fields) *Logger {
	return &Logger{
		Formatter:    l.Formatter,
		Outs:         l.Outs,
		Level:        l.Level,
		LoggerFields: fields, //相当于在原来的基础上增加了一个
	}
}
func (l *Logger) Print(level LoggerLevel, msg any) {
	if l.Level > level {
		//级别不满足 不打印日志
		return
	}
	param := &LoggingFormatterParam{
		Level:        level,
		LoggerFields: l.LoggerFields,
		Msg:          msg,
	}
	formatter := l.Formatter.Formatter(param)
	//fmt.Println(msg)
	for _, out := range l.Outs {
		if out.Out == os.Stdout {
			param.Color = true
			formatter = l.Formatter.Formatter(param)
			fmt.Fprintln(out.Out, formatter)
		}
		if out.Level == -1 || out.Level == level {
			fmt.Fprint(out.Out, formatter)
		}
	}
}
func (l *Logger) SetLogPath(logPath string) {
	l.logPath = logPath
	//写入文件
	all, err := FileWriter(path.Join(l.logPath, "all.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, LoggerWriter{Level: -1, Out: all})
	debug, err := FileWriter(path.Join(l.logPath, "debug.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, LoggerWriter{Level: LevelDebug, Out: debug})
	info, err := FileWriter(path.Join(l.logPath, "info.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, LoggerWriter{Level: LevelInfo, Out: info})
	logError, err := FileWriter(path.Join(l.logPath, "error.log"))
	if err != nil {
		panic(err)
	}
	l.Outs = append(l.Outs, LoggerWriter{Level: LevelError, Out: logError})
}

func (f *LoggerFormatter) formatter(msg any, fields Fields) string {
	now := time.Now()
	if f.Color {
		//要带颜色  error的颜色 为红色 info为绿色 debug为蓝色
		levelColor := f.LevelColor()
		msgColor := f.MsgColor()
		return fmt.Sprintf("%s [zjcgo] %s %s%v%s | level= %s %s %s | msg=%s %#v %s %#v\n",
			yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
			levelColor, f.Level.Level(), reset, msgColor, msg, reset, fields,
		)
	}
	return fmt.Sprintf("[zjcgo] %v | level = %s | msg = %#v \n",
		now.Format("2006/01/02 - 15:04:05"),
		f.Level.Level(), msg, fields,
	)
}
func (level LoggerLevel) Level() string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

func (f *LoggerFormatter) LevelColor() string {
	switch f.Level {
	case LevelDebug:
		return blue
	case LevelInfo:
		return green
	case LevelError:
		return red
	default:
		return cyan
	}
}

func (f *LoggerFormatter) MsgColor() string {
	switch f.Level {
	case LevelDebug:
		return blue
	case LevelInfo:
		return green
	case LevelError:
		return red
	default:
		return cyan
	}
}

func FileWriter(name string) (io.Writer, error) {
	w, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	return w, err
}

func (l *Logger) CloseWriter() {
	for _, out := range l.Outs {
		file := out.Out.(*os.File)
		if file != nil {
			_ = file.Close()
		}
	}
}
func (l *Logger) CheckFileSize(out *LoggerWriter) {
	osFile := out.Out.(*os.File)
	if osFile != nil {
		stat, err := osFile.Stat()
		if err != nil {
			log.Println("logger checkFileSize error info :", err)
			return
		}
		size := stat.Size()
		//这里要检查大小，如果满足条件 就重新创建文件，并且更换logger中的输出
		if l.LogFileSize <= 0 {
			//默认100M
			l.LogFileSize = 100 << 20
		}
		if size >= l.LogFileSize {
			_, fileName := path.Split(osFile.Name())
			name := fileName[0:strings.Index(fileName, ".")]
			w, err := FileWriter(path.Join(l.logPath, msstrings.JoinStrings(name, ".", time.Now().UnixMilli(), ".log")))
			if err != nil {
				log.Println("logger checkFileSize error info :", err)
				return
			}
			out.Out = w
		}
	}

}
