package logger

import (
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger
var Suger *zap.SugaredLogger

// TODO:重构logger
type ZapConfig struct {
	Prefix     string         `yaml:"prefix" mapstructure:"prefix"`
	TimeFormat string         `yaml:"timeFormat" mapstructure:"timeFormat"`
	Level      string         `yaml:"level" mapstructure:"level"`
	Caller     bool           `yaml:"caller" mapstructure:"caller"`
	StackTrace bool           `yaml:"stackTrace" mapstructure:"stackTrace"`
	Writer     string         `yaml:"writer" mapstructure:"writer"`
	Encode     string         `yaml:"encode" mapstructure:"encode"`
	LogFile    *LogFileConfig `yaml:"logFile" mapstructure:"logFile"`
}

type LogFileConfig struct {
	MaxSize  int      `yaml:"maxSize" mapstructure:"maxSize"`
	BackUps  int      `yaml:"backups" mapstructure:"backups"`
	Compress bool     `yaml:"compress" mapstructure:"compress"`
	Output   []string `yaml:"output" mapstructure:"output"`
	Errput   []string `yaml:"errput" mapstructure:"errput"`
}

func InitLogger() {
	config := &ZapConfig{
		Prefix:     "ZapLogTest",
		TimeFormat: "2006/01/02 - 15:04:05",
		Level:      "debug",
		Caller:     true,
		StackTrace: false,
		Writer:     "both",
		Encode:     "console",
		LogFile: &LogFileConfig{
			MaxSize:  20,
			BackUps:  5,
			Compress: true,
			Output:   []string{"./log/output.log"},
			Errput:   []string{},
		},
	}
	// 构建编码器
	encoder := zapEncoder(config)
	// 构建日志级别
	levelEnabler := func() zapcore.Level {
		//TODO:增加生产 info
		return zapcore.DebugLevel
	}()
	// 最后获得Core和Options
	subCore, options := tee(config, encoder, levelEnabler)
	// 创建Logger
	logger := zap.New(subCore, options...)
	Suger = logger.Sugar()
	Logger = logger

}

func zapEncoder(config *ZapConfig) zapcore.Encoder {

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:          "Time",
		LevelKey:         "Level",
		NameKey:          "Logger",
		CallerKey:        "Caller",
		MessageKey:       "Message",
		StacktraceKey:    "StackTrace",
		LineEnding:       zapcore.DefaultLineEnding,
		FunctionKey:      zapcore.OmitKey,
		ConsoleSeparator: "  ",
	}
	// 自定义时间格式
	encoderConfig.EncodeTime = CustomTimeFormatEncoder
	// 日志级别大写
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	// 秒级时间间隔
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	// 简短的调用者输出
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	// 完整的序列化logger名称
	encoderConfig.EncodeName = zapcore.FullNameEncoder
	// 最终的日志编码 json或者console
	switch config.Encode {
	case "json":
		{
			return zapcore.NewJSONEncoder(encoderConfig)
		}
	case "console":
		{
			return zapcore.NewConsoleEncoder(encoderConfig)
		}
	}
	// 默认console
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func tee(cfg *ZapConfig, encoder zapcore.Encoder, levelEnabler zapcore.LevelEnabler) (core zapcore.Core, options []zap.Option) {
	sink := zapWriteSyncer(cfg)
	return zapcore.NewCore(encoder, sink, levelEnabler), buildOptions(cfg, levelEnabler)
}

// 构建Option
func buildOptions(cfg *ZapConfig, levelEnabler zapcore.LevelEnabler) (options []zap.Option) {
	if cfg.Caller {
		options = append(options, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	if cfg.StackTrace {
		options = append(options, zap.AddStacktrace(levelEnabler))
	}
	return
}

// CustomTimeFormatEncoder formats the time for zap logs using the config's TimeFormat.
func CustomTimeFormatEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[Zap]" + " " + t.Format("2006/01/02 - 15:04:05"))
}

func zapWriteSyncer(cfg *ZapConfig) zapcore.WriteSyncer {
	syncers := make([]zapcore.WriteSyncer, 0, 2)

	syncers = append(syncers, zapcore.AddSync(os.Stdout))

	for _, path := range cfg.LogFile.Output {
		logger := &lumberjack.Logger{
			Filename:   path,                 //文件路径
			MaxSize:    cfg.LogFile.MaxSize,  //分割文件的大小
			MaxBackups: cfg.LogFile.BackUps,  //备份次数
			Compress:   cfg.LogFile.Compress, // 是否压缩
			LocalTime:  true,                 //使用本地时间
		}
		syncers = append(syncers, zapcore.Lock(zapcore.AddSync(logger)))
		// }
	}
	return zap.CombineWriteSyncers(syncers...)
}

func Info(template string, args ...any) {
	Suger.Infof(template, args...)
}
func Warn(template string, args ...any) {
	Suger.Warnf(template, args...)
}
func Error(template string, args ...any) {
	Suger.Errorf(template, args...)
}
func Debug(template string, args ...any) {
	Suger.Debugf(template, args...)
}
func Panic(template string, args ...any) {
	Suger.Panicf(template, args...)
}

func Infoln(args ...any) {
	Suger.Infoln(args...)
}
func Warnln(args ...any) {
	Suger.Warnln(args...)
}
func Errorln(args ...any) {
	Suger.Errorln(args...)
}
func Debugln(args ...any) {
	Suger.Debugln(args...)
}
func Panicln(args ...any) {
	Suger.Panicln(args...)
}
