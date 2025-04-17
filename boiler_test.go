package boiler

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type Demo struct {
	value string
}

func TestItStoresValues(t *testing.T) {
	b := New(context.Background())

	err := Register(b, func(b *Boiler) (Demo, error) {
		return Demo{
			value: "apple",
		}, nil
	})
	require.Nil(t, err)

	require.Nil(t, b.Bootstrap())

	resolved, err := Resolve[Demo](b)
	require.Nil(t, err)
	require.Equal(t, "apple", resolved.value)
}

func TestItStoresPointers(t *testing.T) {
	b := New(context.Background())

	initial := "bongo"
	err := Register(b, func(b *Boiler) (*Demo, error) {
		return &Demo{
			value: initial,
		}, nil
	})
	require.Nil(t, err)
	require.Nil(t, b.Bootstrap())

	initial = "fish"
	resolved, err := Resolve[*Demo](b)
	require.Nil(t, err)
	require.Equal(t, "bongo", resolved.value)
}

func TestItMakesFreshItems(t *testing.T) {
	b := New(context.Background())

	initial := "bongo"
	err := Register(b, func(b *Boiler) (*Demo, error) {
		return &Demo{
			value: initial,
		}, nil
	})
	require.Nil(t, err)
	require.Nil(t, b.Bootstrap())

	initial = "fish"
	resolved, err := Resolve[*Demo](b)
	require.Nil(t, err)
	require.Equal(t, "bongo", resolved.value)
	fresh, err := Fresh[*Demo](b)
	require.Nil(t, err)
	require.Equal(t, "fish", fresh.value)
}

func TestItErrorsWhenResolvingUnknownType(t *testing.T) {
	b := New(context.Background())

	require.Nil(t, b.Bootstrap())

	_, err := Resolve[Demo](b)
	require.ErrorIs(t, err, ErrDoesNotExist)
}

func TestItErrorsWhenFreshingUnknownType(t *testing.T) {
	b := New(context.Background())

	require.Nil(t, b.Bootstrap())

	_, err := Fresh[Demo](b)
	require.ErrorIs(t, err, ErrDoesNotExist)
}

func TestItRegistersNamedServices(t *testing.T) {
	b := New(context.Background())

	require.Nil(t, RegisterNamed(b, "bongo", func(*Boiler) (Demo, error) {
		return Demo{value: "bongo"}, nil
	}))
	require.Nil(t, RegisterNamed(b, "orange", func(*Boiler) (Demo, error) {
		return Demo{value: "orange"}, nil
	}))

	require.Nil(t, b.Bootstrap())

	bongo, err := ResolveNamed[Demo](b, "bongo")
	require.Nil(t, err)
	require.Equal(t, "bongo", bongo.value)
	orange, err := ResolveNamed[Demo](b, "orange")
	require.Nil(t, err)
	require.Equal(t, "orange", orange.value)
}

func TestItRegistersDeferredNamedServices(t *testing.T) {
	b := New(context.Background())

	require.Nil(t, RegisterNamedDefered(b, "bongo", func(*Boiler) (Demo, error) {
		return Demo{value: "bongo"}, nil
	}))
	require.Nil(t, RegisterNamedDefered(b, "orange", func(*Boiler) (Demo, error) {
		return Demo{value: "orange"}, nil
	}))

	require.Nil(t, b.Bootstrap())

	bongo, err := ResolveNamed[Demo](b, "bongo")
	require.Nil(t, err)
	require.Equal(t, "bongo", bongo.value)
	orange, err := ResolveNamed[Demo](b, "orange")
	require.Nil(t, err)
	require.Equal(t, "orange", orange.value)
}

func TestItRegistersDeferedServices(t *testing.T) {
	b := New(context.Background())

	called := false
	require.Nil(t, RegisterDeferred(b, func(*Boiler) (Demo, error) {
		called = true
		return Demo{value: "bongo"}, nil
	}))

	require.Nil(t, b.Bootstrap())
	require.False(t, called)

	_, err := Resolve[Demo](b)
	require.Nil(t, err)
	require.True(t, called)
}

func TestItReolvesWithoutDeadlocks(t *testing.T) {
	b := New(context.Background())

	require.Nil(t, Register(b, func(*Boiler) (Demo, error) {
		return Demo{}, nil
	}))
	require.Nil(t, Register(b, func(b *Boiler) (*http.Server, error) {
		_, err := Resolve[Demo](b)
		if err != nil {
			return nil, err
		}
		return &http.Server{}, nil
	}))
	require.Nil(t, b.Bootstrap())
}
