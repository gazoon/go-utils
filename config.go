package utils

import (
	"io/ioutil"
	"os"
	"path"

	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	rootFile  = "main"
	localMark = ".local"
)

func ParseConfig(configDir string, out interface{}, options ...func(*ConfigParserOptions)) error {
	parser := NewConfigParser(configDir, options...)
	return parser.Parse(out)
}

type ConfigParserOptions struct {
	Env         string
	Transformer DataTransformer
	Extension   string
}

type DataTransformer interface {
	Unmarshal([]byte, interface{}) error
	Marshal(interface{}) ([]byte, error)
}

type YamlTransformer struct{}

func (self YamlTransformer) Unmarshal(in []byte, out interface{}) error {
	err := yaml.Unmarshal(in, out)
	return errors.Wrap(err, "yaml unmarshal")
}

func (self YamlTransformer) Marshal(in interface{}) ([]byte, error) {
	b, err := yaml.Marshal(in)
	return b, errors.Wrap(err, "yaml marshal")
}

type ConfigParser struct {
	configDir string
	envDir    string

	options ConfigParserOptions
}

func NewConfigParser(configDir string, options ...func(*ConfigParserOptions)) *ConfigParser {
	parser := &ConfigParser{}
	for _, option := range options {
		option(&parser.options)
	}

	parser.configDir = configDir

	if parser.options.Env == "" {
		parser.options.Env = os.Getenv("ENV")
		if parser.options.Env == "" {
			parser.options.Env = "dev"
		}
	}

	if parser.options.Extension == "" {
		parser.options.Extension = ".yaml"
	}

	if parser.options.Transformer == nil {
		parser.options.Transformer = YamlTransformer{}
	}
	parser.envDir = path.Join(configDir, parser.options.Env)

	return parser
}

func (self *ConfigParser) Parse(out interface{}) error {
	err := self.parse(out)
	return errors.Wrap(err, "can't parse config")
}

func (self *ConfigParser) parse(out interface{}) error {
	configData, err := self.processFile(rootFile)
	if err != nil {
		return err
	}
	otherFiles, err := self.getAllFiles()
	if err != nil {
		return err
	}
	for _, fileName := range otherFiles {
		data, err := self.processFile(fileName)
		if err != nil {
			return err
		}
		configData[fileName] = data
	}
	self.writeToOut(configData, out)
	return nil
}

func (self *ConfigParser) getAllFiles() ([]string, error) {
	uniqueFiles := map[string]bool{}
	for _, dirPath := range []string{self.configDir, self.envDir} {
		dirFiles, err := ioutil.ReadDir(dirPath)
		if err != nil {
			return nil, errors.Wrap(err, "get config dir files")
		}
		for _, info := range dirFiles {
			if info.IsDir() {
				continue
			}
			fileName := info.Name()
			if !strings.HasSuffix(fileName, self.options.Extension) {
				continue
			}
			if strings.Contains(fileName, localMark) {
				continue
			}
			if strings.Contains(fileName, rootFile) {
				continue
			}
			uniqueFiles[strings.TrimSuffix(fileName, self.options.Extension)] = true
		}
	}
	var resultFiles []string
	for fileName := range uniqueFiles {
		resultFiles = append(resultFiles, fileName)
	}
	return resultFiles, nil
}

func (self *ConfigParser) processFile(fileName string) (map[string]interface{}, error) {
	resultData := map[string]interface{}{}

	generalFile := path.Join(self.configDir, fileName+self.options.Extension)
	err := self.mergeFileData(generalFile, resultData)
	if err != nil {
		return nil, err
	}

	generalLocalFile := path.Join(self.configDir, fileName+localMark+self.options.Extension)
	err = self.mergeFileData(generalLocalFile, resultData)
	if err != nil {
		return nil, err
	}

	envFile := path.Join(self.envDir, fileName+self.options.Extension)
	err = self.mergeFileData(envFile, resultData)
	if err != nil {
		return nil, err
	}

	envLocalFile := path.Join(self.envDir, fileName+localMark+self.options.Extension)
	err = self.mergeFileData(envLocalFile, resultData)
	if err != nil {
		return nil, err
	}

	return resultData, nil
}

func (self *ConfigParser) mergeFileData(filePath string, resultData map[string]interface{}) error {
	b, exists, err := readFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "file: %s", filePath)
	}
	if exists {
		fileData := map[string]interface{}{}
		err = self.options.Transformer.Unmarshal(b, fileData)
		if err != nil {
			return errors.Wrapf(err, "file: %s", filePath)
		}
		resultData = mergeData(resultData, fileData)
	}
	return nil
}

func (self *ConfigParser) writeToOut(configData map[string]interface{}, out interface{}) error {
	outMap, ok := out.(map[string]interface{})
	if ok {
		for k, v := range configData {
			outMap[k] = v
		}
		return nil
	}
	bytesRepresentation, err := self.options.Transformer.Marshal(configData)
	if err != nil {
		return err
	}
	return self.options.Transformer.Unmarshal(bytesRepresentation, out)

}

func mergeData(to, from map[string]interface{}) map[string]interface{} {
	for k, v := range from {
		to[k] = v
	}
	return to
}

func readFile(filePath string) ([]byte, bool, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, errors.Wrap(err, "read config file")
	}
	return b, true, nil
}

type RootConfig struct {
	ServiceName string `yaml:"service_name" json:"service_name"`
	Port        int    `yaml:"port" json:"port"`
	LogLevel    string `yaml:"log_level" json:"log_level"`
}

type DatabaseSettings struct {
	Host            string `yaml:"host" json:"host"`
	Port            int    `yaml:"port" json:"port"`
	User            string `yaml:"user" json:"user"`
	Database        string `yaml:"database" json:"database"`
	Password        string `yaml:"password" json:"password"`
	Timeout         int    `yaml:"timeout" json:"timeout"`
	PoolSize        int    `yaml:"pool_size" json:"pool_size"`
	RetriesNum      int    `yaml:"retries_num" json:"retries_num"`
	RetriesInterval int    `yaml:"retries_interval" json:"retries_interval"`
}

type S3Setting struct {
	Region string `yaml:"region" json:"region"`
	Bucket string `yaml:"bucket" json:"bucket"`
}

type AwsCreds struct {
	AccountID     string `yaml:"account_id" json:"account_id"`
	AccountSecret string `yaml:"account_secret" json:"account_secret"`
}

type MongoDBSettings struct {
	DatabaseSettings `yaml:",inline" json:",inline"`
	Collection       string `yaml:"collection" json:"collection"`
}

type SentrySettings struct {
	DSN string `yaml:"dsn" json:"dsn"`
}

type TelegramSettings struct {
	APIToken    string `yaml:"api_token" json:"api_token"`
	BotName     string `yaml:"bot_name" json:"bot_name"`
	HttpTimeout int    `yaml:"http_timeout" json:"http_timeout"`
	Retries     int    `yaml:"retries" json:"retries"`
}

type GoogleAPI struct {
	APIKey      string `yaml:"api_key" json:"api_key"`
	HttpTimeout int    `yaml:"http_timeout" json:"http_timeout"`
}

type BotInfo struct {
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Admins   []int  `yaml:"admins"`
}
