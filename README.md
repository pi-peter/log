# Structured Logging for Humans

[![go.dev][pkg-img]][pkg] [![goreport][report-img]][report] [![coverage][cov-img]][cov]

## Features

* No Dependencies
* Intuitive Interfaces
* JSON/TSV/Printf Loggers
* Rotating File Writer
* Pretty Console Writer
* Dynamic Log Level
* Contextual Fields
* High Performance

## Interfaces

### Logger
```go
// DefaultLogger is the global logger.
var DefaultLogger = Logger{
	Level:      DebugLevel,
	Caller:     0,
	TimeField:  "",
	TimeFormat: "",
	Writer:     os.Stderr,
}

// A Logger represents an active logging object that generates lines of JSON output to an io.Writer.
type Logger struct {
	// Level defines log levels.
	Level Level

	// Caller determines if adds the file:line of the "caller" key.
	Caller int

	// TimeField defines the time filed name in output.  It uses "time" in if empty.
	TimeField string

	// TimeFormat specifies the time format in output. It uses time.RFC3389 in if empty.
	// If set with `TimeFormatUnix`, `TimeFormatUnixMs`, times are formated as UNIX timestamp.
	TimeFormat string

	// Writer specifies the writer of output. It uses os.Stderr in if empty.
	Writer io.Writer
}
```

### FileWriter & ConsoleWriter
```go
// FileWriter is an io.WriteCloser that writes to the specified filename.
type FileWriter struct {
	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.
	Filename string

	// FileMode represents the file's mode and permission bits.  The default
	// mode is 0644
	FileMode os.FileMode

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated.
	MaxSize int64

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files
	MaxBackups int

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool

	// HostName determines if the hostname used for formatting in backup files.
	HostName bool
}

// ConsoleWriter parses the JSON input and writes it in an
// (optionally) colorized, human-friendly format to os.Stderr
type ConsoleWriter struct {
	// ColorOutput determines if used colorized output.
	ColorOutput bool

	// QuoteString determines if quoting string values.
	QuoteString bool

	// EndWithMessage determines if output message in the end.
	EndWithMessage bool
}
```

## Getting Started

### Simple Logging Example

A out of box example. [![playground][play-simple-img]][play-simple]
```go
package main

import (
	"github.com/phuslu/log"
)

func main() {
	log.Printf("Hello, %s", "世界")
	log.Info().Str("foo", "bar").Int("number", 42).Msg("a structured logger")
}

// Output:
//   {"time":"2020-03-22T09:58:41.828Z","message":"Hello, 世界"}
//   {"time":"2020-03-22T09:58:41.828Z","level":"info","foo":"bar","number":42,"message":"a structure logger"}
```
> Note: By default log writes to `os.Stderr`

### Customize the configuration and formatting:

To customize logger filed name and format. [![playground][play-customize-img]][play-customize]
```go
log.DefaultLogger = log.Logger{
	Level:      log.InfoLevel,
	Caller:     1,
	TimeField:  "date",
	TimeFormat: "2006-01-02",
	Writer:     os.Stderr,
}
log.Info().Str("foo", "bar").Msgf("hello %s", "world")

// Output: {"date":"2019-07-04","level":"info","caller":"test.go:42","foo":"bar","message":"hello world"}
```

### Rotating File Writer

```go
package main

import (
	"time"

	"github.com/phuslu/log"
	"github.com/robfig/cron/v3"
)

func main() {
	logger := log.Logger{
		Level:      log.ParseLevel("info"),
		Writer:     &log.FileWriter{
			Filename:   "main.log",
			FileMode:   0600,
			MaxSize:    50*1024*1024,
			MaxBackups: 7,
			LocalTime:  false,
		},
	}

	runner := cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))
	runner.AddFunc("0 0 * * * *", func() { logger.Writer.(*log.FileWriter).Rotate() })
	go runner.Run()

	for {
		time.Sleep(time.Second)
		logger.Info().Msg("hello world")
	}
}
```

### Pretty Console Writer

To log a human-friendly, colorized output, use `log.ConsoleWriter`. [![playground][play-pretty-img]][play-pretty]

```go
if log.IsTerminal(os.Stderr.Fd()) {
	log.DefaultLogger = log.Logger{
		Caller: 1,
		Writer: &log.ConsoleWriter{ColorOutput: true},
	}
}

log.Printf("a printf style line")
log.Info().Err(errors.New("an error")).Int("everything", 42).Str("foo", "bar").Msg("hello world")
```
![Pretty logging][pretty-logging-img]
> Note: pretty logging also works on windows console

### Dynamic log Level

To change log level on the fly, use `log.DefaultLogger.SetLevel`. [![playground][play-dynamic-img]][play-dynamic]

```go
log.DefaultLogger.SetLevel(log.InfoLevel)
log.Debug().Msg("debug log")
log.Info().Msg("info log")
log.Warn().Msg("warn log")
log.DefaultLogger.SetLevel(log.DebugLevel)
log.Debug().Msg("debug log")
log.Info().Msg("info log")

// Output:
//   {"time":"2020-03-24T05:06:54.674Z","level":"info","message":"info log"}
//   {"time":"2020-03-24T05:06:54.674Z","level":"warn","message":"warn log"}
//   {"time":"2020-03-24T05:06:54.675Z","level":"debug","message":"debug log"}
//   {"time":"2020-03-24T05:06:54.675Z","level":"info","message":"info log"}
```

