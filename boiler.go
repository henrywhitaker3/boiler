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
	makers    map[string]maker
	setups    []func(*Boiler) error
	isSetup   bool
	shutMu    *sync.Mutex
	shutdowns []func(b *Boiler) error
	version   Version
}

type maker func(*Boiler) (any, error)

func New(ctx context.Context) *Boiler {
	return &Boiler{
		ctx:       ctx,
		mu:        &sync.Mutex{},
		services:  map[string]any{},
		makers:    map[string]maker{},
		setups:    []func(*Boiler) error{},
		shutMu:    &sync.Mutex{},
		shutdowns: []func(b *Boiler) error{},
	}
}

func (b *Boiler) Context() context.Context {
	return b.ctx
}

func (b *Boiler) SetVersion(v Version) {
	b.version = v
}

func (b *Boiler) Version() Version {
	return b.version
}

func (b *Boiler) Bootstrap() error {
	for name, do := range b.makers {
		if _, ok := b.services[name]; ok {
			continue
		}
		thing, err := do(b)
		if err != nil {
			return fmt.Errorf("%w %s: %w", ErrCouldNotMake, name, err)
		}
		b.mu.Lock()
		b.services[name] = thing
		b.mu.Unlock()
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

func (b *Boiler) MustBootstrap() {
	if err := b.Bootstrap(); err != nil {
		panic(err)
	}
}

func (b *Boiler) RegisterSetup(f func(b *Boiler) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.setups = append(b.setups, f)
}

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

func Resolve[T any](b *Boiler) (T, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var empty T
	name, err := name[T]()
	if err != nil {
		return empty, err
	}

	svc, ok := b.services[name]
	if !ok {
		return empty, fmt.Errorf("%w: %s", ErrDoesNotExist, name)
	}

	resolved, ok := svc.(T)
	if !ok {
		return empty, ErrWrongType
	}

	return resolved, nil
}

func MustResolve[T any](b *Boiler) T {
	resolved, err := Resolve[T](b)
	if err != nil {
		panic(err)
	}
	return resolved
}

func Fresh[T any](b *Boiler) (T, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var empty T
	name, err := name[T]()
	if err != nil {
		return empty, err
	}

	maker, ok := b.makers[name]
	if !ok {
		return empty, ErrDoesNotExist
	}

	svc, err := maker(b)
	if err != nil {
		return empty, fmt.Errorf("%w %s: %w", ErrCouldNotMake, name, err)
	}

	resolved, ok := svc.(T)
	if !ok {
		return empty, ErrWrongType
	}

	return resolved, nil
}

func MustFresh[T any](b *Boiler) T {
	resolved, err := Fresh[T](b)
	if err != nil {
		panic(err)
	}
	return resolved
}

type Provider[T any] func(*Boiler) (T, error)

func Register[T any](b *Boiler, p Provider[T]) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	name, err := name[T]()
	if err != nil {
		return fmt.Errorf("generate type name: %w", err)
	}

	if _, ok := b.makers[name]; ok {
		return ErrAlreadyExists
	}

	b.makers[name] = func(b *Boiler) (any, error) {
		return p(b)
	}

	return nil
}

func MustRegister[T any](b *Boiler, p Provider[T]) {
	if err := Register[T](b, p); err != nil {
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
