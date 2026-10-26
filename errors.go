package saga

import (
	"errors"
	"fmt"
)

var (
	ErrorRootEmpty = errors.New("root is empty")
	ErrorNoneKind  = errors.New("unsupported node kind")
	ErrorNoneMode  = errors.New("unsupported node mode")

	ErrorNodeNoExecuteFunc = func(name string) error { return fmt.Errorf("step %q has no execute function", name) }
	ErrorNodeExecutedFunc  = func(name string, err error) error { return fmt.Errorf("step %q failed: %w", name, err) }
)

var (
	ErrorSequenceType = errors.New("sequence error")
	ErrorParallelType = errors.New("parallel error")
)

var (
	ErrorCompensation = errors.New("compensation error")

	ErrorCompensationActionFunc = func(name string, err error) error { return fmt.Errorf("%w for %s: %w", ErrorCompensation, name, err) }
)

var (
	ErrorRegistryNil          = errors.New("registry is nil")
	ErrorRegistryName         = errors.New("registry name is empty")
	ErrorRegistryNilFunc      = errors.New("registry function is nil")
	ErrorRegistryDuplicate    = errors.New("function already registered")
	ErrorRegistryNotFound     = errors.New("function not registered")
	ErrorRegistryTypeMismatch = errors.New("registered function type mismatch")
)
