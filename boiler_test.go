package boiler

import (
	"context"
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

	require.Nil(t, RegisterNamed[Demo](b, "bongo", func(*Boiler) (Demo, error) {
		return Demo{value: "bongo"}, nil
	}))
	require.Nil(t, RegisterNamed[Demo](b, "orange", func(*Boiler) (Demo, error) {
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
