# Boiler

[![Test](https://github.com/henrywhitaker3/boiler/actions/workflows/test.yaml/badge.svg)](https://github.com/henrywhitaker3/boiler/actions/workflows/test.yaml)

Boiler is a small, type-safe dependency-injection container for Go. Register constructors, bootstrap the container, and resolve dependencies directly from the container with Go 1.27 generic methods.

## Requirements

Boiler requires **Go 1.27+**. Its receiver-based generic API relies on generic methods, introduced in Go 1.27.

## Upgrading from v1

This is a breaking major release. Update your dependency and imports to include the `/v2` major-version suffix:

```go
import "github.com/henrywhitaker3/boiler/v2"
```

The generic helpers are now methods on `*boiler.Boiler`; for example, replace `boiler.Resolve[Service](b)` with `b.Resolve[Service]()`. See the quick start below for the new API shape.

## Install

```sh
go get github.com/henrywhitaker3/boiler/v2
```

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/henrywhitaker3/boiler/v2"
)

type Config struct {
	Greeting string
}

type Greeter struct {
	config Config
}

func main() {
	b := boiler.New(context.Background())

	b.MustRegister(func(*boiler.Boiler) (Config, error) {
		return Config{Greeting: "Hello, world!"}, nil
	})

	b.MustRegister(func(b *boiler.Boiler) (Greeter, error) {
		config, err := b.Resolve[Config]()
		if err != nil {
			return Greeter{}, err
		}
		return Greeter{config: config}, nil
	})

	b.MustBootstrap()

	greeter := b.MustResolve[Greeter]()
	fmt.Println(greeter.config.Greeting)
}
```

Services are identified by their concrete Go type. A provider can resolve other registered services, as shown above.

## Service lifetime

`Register` creates a singleton service. Its provider runs during `Bootstrap`, and subsequent calls to `Resolve` return that stored value.

Use `RegisterDeferred` to postpone construction until the service is first resolved. Use `Fresh` when you need a new instance from a registered provider without replacing the stored singleton.

```go
b.MustRegisterDeferred(func(*boiler.Boiler) (*Client, error) {
	return newClient(), nil
})

// Constructs and stores the singleton on first use.
client := b.MustResolve[*Client]()

// Constructs a separate instance each time.
anotherClient := b.MustFresh[*Client]()
```

## Named services

When multiple values share a type, register and resolve them by name.

```go
b.MustRegisterNamed("primary", func(*boiler.Boiler) (DB, error) {
	return openPrimaryDB(), nil
})
b.MustRegisterNamed("replica", func(*boiler.Boiler) (DB, error) {
	return openReplicaDB(), nil
})

primary := b.MustResolveNamed[DB]("primary")
replica := b.MustResolveNamed[DB]("replica")
```

`RegisterNamedDefered` provides the named deferred equivalent. The spelling is retained for API compatibility.

## Lifecycle hooks

Setup functions run once, after all non-deferred services have been bootstrapped. Shutdown functions run when `Shutdown` is called.

```go
b.RegisterSetup(func(*boiler.Boiler) error {
	return nil
})

b.RegisterShutdown(func(*boiler.Boiler) error {
	return nil
})

if err := b.Bootstrap(); err != nil {
	panic(err)
}
defer func() {
	if err := b.Shutdown(); err != nil {
		panic(err)
	}
}()
```

## API at a glance

| Need | Method |
| --- | --- |
| Register a singleton | `Register` / `MustRegister` |
| Register a lazy singleton | `RegisterDeferred` / `MustRegisterDeferred` |
| Register by name | `RegisterNamed` / `MustRegisterNamed` |
| Resolve a singleton | `Resolve[T]()` / `MustResolve[T]()` |
| Resolve by name | `ResolveNamed[T](name)` / `MustResolveNamed[T](name)` |
| Create an untracked instance | `Fresh[T]()` / `MustFresh[T]()` |

The `Must*` variants panic when an operation fails. Prefer the error-returning methods at application boundaries where errors should be handled or reported.
