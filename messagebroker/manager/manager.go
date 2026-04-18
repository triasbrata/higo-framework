package manager

type ShouldConnectionHave interface {
	Close() error
}
type Manager[T ShouldConnectionHave] interface {
	SetCon(con T)
	GetCon() T
	Ready() <-chan struct{}
	Release() error
}
