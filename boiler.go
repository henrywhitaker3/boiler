package boiler

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

type Version string

type Boiler struct {
	ctx       context.Context
	mu        *sync.Mutex
	services  map[string]any
	makers    []maker
	setups    []func(*Boiler) error
	isSetup   bool
	shutMu    *sync.Mutex
	shutdowns []func(b *Boiler) error
	version   Version
	obs       *observer
}

type maker struct {
	name    string
	defered bool
	maker   func(*Boiler) (any, error)
}

func New(ctx context.Context) *Boiler {
	return &Boiler{
		ctx:       ctx,
		mu:        &sync.Mutex{},
		services:  map[string]any{},
		makers:    []maker{},
		setups:    []func(*Boiler) error{},
		shutMu:    &sync.Mutex{},
		shutdowns: []func(b *Boiler) error{},
		obs:       &observer{},
	}
}

// Returns the initial context used to create the boiler instance
func (b *Boiler) Context() context.Context {
	return b.ctx
}

// Set the version number of the application
func (b *Boiler) SetVersion(v Version) {
	b.version = v
}

// Returns the version number of the application
func (b *Boiler) Version() Version {
	return b.version
}

func (b *Boiler) SetLogger(l Logger) {
	b.obs.logger = l
}

// Bootstrap all the services that have been registered
//
// The first time this runs, all of the setups will also run.
func (b *Boiler) Bootstrap() error {
	for _, maker := range b.makers {
		if !maker.defered {
			if _, ok := b.retrieve(maker.name); ok {
				continue
			}
			b.obs.observeBootstrap(maker.name)
			if err := b.make(maker); err != nil {
				return err
			}
		}
	}

	if !b.isSetup {
		for _, f := range b.setups {
			if err := f(b); err != nil {
				return fmt.Errorf("setup func failed: %w", err)
			}
		}
	}
	b.isSetup = true

	return nil
}

func (b *Boiler) make(m maker) error {
	thing, err := m.maker(b)
	if err != nil {
		return fmt.Errorf("%w %s: %w", ErrCouldNotMake, m.name, err)
	}
	b.mu.Lock()
	b.services[m.name] = thing
	b.mu.Unlock()
	return nil
}

func (b *Boiler) MustBootstrap() {
	if err := b.Bootstrap(); err != nil {
		panic(err)
	}
}

// Register a function to be called when the instance is first bootstrapped.
func (b *Boiler) RegisterSetup(f func(b *Boiler) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.setups = append(b.setups, f)
}

// Registers a function to be run when the instances Shutdown() method is called
func (b *Boiler) RegisterShutdown(f func(b *Boiler) error) {
	b.shutMu.Lock()
	defer b.shutMu.Unlock()
	b.shutdowns = append(b.shutdowns, f)
}

func (b *Boiler) Shutdown() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, f := range b.shutdowns {
		if err := f(b); err != nil {
			return err
		}
	}
	return nil
}

func (b *Boiler) findMaker(name string) (maker, bool) {
	for _, m := range b.makers {
		if m.name == name {
			return m, true
		}
	}
	return maker{}, false
}

func (b *Boiler) retrieve(name string) (any, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	svc, ok := b.services[name]
	return svc, ok
}

// Resolve a service from the instance
func (b *Boiler) Resolve[T any]() (T, error) {
	var empty T
	name, err := name[T]()
	if err != nil {
		return empty, err
	}

	b.obs.observeResolve(name)

	svc, ok := b.retrieve(name)
	if !ok {
		maker, ok := b.findMaker(name)
		if ok {
			if err := b.make(maker); err != nil {
				return empty, err
			}
			return b.Resolve[T]()
		}
		return empty, fmt.Errorf("%w: %s", ErrDoesNotExist, name)
	}

	resolved, ok := svc.(T)
	if !ok {
		return empty, ErrWrongType
	}

	return resolved, nil
}

func (b *Boiler) MustResolve[T any]() T {
	resolved, err := b.Resolve[T]()
	if err != nil {
		panic(err)
	}
	return resolved
}

func (b *Boiler) ResolveNamed[T any](name string) (T, error) {
	b.obs.observeResolve(name)

	var empty T
	svc, ok := b.retrieve(name)
	if !ok {
		maker, ok := b.findMaker(name)
		if ok {
			if err := b.make(maker); err != nil {
				return empty, err
			}
			return b.ResolveNamed[T](name)
		}
		return empty, fmt.Errorf("%w: %s", ErrDoesNotExist, name)
	}

	resolved, ok := svc.(T)
	if !ok {
		return empty, ErrWrongType
	}
	return resolved, nil
}

