package utils

import (
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"syscall"

	"context"

	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gazoon/go-utils/logging"
	"net/http"
	"path"
	"time"
)

type ContextKey int

var (
	RequestIdCtxKey = ContextKey(1)
)

func ObjToString(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("cannot represent as json: %s", err)
	}
	return string(b)
}

func GetCurrentFileDir() string {
	_, file, _, _ := runtime.Caller(1)
	return path.Dir(file)
}

func WaitingForShutdown() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Infof("Received shutdown signal: %s", <-ch)
}

func FunctionName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func TimestampMilliseconds() int {
	return int(time.Now().UnixNano() / 1000000)
}

func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func PrepareContext(requestID string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIdCtxKey, requestID)
	logger := logging.WithRequestID(requestID)
	ctx = logging.NewContext(ctx, logger)
	return ctx
}

func RecoveryHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				logger := logging.FromContext(r.Context())
				logger.Errorf("Panic recovered: %s", err)
			}
		}()
		h.ServeHTTP(w, r)
	})
}
