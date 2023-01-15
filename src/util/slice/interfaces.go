package slice

// Named used to ensure implementing struct has GetName method for functions in find.go and sort.go
type Named interface {
	GetName() string
}

// Interface for testing purposes
type TestNamedStruct struct {
	Name string
	Slice []int
}

func (o TestNamedStruct) GetName() string {
	return o.Name
}
