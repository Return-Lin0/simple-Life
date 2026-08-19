// Package timeutil 提供全局统一时区（Asia/Hong_Kong）与时间格式转换工具。
// 约定：接口传输格式 YYYY-MM-DD HH:mm:ss，日期 YYYY-MM-DD，时间 HH:mm:ss。
package timeutil

import (
	"time"
)

// Loc 是全局统一时区，避免服务器默认时区不一致导致提醒时间错乱。
var Loc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Hong_Kong")
	if err != nil {
		// 极端环境下回退到本地时区，保证进程可启动
		return time.Local
	}
	return loc
}()

// DateLayout / TimeLayout / DateTimeLayout 为统一格式常量。
const (
	DateLayout     = "2006-01-02"
	TimeLayout     = "15:04:05"
	DateTimeLayout = "2006-01-02 15:04:05"
)

// Now 返回当前时间（Asia/Hong_Kong 时区）。
func Now() time.Time {
	return time.Now().In(Loc)
}

// ParseDate 解析 YYYY-MM-DD。
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation(DateLayout, s, Loc)
}

// FormatDate 格式化日期。
func FormatDate(t time.Time) string {
	return t.In(Loc).Format(DateLayout)
}

// ParseClock 解析 HH:mm:ss。
func ParseClock(s string) (time.Time, error) {
	return time.ParseInLocation(TimeLayout, s, Loc)
}

// FormatDateTime 格式化完整时间。
func FormatDateTime(t time.Time) string {
	return t.In(Loc).Format(DateTimeLayout)
}

// ParseDateTime 解析完整时间。
func ParseDateTime(s string) (time.Time, error) {
	return time.ParseInLocation(DateTimeLayout, s, Loc)
}

// CombineDateAndTime 将日期与时间合成完整时刻。
func CombineDateAndTime(date, clock string) (time.Time, error) {
	d, err := ParseDate(date)
	if err != nil {
		return time.Time{}, err
	}
	t, err := ParseClock(clock)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), 0, Loc), nil
}

// StartOfDay 返回某日 00:00:00。
func StartOfDay(t time.Time) time.Time {
	nt := t.In(Loc)
	return time.Date(nt.Year(), nt.Month(), nt.Day(), 0, 0, 0, 0, Loc)
}

// DaysBetween 计算 end - start 的天数（按日期差）。
func DaysBetween(start, end time.Time) int {
	s := StartOfDay(start)
	e := StartOfDay(end)
	return int(e.Sub(s).Hours() / 24)
}
