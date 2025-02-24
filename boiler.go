package boiler

import (
	"fmt"
	"reflect"
	"sync"
)

type Boiler struct {
	mu       *sync.Mutex
	services map[string]any
	makers   map[string]maker
}

type maker func(*Boiler) (any, error)

func New() *Boiler {
	return &Boiler{
		mu:       &sync.Mutex{},
		services: map[string]any{},
		makers:   map[string]maker{},
	}
}

func (b *Boiler) Bootstrap() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for name, do := range b.makers {
		thing, err := do(b)
		if err != nil {
			return fmt.Errorf("%w %s: %w", ErrCouldNotMake, name, err)
		}
		b.services[name] = thing
	}

	return nil
}

func (b *Boiler) MustBootstrap() {
	if err := b.Bootstrap(); err != nil {
		panic(err)
	}
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
		return empty, ErrDoesNotExist
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
