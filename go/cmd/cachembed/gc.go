package cachembed

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yammerjp/cachembed/internal/storage"
)

func runGarbageCollection(cmd GCCmd, dsn string) {
	duration, err := parseDuration(cmd.Before)
	if err != nil {
		slog.Error("invalid duration format", "error", err, "value", cmd.Before)
		os.Exit(1)
	}

	db, err := storage.NewDB(dsn)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 最大IDを取得（EndIDが指定されていない場合に使用）
	maxID, err := db.GetMaxID()
	if err != nil {
		slog.Error("failed to get max ID", "error", err)
		os.Exit(1)
	}

	// EndIDが0（未指定）の場合は最大IDを使用
	endID := cmd.EndID
	if endID == 0 {
		endID = maxID
	}

	// GC実行
	if err := db.DeleteEntriesBeforeWithSleep(duration, cmd.StartID, endID, int64(cmd.Batch), time.Duration(cmd.Sleep)*time.Second); err != nil {
		slog.Error("failed to run garbage collection", "error", err)
		os.Exit(1)
	}

	slog.Info("garbage collection completed successfully")
}

// parseDuration は "24h", "7d", "30d" のような文字列をtime.Durationに変換します
func parseDuration(s string) (time.Duration, error) {
	// 日単位の指定をサポート
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid day format: %w", err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// 時間単位の指定
	return time.ParseDuration(s)
}
