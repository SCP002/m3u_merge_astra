package slice

// Interface for testing purposes
type TestNamedStruct struct {
	Name string
	Slice []int
}

func (o TestNamedStruct) GetName() string {
	return o.Name
}
