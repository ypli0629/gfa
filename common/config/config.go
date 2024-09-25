package config

import (
	"fmt"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"path/filepath"
	"strings"
)

var (
	config *Config
)

type Config struct {
	koanf.Koanf

	ConfigName   string   `json:"configName"`
	ConfigType   []string `json:"configType"`
	AutomaticEnv bool     `json:"automaticEnv"`
	Paths        []string `json:"paths"`
}

type OptionFunc func(config *Config)

func WithPath(path string) OptionFunc {
	return func(config *Config) {
		config.Paths = append(config.Paths, path)
	}
}

func WithConfigName(name string) OptionFunc {
	return func(config *Config) {
		config.ConfigName = name
	}
}

func WithConfigType(configType string) OptionFunc {
	return func(config *Config) {
		if slice.IndexOf(config.ConfigType, configType) != -1 {
			config.ConfigType = append(config.ConfigType, configType)
		}
	}
}

func WithAutomaticEnv(flag bool) OptionFunc {
	return func(config *Config) {
		config.AutomaticEnv = flag
	}
}

func Setup(opts ...OptionFunc) {
	configLogger := logger.NewBasic(logger.Config{})

	config = &Config{
		ConfigName:   "application",
		ConfigType:   []string{"yaml", "yml"},
		AutomaticEnv: true,
	}
	for _, opt := range opts {
		opt(config)
	}

	supportedParsers := map[string]koanf.Parser{
		"yaml": yaml.Parser(),
		"yml":  yaml.Parser(),
	}

	config.Koanf = *koanf.New(".")
	// load from env
	if config.AutomaticEnv {
		err := config.Load(env.Provider("", ".", func(s string) string {
			return s
		}), nil)
		if err != nil {
			configLogger.Panicf(nil, "Fail to load environment variables, %s", err)
		}
	}
	// load from file
	// add current path
	if slice.IndexOf(config.Paths, ".") == -1 {
		config.Paths = append(config.Paths, ".")
	}
	for _, p := range config.Paths {
		for _, ct := range config.ConfigType {
			parser, ok := supportedParsers[ct]
			if !ok {
				configLogger.Panicf(nil, "Unsupported config type: %s", ct)
			}

			configFilePath, err := filepath.Abs(fmt.Sprintf("%s/%s.%s", p, config.ConfigName, ct))
			if err != nil {
				configLogger.Panicf(nil, "Fail to get config file path, %s", err)
			}

			if !fileutil.IsExist(configFilePath) {
				configLogger.Debugf(nil, "Config file %s not exist", configFilePath)
				continue
			}

			err = config.Koanf.Load(file.Provider(configFilePath), parser)
			if err != nil {
				configLogger.Panicf(nil, "Fail to load config file %s, %s", configFilePath, err)
				return
			}
		}

	}
}

func SetDefault(name string, value any) {
	err := config.Koanf.Load(confmap.Provider(map[string]interface{}{name: value}, "."), nil)
	if err != nil {
		logger.Panic(err)
		return
	}
}

func GetString(key string) string {
	return config.Koanf.String(key)
}

func GetInt(key string) int {
	return config.Koanf.Int(key)
}

func GetBool(key string) bool {
	return config.Koanf.Bool(key)
}

func GetFloat64(key string) float64 {
	return config.Koanf.Float64(key)
}

func GetStringSlice(key string) []string {
	return config.Koanf.Strings(key)
}

func GetStringMapString(key string) map[string]string {
	return config.Koanf.StringMap(key)
}

func GetStringMapStringSlice(key string) map[string][]string {
	return config.Koanf.StringsMap(key)
}

func Get(key string) any {
	return config.Koanf.Get(key)
}

type UnmarshalKeyOpt = func(conf *koanf.UnmarshalConf)

func UnmarshalKey(key string, rawVal any, opts ...UnmarshalKeyOpt) error {
	conf := koanf.UnmarshalConf{
		Tag: "mapstructure",
		DecoderConfig: &mapstructure.DecoderConfig{
			MatchName: func(mapKey, fieldName string) bool {
				return strings.EqualFold(mapKey, fieldName) || strings.EqualFold(strutil.CamelCase(mapKey), fieldName)
			},
			Result:           rawVal,
			WeaklyTypedInput: true,
		},
	}
	for _, opt := range opts {
		opt(&conf)
	}
	return config.Koanf.UnmarshalWithConf(key, rawVal, conf)
}

func GetDefault() *Config {
	return config
}

func Raw() *koanf.Koanf {
	return &config.Koanf
}
