package gofast

type Version interface {
	Less() bool
	Equal() bool
	String() string
	Value() interface{}
}
