package clog

import "go.opentelemetry.io/otel/log"

func convertLevel(level logLevel) log.Severity {
	switch level {
	case LevelDebug:
		return log.SeverityDebug
	case LevelInfo:
		return log.SeverityInfo
	case LevelError:
		return log.SeverityError
	case LevelDisabled:
		fallthrough
	default:
		return log.SeverityUndefined
	}
}
