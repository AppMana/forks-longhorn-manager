package ns

import "time"

// Windows HostProcess containers already execute in the host's Windows
// namespace. Keep the shared RunFunc contract without emulating Linux setns.
type JoinerDescriptor struct{}

func (j *JoinerDescriptor) Revert() error { return nil }

func (j *JoinerDescriptor) Run(fn func() (interface{}, error)) (interface{}, error) { return fn() }

type JoinerInterface interface {
	Revert() error
	Run(fn func() (interface{}, error)) (interface{}, error)
}

type NewJoinerFunc func(string, time.Duration) (JoinerInterface, error)

var NewJoiner NewJoinerFunc = newJoiner

func newJoiner(string, time.Duration) (JoinerInterface, error) { return &JoinerDescriptor{}, nil }

func RunFunc(fn func() (interface{}, error), _ time.Duration) (interface{}, error) { return fn() }
