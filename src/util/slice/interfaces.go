package slice

// Named used to ensure implementing struct has GetName method for functions in find.go and sort.go
type Named interface {
	GetName() string
}
