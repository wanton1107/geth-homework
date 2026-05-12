package wanton

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Teacher struct {
	School string `json:"school,omitempty"`
	Class  string `json:"class"`
	Person
}