### Contextual Fields

To add preserved `key:value` pairs to each event, use `log.NewContext()`. [![playground][play-context-img]][play-context]

```go
ctx := log.NewContext().Str("ctx_str", "a ctx str").Value()

logger := log.Logger{Level: log.InfoLevel}
logger.Debug().Context(ctx).Int("no0", 0).Msg("zero")
logger.Info().Context(ctx).Int("no1", 1).Msg("first")
logger.Info().Context(ctx).Int("no2", 2).Msg("second")

// Output:
//   {"time":"2020-07-12T05:03:43.949Z","level":"info","ctx_str":"a ctx str","no1":1,"message":"first"}
//   {"time":"2020-07-12T05:03:43.949Z","level":"info","ctx_str":"a ctx str","no2":2,"message":"second"}
```

### Logging to syslog

```go
package main

import (
	"log/syslog"

	"github.com/phuslu/log"
)

func main() {
	logger, err := syslog.NewLogger(syslog.LOG_INFO, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("new syslog error")
	}

	log.DefaultLogger.Writer = logger.Writer()
	log.Info().Str("foo", "bar").Msg("a syslog message")
}
```

### High Performance

A quick and simple benchmark with zap/zerolog/onelog

```go
// go test -v -run=none -bench=. -benchtime=10s -benchmem log_test.go
package main

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/francoispqt/onelog"
	"github.com/phuslu/log"
	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var fakeMessage = "Test logging, but use a somewhat realistic message length. "

func BenchmarkZap(b *testing.B) {
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(ioutil.Discard),
		zapcore.InfoLevel,
	))
	for i := 0; i < b.N; i++ {
		logger.Info(fakeMessage, zap.String("foo", "bar"), zap.Int("int", 123))
	}
}

func BenchmarkOneLog(b *testing.B) {
	logger := onelog.New(ioutil.Discard, onelog.INFO)
	logger.Hook(func(e onelog.Entry) { e.Int64("time", time.Now().Unix()) })
	for i := 0; i < b.N; i++ {
		logger.InfoWith(fakeMessage).String("foo", "bar").Int("int", 123).Write()
	}
}

func BenchmarkZeroLog(b *testing.B) {
	logger := zerolog.New(ioutil.Discard).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	for i := 0; i < b.N; i++ {
		logger.Info().Str("foo", "bar").Int("int", 123).Msg(fakeMessage)
	}
}

func BenchmarkPhusLog(b *testing.B) {
	logger := log.Logger{
		TimeFormat: log.TimeFormatUnix,
		Writer:     ioutil.Discard,
	}
	for i := 0; i < b.N; i++ {
		logger.Info().Str("foo", "bar").Int("int", 123).Msg(fakeMessage)
	}
}
```
Performance results on my laptop:
```
BenchmarkZap-16        	14383234	       828 ns/op	     128 B/op	       1 allocs/op
BenchmarkOneLog-16     	42118165	       285 ns/op	       0 B/op	       0 allocs/op
BenchmarkZeroLog-16    	46848562	       252 ns/op	       0 B/op	       0 allocs/op
BenchmarkPhusLog-16    	78545636	       155 ns/op	       0 B/op	       0 allocs/op
```

[pkg-img]: https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square
[pkg]: https://pkg.go.dev/github.com/phuslu/log?tab=doc
[report-img]: https://goreportcard.com/badge/github.com/phuslu/log
[report]: https://goreportcard.com/report/github.com/phuslu/log
[cov-img]: http://gocover.io/_badge/github.com/phuslu/log
[cov]: https://gocover.io/github.com/phuslu/log
[pretty-logging-img]: https://user-images.githubusercontent.com/195836/77247067-5cf24000-6c68-11ea-9e65-6cdc00d82384.png
[play-simple-img]: https://img.shields.io/badge/playground-NGV25aBKmYH-29BEB0?style=flat&logo=go
[play-simple]: https://play.golang.org/p/NGV25aBKmYH
[play-customize-img]: https://img.shields.io/badge/playground-EaFFre1DUVJ-29BEB0?style=flat&logo=go
[play-customize]: https://play.golang.org/p/EaFFre1DUVJ
[play-pretty-img]: https://img.shields.io/badge/playground-62bWGk67apR-29BEB0?style=flat&logo=go
[play-pretty]: https://play.golang.org/p/62bWGk67apR
[play-dynamic-img]: https://img.shields.io/badge/playground-0S--JT7h--QXI-29BEB0?style=flat&logo=go
[play-dynamic]: https://play.golang.org/p/0S-JT7h-QXI
[play-context-img]: https://img.shields.io/badge/playground-ttnMKCLSjyw-29BEB0?style=flat&logo=go
[play-context]: https://play.golang.org/p/ttnMKCLSjyw
