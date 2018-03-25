package utils

import (
	"io/ioutil"
	"os"
	"path"

	"strings"

	"github.com/mitchellh/mapstructure"
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
	Env        string
	FileParser func([]byte, interface{}) error
	Extension  string
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

	if parser.options.FileParser == nil {
		parser.options.FileParser = yaml.Unmarshal
	}
	parser.envDir = path.Join(configDir, parser.options.Env)

	return parser
}

func (self *ConfigParser) Parse(out interface{}) error {
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
	writeToOut(configData, out)
	return nil
}

func (self *ConfigParser) getAllFiles() ([]string, error) {
	uniqueFiles := map[string]bool{}
	for _, dirPath := range []string{self.configDir, self.envDir} {
		dirFiles, err := ioutil.ReadDir(dirPath)
		if err != nil {
			return nil, err
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
		return err
	}
	if exists {
		fileData := map[string]interface{}{}
		err = self.options.FileParser(b, fileData)
		if err != nil {
			return err
		}
		resultData = mergeData(resultData, fileData)
	}
	return nil
}

func writeToOut(configData map[string]interface{}, out interface{}) error {
	outMap, ok := out.(map[string]interface{})
	if ok {
		for k, v := range configData {
			outMap[k] = v
		}
	}

	return mapstructure.Decode(configData, out)
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
		return nil, false, err
	}
	return b, true, nil
}
