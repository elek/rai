package tool

type Tool interface {
	Name() string
	Description() string
	Input() any
	Handle(input any) string
}
