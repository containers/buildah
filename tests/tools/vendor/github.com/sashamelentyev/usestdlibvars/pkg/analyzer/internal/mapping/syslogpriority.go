//go:build !windows && !plan9

package mapping

import (
	"log/syslog"
	"strconv"
)

func init() {
	// SyslogPriority[strconv.Itoa(int(syslog.LOG_EMERG))] = "syslog.LOG_EMERG"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_ALERT))] = "syslog.LOG_ALERT"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_CRIT))] = "syslog.LOG_CRIT"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_ERR))] = "syslog.LOG_ERR"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_WARNING))] = "syslog.LOG_WARNING"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_NOTICE))] = "syslog.LOG_NOTICE"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_INFO))] = "syslog.LOG_INFO"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_DEBUG))] = "syslog.LOG_DEBUG"

	// SyslogPriority[strconv.Itoa(int(syslog.LOG_KERN))] = "syslog.LOG_KERN"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_USER))] = "syslog.LOG_USER"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_MAIL))] = "syslog.LOG_MAIL"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_DAEMON))] = "syslog.LOG_DAEMON"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_AUTH))] = "syslog.LOG_AUTH"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_SYSLOG))] = "syslog.LOG_SYSLOG"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LPR))] = "syslog.LOG_LPR"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_NEWS))] = "syslog.LOG_NEWS"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_UUCP))] = "syslog.LOG_UUCP"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_CRON))] = "syslog.LOG_CRON"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_AUTHPRIV))] = "syslog.LOG_AUTHPRIV"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_FTP))] = "syslog.LOG_FTP"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LOCAL0))] = "syslog.LOG_LOCAL0"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LOCAL1))] = "syslog.LOG_LOCAL1"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LOCAL2))] = "syslog.LOG_LOCAL2"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LOCAL3))] = "syslog.LOG_LOCAL3"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LOCAL4))] = "syslog.LOG_LOCAL4"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LOCAL5))] = "syslog.LOG_LOCAL5"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LOCAL6))] = "syslog.LOG_LOCAL6"
	SyslogPriority[strconv.Itoa(int(syslog.LOG_LOCAL7))] = "syslog.LOG_LOCAL7"
}
