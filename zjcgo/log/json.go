package log

import (
	"encoding/json"
	"fmt"
	"time"
)

type JsonFormatter struct {
	TimeDisplay bool
}

func (f *JsonFormatter) Formatter(param *LoggingFormatterParam) string {
	now := time.Now()
	if param.LoggerFields == nil {
		param.LoggerFields = make(Fields)
	}
	if f.TimeDisplay {
		timeNow := now.Format("2006/01/02 - 15:04:05")
		param.LoggerFields["log_time"] = timeNow
	}
	param.LoggerFields["msg"] = param.Msg
	marshal, _ := json.Marshal(param.LoggerFields)
	return fmt.Sprint(string(marshal))
}

func (f *JsonFormatter) LevelColor(level LoggerLevel) string {
	switch level {
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

func (f *JsonFormatter) MsgColor(level LoggerLevel) string {
	switch level {
	case LevelDebug:
		return ""
	case LevelInfo:
		return ""
	case LevelError:
		return red
	default:
		return cyan
	}
}
