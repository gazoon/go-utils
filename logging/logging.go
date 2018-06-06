package logging

import (
	"strings"

	"context"
	"net/http"

	"fmt"

	log "github.com/Sirupsen/logrus"
)

var (
	gLogger = WithPackage("logging")
)

type ContextKey int

const (
	ServiceNameField = "service_name"
	RequestIDField   = "request_id"
	PackageField     = "package"

	loggerCtxKey = ContextKey(1)
)

type LoggerMixin struct {
	Logger *log.Entry
}

func NewLoggerMixin(packageName string, additional log.Fields) *LoggerMixin {
	logger := WithPackage(packageName)
	if len(additional) > 0 {
		logger = logger.WithFields(additional)
	}
	return &LoggerMixin{logger}
}

func (l *LoggerMixin) GetLogger(ctx context.Context) *log.Entry {
	return FromContextAndBase(ctx, l.Logger)
}

type customFormatter struct {
	logFormatter     log.Formatter
	additionalFields log.Fields
}

func WithPackage(packageName string) *log.Entry {
	return log.WithField(PackageField, packageName)
}

func WithRequestID(requestID string) *log.Entry {
	return log.WithField(RequestIDField, requestID)
}

func WithRequestIDAndBase(requestID string, base *log.Entry) *log.Entry {
	requestLogger := WithRequestID(requestID)
	return base.WithFields(requestLogger.Data)
}

func (cf *customFormatter) Format(e *log.Entry) ([]byte, error) {
	data := make(log.Fields, len(e.Data)+len(cf.additionalFields))
	for k, v := range e.Data {
		data[k] = v
	}
	for k, v := range cf.additionalFields {
		data[k] = v
	}
	var newEntry = new(log.Entry)
	*newEntry = *e
	newEntry.Data = data
	return cf.logFormatter.Format(newEntry)
}

func FromContext(ctx context.Context) *log.Entry {
	logger, ok := ctx.Value(loggerCtxKey).(*log.Entry)
	if !ok {
		logger = log.NewEntry(log.StandardLogger())
	}
	return logger
}

func FromContextAndBase(ctx context.Context, base *log.Entry) *log.Entry {
	ctxLogger := FromContext(ctx)
	return base.WithFields(ctxLogger.Data)
}

func NewContext(ctx context.Context, logger *log.Entry) context.Context {
	oldLogger, ok := ctx.Value(loggerCtxKey).(*log.Entry)
	if ok {
		logger = oldLogger.WithFields(logger.Data)
	}
	return context.WithValue(ctx, loggerCtxKey, logger)
}

func NewContextBackground(logger *log.Entry) context.Context {
	return NewContext(context.Background(), logger)
}

func NewFormatter(serviceName string) log.Formatter {
	formatter := &customFormatter{logFormatter: &log.TextFormatter{}, additionalFields: log.Fields{
		ServiceNameField: serviceName,
	}}
	return formatter
}

func getLogLevel(logLevelName string) log.Level {
	var logLevel log.Level
	switch strings.ToLower(logLevelName) {
	case "debug":
		logLevel = log.DebugLevel
	case "info":
		logLevel = log.InfoLevel
	case "warning":
		logLevel = log.WarnLevel
	case "error":
		logLevel = log.ErrorLevel
	default:
		logLevel = log.InfoLevel
	}
	return logLevel
}

func PatchStdLog(logLevelName, serviceName string) {
	logLevel := getLogLevel(logLevelName)
	formatter := NewFormatter(serviceName)
	log.SetLevel(logLevel)
	log.SetFormatter(formatter)
}

func StartLevelToggle(togglePath string, port int) {
	mux := http.NewServeMux()
	mux.HandleFunc(togglePath, func(w http.ResponseWriter, r *http.Request) {
		levelName := r.FormValue("level")
		level := getLogLevel(levelName)
		gLogger.Infof("Toggle global log level from %s to %s", log.GetLevel(), level)
		log.SetLevel(level)
		fmt.Fprint(w, level)
	})
	gLogger.Infof("Toggle server is running on %d port, path=%s", port, togglePath)
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
	}()
}
