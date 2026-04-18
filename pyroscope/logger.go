package pyroscope

import (
	"fmt"
	"log/slog"
)

type pyLog struct{}

func (p *pyLog) Debugf(msg string, args ...any) { slog.Debug(fmt.Sprintf(msg, args...)) }
func (p *pyLog) Errorf(msg string, args ...any) { slog.Error(fmt.Sprintf(msg, args...)) }
func (p *pyLog) Infof(msg string, args ...any)  { slog.Info(fmt.Sprintf(msg, args...)) }
