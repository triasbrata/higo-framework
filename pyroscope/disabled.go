package pyroscope

type disabledProfiler struct{}

func (d *disabledProfiler) Stop() error { return nil }
