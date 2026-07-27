package model

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"github.com/QuantumNous/new-api/common"
	sqlitedriver "github.com/glebarez/go-sqlite"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultSlowThresholdMs = 200
	maxSlowThresholdMs     = 60 * 60 * 1000
)

func newGormConfig(prepareStmt bool) *gorm.Config {
	return &gorm.Config{
		PrepareStmt: prepareStmt,
		Logger:      newGormLogger(),
	}
}

func newGormLogger() logger.Interface {
	return newGormLoggerWithWriter(os.Stdout)
}

func newGormLoggerWithWriter(w io.Writer) logger.Interface {
	slowThresholdMs := common.GetEnvOrDefault("SQL_SLOW_THRESHOLD_MS", defaultSlowThresholdMs)
	if slowThresholdMs < 0 || slowThresholdMs > maxSlowThresholdMs {
		common.SysError(fmt.Sprintf("invalid SQL_SLOW_THRESHOLD_MS %d (allowed 0-%d, 0 disables slow query log), using default %d", slowThresholdMs, maxSlowThresholdMs, defaultSlowThresholdMs))
		slowThresholdMs = defaultSlowThresholdMs
	}
	return &sanitizedGormLogger{Interface: logger.New(log.New(w, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             time.Duration(slowThresholdMs) * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      !common.DebugEnabled,
		Colorful:                  true,
	})}
}

// sanitizedGormLogger 在非 DEBUG 下把数据库驱动错误消息收敛为错误码:
// ParameterizedQueries 只过滤 SQL 字符串,而 MySQL 1062、PG 23505 这类
// 驱动错误消息本身会内联数据值,同样是泄漏面。
type sanitizedGormLogger struct {
	logger.Interface
}

func (l *sanitizedGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &sanitizedGormLogger{Interface: l.Interface.LogMode(level)}
}

// gorm 对配置的 logger 本体做 ParamsFilter 类型断言,包装层必须转发。
func (l *sanitizedGormLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if pf, ok := l.Interface.(gorm.ParamsFilter); ok {
		return pf.ParamsFilter(ctx, sql, params...)
	}
	return sql, params
}

func (l *sanitizedGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	// ErrRecordNotFound 原样透传,内层 logger 依赖 errors.Is 识别并忽略它
	if err != nil && !common.DebugEnabled && !errors.Is(err, gorm.ErrRecordNotFound) {
		err = sanitizeDBError(err)
	}
	l.Interface.Trace(ctx, begin, fc, err)
}

// sanitizeDBError 只收敛四种数据库驱动的错误(消息由数据库服务端生成,可能内联
// 数据值);网络/上下文等其它错误消息不含查询数据,原样保留以便运维排障。
func sanitizeDBError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return fmt.Errorf("mysql error %d", mysqlErr.Number)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("postgres error SQLSTATE %s", pgErr.Code)
	}
	var chErr *proto.Exception
	if errors.As(err, &chErr) {
		return fmt.Errorf("clickhouse error %d", chErr.Code)
	}
	var sqliteErr *sqlitedriver.Error
	if errors.As(err, &sqliteErr) {
		return fmt.Errorf("sqlite error %d", sqliteErr.Code())
	}
	return err
}
