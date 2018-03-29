package ncerr

// Option is an Error option function
type Option func(*Error)

func WithMessage(msg string) Option { return func(e *Error) { e.Message = msg } }
func WithType(t Type) Option        { return func(e *Error) { e.Type = t } }
