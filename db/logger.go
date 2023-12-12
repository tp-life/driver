package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-errors/errors"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type LoggerWrapper struct {
	logger        slog.Logger
	SlowThreshold time.Duration
	LogLevel      gormlogger.LogLevel
}

func NewLoggerWrapper(logger slog.Logger, slowThreshold time.Duration, logLevel gormlogger.LogLevel) *LoggerWrapper {
	return &LoggerWrapper{logger: logger, SlowThreshold: slowThreshold, LogLevel: logLevel}
}

func (l *LoggerWrapper) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

func (l *LoggerWrapper) Info(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.logger.InfoContext(ctx, fmt.Sprintf(s, i...))
	}
}

func (l *LoggerWrapper) Warn(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.logger.WarnContext(ctx, fmt.Sprintf(s, i...))
	}
}

func (l *LoggerWrapper) Error(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.logger.ErrorContext(ctx, fmt.Sprintf(s, i...))
	}
}

func (l *LoggerWrapper) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	latency := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && !errors.Is(err, gormlogger.ErrRecordNotFound):
		sql, rows := fc()
		l.logger.ErrorContext(
			ctx,
			"sql execute error",
			slog.Any("scene", "mysql_client"),
			slog.Any("err", err),
			slog.Any("latency", latency),
			slog.Any("sql", sql),
			slog.Any("line", utils.FileWithLineNum()),
			slog.Any("rows", rows),
		)
	case latency > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		sql, rows := fc()
		l.logger.WarnContext(
			ctx,
			fmt.Sprintf("sql execute slow >= %v", l.SlowThreshold),
			slog.Any("scene", "mysql_client"),
			slog.Any("latency", latency),
			slog.Any("sql", sql),
			slog.Any("line", utils.FileWithLineNum()),
			slog.Any("rows", rows),
		)
	case l.LogLevel == gormlogger.Info:
		sql, rows := fc()
		l.logger.InfoContext(
			ctx,
			"sql execute",
			slog.Any("scene", "mysql_client"),
			slog.Any("latency", latency),
			slog.Any("sql", sql),
			slog.Any("line", utils.FileWithLineNum()),
			slog.Any("rows", rows),
		)
	}
}
