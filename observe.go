package boiler

type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
}

type observer struct {
	logger Logger
}

func newObserver(l Logger) *observer {
	return &observer{
		logger: l,
	}
}

func (o *observer) observeBootstrap(kind string) {
	if o.logger != nil {
		o.logger.Info("bootstrapping service", "idnetifier", kind)
	}
}

func (o *observer) observeRegister(kind string) {
	if o.logger != nil {
		o.logger.Debug("registering service", "idnetifier", kind)
	}
}

func (o *observer) observeRegisterDeferred(kind string) {
	if o.logger != nil {
		o.logger.Debug("registering deferred service", "idnetifier", kind)
	}
}

func (o *observer) observeResolve(kind string) {
	if o.logger != nil {
		o.logger.Debug("resolving service", "idnetifier", kind)
	}
}