func (b *Boiler) MustResolveNamed[T any](name string) T {
	svc, err := b.ResolveNamed[T](name)
	if err != nil {
		panic(err)
	}
	return svc
}

// Resolve a new instance of the service
func (b *Boiler) Fresh[T any]() (T, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var empty T
	name, err := name[T]()
	if err != nil {
		return empty, err
	}

	maker, ok := b.findMaker(name)
	if !ok {
		return empty, ErrDoesNotExist
	}

	svc, err := maker.maker(b)
	if err != nil {
		return empty, fmt.Errorf("%w %s: %w", ErrCouldNotMake, name, err)
	}

	resolved, ok := svc.(T)
	if !ok {
		return empty, ErrWrongType
	}

	return resolved, nil
}

func (b *Boiler) MustFresh[T any]() T {
	resolved, err := b.Fresh[T]()
	if err != nil {
		panic(err)
	}
	return resolved
}

type Provider[T any] func(*Boiler) (T, error)

// Register a service in the container
func (b *Boiler) Register[T any](p Provider[T]) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	name, err := name[T]()
	if err != nil {
		return fmt.Errorf("generate type name: %w", err)
	}

	b.obs.observeRegister(name)

	if _, ok := b.findMaker(name); ok {
		return fmt.Errorf("%w: %s", ErrAlreadyExists, name)
	}

	b.makers = append(b.makers, maker{
		name: name,
		maker: func(b *Boiler) (any, error) {
			return p(b)
		},
	})

	return nil
}

func (b *Boiler) MustRegister[T any](p Provider[T]) {
	if err := b.Register(p); err != nil {
		panic(err)
	}
}

func (b *Boiler) RegisterNamed[T any](name string, p Provider[T]) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.findMaker(name); ok {
		return fmt.Errorf("%s: %s", ErrAlreadyExists, name)
	}

	b.obs.observeRegister(name)

	b.makers = append(b.makers, maker{
		name: name,
		maker: func(b *Boiler) (any, error) {
			return p(b)
		},
	})

	return nil
}

func (b *Boiler) MustRegisterNamed[T any](name string, p Provider[T]) {
	if err := b.RegisterNamed(name, p); err != nil {
		panic(err)
	}
}

func (b *Boiler) RegisterDeferred[T any](p Provider[T]) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	name, err := name[T]()
	if err != nil {
		return fmt.Errorf("generate type name: %w", err)
	}

	b.obs.observeRegisterDeferred(name)

	if _, ok := b.findMaker(name); ok {
		return fmt.Errorf("%w: %s", ErrAlreadyExists, name)
	}

	b.makers = append(b.makers, maker{
		name:    name,
		defered: true,
		maker: func(b *Boiler) (any, error) {
			return p(b)
		},
	})

	return nil
}

func (b *Boiler) MustRegisterDeferred[T any](p Provider[T]) {
	if err := b.RegisterDeferred(p); err != nil {
		panic(err)
	}
}

func (b *Boiler) RegisterNamedDefered[T any](name string, p Provider[T]) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.findMaker(name); ok {
		return fmt.Errorf("%s: %s", ErrAlreadyExists, name)
	}

	b.obs.observeRegisterDeferred(name)

	b.makers = append(b.makers, maker{
		name:    name,
		defered: true,
		maker: func(b *Boiler) (any, error) {
			return p(b)
		},
	})

	return nil
}

func (b *Boiler) MustRegisterNamedDefered[T any](name string, p Provider[T]) {
	if err := b.RegisterNamedDefered(name, p); err != nil {
		panic(err)
	}
}

func name[T any]() (string, error) {
	typeOf := reflect.TypeFor[T]()
	if typeOf.Name() != "" {
		return fmt.Sprintf("%s/%s", typeOf.PkgPath(), typeOf.Name()), nil
	}

	if typeOf.Kind() == reflect.Pointer {
		typeOfPtr := typeOf.Elem()
		return fmt.Sprintf("*%s.%s", typeOfPtr.PkgPath(), typeOfPtr.Name()), nil
	}

	return "", ErrUnknownType
}
