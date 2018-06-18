package ncerr

import "fmt"

// Option is an Error option function
type Option func(*Error)

func WithError(err error) Option    { return func(e *Error) { e.Message = err.Error() } }
func WithMessage(msg string) Option { return func(e *Error) { e.Message = msg } }
func WithMessageF(format string, args ...interface{}) Option {
	return func(e *Error) { e.Message = fmt.Sprintf(format, args...) }
}
func WithType(t Type) Option { return func(e *Error) { e.Type = t } }
