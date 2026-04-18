package pyroscope

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
