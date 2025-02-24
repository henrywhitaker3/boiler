# Boiler

A simple DI container

## Usage

```go
type Demo struct{
    value string
}

type UsesDemo struct{
    demo *Demo
}

func (u UsesDemo) Print() {
    fmt.Println(u.demo.value)
}


b := boiler.New()

boiler.MustRegister(b, func(*Boiler) (*Demo, error) {
    return &Demo{
        value: "bongo",
    }, nil
})

boiler.MustRegister(b, func(b *Boiler) (UsesDemo, error) {
    demo, err := boiler.Resolve[*Demo](b)
    if err != nil {
        return UsesDemo{}, err
    }
    return UsesDemo{demo: demo}, nil
})

b.MustBootstrap()

boiler.MustResolve[UsesDemo].Print() // prints: bongo
```
