package pyroscope

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	py "github.com/grafana/pyroscope-go"
	"github.com/triasbrata/higo/instrumentation"
	"go.uber.org/fx"
)

type PyroscopeConfig struct {
	Enabled       bool
	ServerAddress string
}

type PyroscopeConfigProvider interface {
	GetPyroscopeConfig() PyroscopeConfig
}

// Profiler abstracts pyroscope.Profiler so servers don't hard-depend on the concrete type.
type Profiler interface {
	Stop() error
}

type noopProfiler struct{}

func (n *noopProfiler) Stop() error { return nil }

type pyLog struct{}

func (p *pyLog) Debugf(msg string, args ...interface{}) { slog.Debug(fmt.Sprintf(msg, args...)) }
func (p *pyLog) Errorf(msg string, args ...interface{}) { slog.Error(fmt.Sprintf(msg, args...)) }
func (p *pyLog) Infof(msg string, args ...interface{})  { slog.Info(fmt.Sprintf(msg, args...)) }

func LoadPyroscope() fx.Option {
	return fx.Module("pkg/pyroscope", fx.Provide(func(cfProv PyroscopeConfigProvider, insProv instrumentation.InstrumentationProvider) (Profiler, error) {
		cf := cfProv.GetPyroscopeConfig()
		if !cf.Enabled {
			return &noopProfiler{}, nil
		}

		insCfg := insProv.GetInstrumentationConfig()
		runtime.SetMutexProfileFraction(5)
		runtime.SetBlockProfileRate(5)

		hostName, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		p, err := py.Start(py.Config{
			ApplicationName: insCfg.AppName,
			ServerAddress:   cf.ServerAddress,
			Logger:          &pyLog{},
			Tags:            map[string]string{"instance": hostName},
			ProfileTypes: []py.ProfileType{
				py.ProfileCPU,
				py.ProfileInuseObjects,
				py.ProfileAllocObjects,
				py.ProfileInuseSpace,
				py.ProfileAllocSpace,
				py.ProfileGoroutines,
				py.ProfileMutexCount,
				py.ProfileMutexDuration,
				py.ProfileBlockCount,
				py.ProfileBlockDuration,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("error when init pyroscope : %w", err)
		}
		return p, nil
	}))
}
