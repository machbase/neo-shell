package ginutil

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	logging "github.com/machbase/neo-logging"
	"github.com/machbase/neo-shell/util"
	"github.com/teris-io/shortid"
)

const LOGGING = "neo-shell/ginlog"
const TRACKID = "neo-shell/trackid"
const HEADER_TRACKID = "x-agwd-trackid"

func RecoveryWithLogging(log logging.Log, recovery ...gin.RecoveryFunc) gin.HandlerFunc {
	gin.DefaultWriter = log
	gin.DefaultErrorWriter = log

	if len(recovery) > 0 {
		return gin.CustomRecoveryWithWriter(log, recovery[0])
	}
	return gin.CustomRecoveryWithWriter(log, func(c *gin.Context, err any) {
		trackId := c.GetString(TRACKID)
		if len(trackId) > 0 {
			log.Errorf("%s panic %s", trackId, err)
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

type HttpLoggerFilter func(req *http.Request, statusCode int, latency time.Duration) bool

func HttpLogger(loggingName string) gin.HandlerFunc {
	return HttpLoggerWithFilter(loggingName, nil)
}

func HttpLoggerWithFilter(loggingName string, filter HttpLoggerFilter) gin.HandlerFunc {
	log := logging.GetLog(loggingName)
	return logger(log, filter)
}

func HttpLoggerWithFile(loggingName string, filename string) gin.HandlerFunc {
	return HttpLoggerWithFileConf(loggingName,
		logging.LogFileConf{
			Filename:             filename,
			Level:                "DEBUG",
			MaxSize:              10,
			MaxBackups:           2,
			MaxAge:               7,
			Compress:             false,
			Append:               true,
			RotateSchedule:       "@midnight",
			Console:              false,
			PrefixWidth:          20,
			EnableSourceLocation: false,
		})
}

func HttpLoggerWithFileConf(loggingName string, fileConf logging.LogFileConf) gin.HandlerFunc {
	return HttpLoggerWithFilterAndFileConf(loggingName, nil, fileConf)
}

func HttpLoggerWithFilterAndFileConf(loggingName string, filter HttpLoggerFilter, fileConf logging.LogFileConf) gin.HandlerFunc {
	if len(fileConf.Filename) > 0 {
		return logger(logging.NewLogFile(loggingName, fileConf), filter)
	} else {
		return HttpLoggerWithFilter(loggingName, filter)
	}
}

func logger(log logging.Log, filter HttpLoggerFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var trxId string
		if id := c.Request.Header.Get(HEADER_TRACKID); len(id) > 0 {
			trxId = id
		} else {
			trxId, _ = shortid.Generate()
			trxId = util.StrPad(trxId, 10, "_", "RIGHT")
		}
		c.Set(TRACKID, trxId)
		c.Set(LOGGING, log)

		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// ignore health checker
		if strings.HasSuffix(c.Request.URL.Path, "/healthz") && c.Request.Method == http.MethodGet {
			return
		}

		// Stop timer
		TimeStamp := time.Now()
		Latency := TimeStamp.Sub(start)

		StatusCode := c.Writer.Status()

		// filter가 존재하고 log를 남기지 않도록 false를 반환한 경우
		if filter != nil && !filter(c.Request, StatusCode, Latency) {
			return
		}

		url := c.Request.Host + c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if len(raw) > 0 {
			url = url + "?" + raw
		}

		ClientIP := c.ClientIP()
		Proto := c.Request.Proto
		Method := c.Request.Method
		ErrorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()
		if len(ErrorMessage) > 0 {
			ErrorMessage = "\n" + ErrorMessage
		}

		WriteSize := c.Writer.Size()
		if WriteSize == -1 {
			WriteSize = 0
		}
		ReadSize := c.Request.ContentLength

		color := ""
		reset := "\033[0m"
		level := logging.LevelDebug

		switch {
		case StatusCode >= http.StatusContinue && StatusCode < http.StatusOK:
			color, reset = "", "" // 1xx
		case StatusCode >= http.StatusOK && StatusCode < http.StatusMultipleChoices:
			color = "\033[97;42m" // 2xx green
		case StatusCode >= http.StatusMultipleChoices && StatusCode < http.StatusBadRequest:
			color = "\033[90;47m" // 3xx white
		case StatusCode >= http.StatusBadRequest && StatusCode < http.StatusInternalServerError:
			color = "\033[90;43m" // 4xx yellow
			level = logging.LevelWarn
		default:
			color = "\033[97;41m" // 5xx red
			level = logging.LevelError
		}

		log.Logf(level, "%-10s |%s %3d %s| %13v | %15s | %5d | %5d | %s %-7s %s%s",
			trxId,
			color, StatusCode, reset,
			Latency,
			ClientIP,
			ReadSize,
			WriteSize,
			Proto,
			Method,
			url,
			ErrorMessage,
		)
	}
}
