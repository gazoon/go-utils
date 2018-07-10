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
	"github.com/getsentry/raven-go"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

var WordsSplitRegexp = regexp.MustCompile(`(["=@'|/<>\\;:.,\s!?]+)`)

func SplitWordsLowered(text string) []string {
	words := SplitWords(text)
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	return words
}

func SplitWords(text string) []string {
	var result []string
	for _, w := range WordsSplitRegexp.Split(text, -1) {
		if len(w) != 0 {
			result = append(result, w)
		}
	}
	return result
}

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
	signalCode := <-ch
	log.WithField("signal", signalCode).Info("Received shutdown signal")
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

func UTCNow() time.Time {
	return time.Now().UTC()
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

func InitializeSentry(dsn string) error {
	err := raven.SetDSN(dsn)
	if err != nil {
		return errors.Wrap(err, "raven set dsn")
	}
	raven.SetEnvironment(GetEnv())
	return nil
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

func ConvertBsonToMap(data interface{}) interface{} {
	switch item := data.(type) {
	case bson.M:
		result := make(map[string]interface{}, len(item))
		for k, v := range item {
			result[k] = ConvertBsonToMap(v)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(item))
		for k, v := range item {
			result[k] = ConvertBsonToMap(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(item))
		for i, v := range item {
			result[i] = ConvertBsonToMap(v)
		}
		return result
	default:
		return data
	}
	return data
}

func GetEnv() string {
	env := os.Getenv("ENV")
	if env == "" {
		return "dev"
	}
	return env
}

func ExtractLastPathPart(uri string) (string, error) {
	if uri == "" {
		return "", errors.New("empty url")
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", errors.Wrap(err, "parse url")
	}
	urlPath := strings.TrimSuffix(u.Path, "/")
	_, lastPart := path.Split(urlPath)
	if lastPart == "" {
		return "", errors.New("url has empty path")
	}
	return lastPart, nil
}
