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
	"github.com/gazoon/go-utils/request"
	"net/http"
	"path"
	"sync"
	"time"
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

func TimestampSeconds() int {
	return int(TimestampMilliseconds() / 1000)
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

func CreateContext() context.Context {
	ctx := context.Background()
	return FillContext(ctx)
}

func FillContext(ctx context.Context) context.Context {
	requestId := request.NewRequestId()
	ctx = request.NewContext(ctx, requestId)
	logger := logging.WithRequestID(requestId)
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

func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
