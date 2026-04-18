package pyroscope

import (
	"fmt"
	"os"
	"runtime"

	py "github.com/grafana/pyroscope-go"
	"github.com/triasbrata/higo-framework/instrumentation"
	"go.uber.org/fx"
)

func LoadDisabledProfiler() fx.Option {
	return fx.Module("pkg/pyroscope", fx.Provide(func() Profiler {
		return &disabledProfiler{}
	}))
}

func LoadPyroscope() fx.Option {
	return fx.Module("pkg/pyroscope", fx.Provide(func(cfProv PyroscopeConfigProvider, insProv instrumentation.InstrumentationProvider) (Profiler, error) {
		cf := cfProv.GetPyroscopeConfig()
		if !cf.Enabled {
			return &disabledProfiler{}, nil
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
