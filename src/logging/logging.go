package logging

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"

	color "git.handmade.network/hmn/hmn/src/ansicolor"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.ErrorStackMarshaler = oops.ZerologStackMarshaler
	log.Logger = log.Output(NewPrettyZerologWriter())
	zerolog.SetGlobalLevel(config.Config.LogLevel)
}

func GlobalLogger() *zerolog.Logger {
	return &log.Logger
}

func Trace() *zerolog.Event {
	return log.Trace().Timestamp().Stack()
}

func Debug() *zerolog.Event {
	return log.Debug().Timestamp().Stack()
}

func Info() *zerolog.Event {
	return log.Info().Timestamp().Stack()
}

func Warn() *zerolog.Event {
	return log.Warn().Timestamp().Stack()
}

func Error() *zerolog.Event {
	return log.Error().Timestamp().Stack()
}

func Panic() *zerolog.Event {
	return log.Panic().Timestamp().Stack()
}

func Fatal() *zerolog.Event {
	return log.Fatal().Timestamp().Stack()
}

func With() zerolog.Context {
	return log.With().Stack()
}

type PrettyZerologWriter struct {
	wd                  string
	wasLastLogMultiline bool
}

type PrettyLogEntry struct {
	Timestamp  string
	Level      string
	Message    string
	Error      string
	StackTrace []interface{}

	OtherFields []PrettyField
}

type PrettyField struct {
	Name  string
	Value interface{}
}

var ColorFromLevel = map[string]string{
	"trace": color.Gray,
	"debug": color.Gray,
	"info":  color.BgBlue,
	"warn":  color.BgYellow,
	"error": color.BgRed,
	"fatal": color.BgRed,
	"panic": color.BgRed,
}

func NewPrettyZerologWriter() *PrettyZerologWriter {
	wd, _ := os.Getwd()
	return &PrettyZerologWriter{
		wd:                  wd,
		wasLastLogMultiline: false,
	}
}

func (w *PrettyZerologWriter) Write(p []byte) (int, error) {
	// TODO: panic recovery so we log _something_

	var fields map[string]interface{}
	err := json.Unmarshal(p, &fields)
	if err != nil {
		return os.Stderr.Write(p)
	}

	var pretty PrettyLogEntry
	for name, val := range fields {
		switch name {
		case zerolog.TimestampFieldName:
			pretty.Timestamp = val.(string)
		case zerolog.LevelFieldName:
			pretty.Level = val.(string)
		case zerolog.MessageFieldName:
			pretty.Message = val.(string)
		case zerolog.ErrorFieldName:
			pretty.Error = val.(string)
		case zerolog.ErrorStackFieldName:
			pretty.StackTrace = val.([]interface{})
		default:
			pretty.OtherFields = append(pretty.OtherFields, PrettyField{
				Name:  name,
				Value: val,
			})
		}
	}

	sort.Slice(pretty.OtherFields, func(i, j int) bool {
		return strings.Compare(pretty.OtherFields[i].Name, pretty.OtherFields[j].Name) < 0
	})

	isMultiline := (pretty.Error != "" || pretty.StackTrace != nil || pretty.OtherFields != nil)

	var b strings.Builder
	if isMultiline || w.wasLastLogMultiline {
		b.WriteString("---------------------------------------\n")
	}
	b.WriteString(pretty.Timestamp)
	b.WriteString(" ")
	if pretty.Level != "" {
		b.WriteString(ColorFromLevel[pretty.Level])
		b.WriteString(color.Bold)
		b.WriteString(strings.ToUpper(pretty.Level))
		b.WriteString(color.Reset)
		b.WriteString(": ")
	}
	b.WriteString(pretty.Message)
	b.WriteString("\n")
	if pretty.Error != "" {
		b.WriteString("  " + color.Bold + color.Red + "ERROR:" + color.Reset + " ")
		b.WriteString(pretty.Error)
		b.WriteString("\n")
	}
	if len(pretty.OtherFields) > 0 {
		b.WriteString("  " + color.Bold + color.Blue + "Fields:" + color.Reset + "\n")
		for _, field := range pretty.OtherFields {
			valuePretty, _ := json.MarshalIndent(field.Value, "    ", "  ")
			b.WriteString("    ")
			b.WriteString(field.Name)
			b.WriteString(": ")
			b.WriteString(string(valuePretty))
			b.WriteString("\n")
		}
	}
	if pretty.StackTrace != nil {
		b.WriteString("  " + color.Bold + color.Blue + "Stack trace:" + color.Reset + "\n")
		for _, frame := range pretty.StackTrace {
			frameMap := frame.(map[string]interface{})
			file := frameMap["file"].(string)
			file = strings.Replace(file, w.wd, ".", 1)

			b.WriteString("    ")
			b.WriteString(frameMap["function"].(string))
			b.WriteString(" (")
			b.WriteString(file)
			b.WriteString(":")
			b.WriteString(strconv.Itoa(int(frameMap["line"].(float64))))
			b.WriteString(")\n")
		}
	}

	w.wasLastLogMultiline = isMultiline

	return os.Stderr.Write([]byte(b.String()))
}

func LogPanics(logger *zerolog.Logger) {
	if r := recover(); r != nil {
		LogPanicValue(logger, r, "recovered from panic")
	}
}

func LogPanicValue(logger *zerolog.Logger, val interface{}, msg string) {
	if logger == nil {
		logger = GlobalLogger()
	}

	if err, ok := val.(error); ok {
		l := logger.Error().Err(err)
		if _, ok := err.(*oops.Error); !ok {
			l = l.Interface(zerolog.ErrorStackFieldName, oops.Trace())
		}
		l.Msg(msg)
	} else {
		logger.Error().
			Interface("recovered", val).
			Interface(zerolog.ErrorStackFieldName, oops.Trace()).
			Msg(msg)
	}
}
