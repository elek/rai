# RAI - Go Project Guidelines

## Build Commands
```
go build                # Build the project
go run .                # Run the application
go test ./...           # Run all tests
go test ./pkg/...       # Run tests in specific package
go test -v ./pkg/... -run TestName # Run specific test
go vet ./...            # Run Go's built-in linter
```

## Code Style Guidelines
- Follow Go's official style guide: gofmt
- Run `go fmt ./...` before committing
- Use `golangci-lint run` for comprehensive linting
- Use descriptive camelCase for variable names, PascalCase for exported names
- Group imports: standard library, third-party, internal packages
- Error handling: always check errors, use meaningful error messages. Use `github.com/pkg/errors`
- Prefer explicit error returns over panics
- Use context.Context for request-scoped values/cancellation
- Document all exported functions, types, and constants
- Write tests for all exported functions. Use `github.com/stretchr/testify/assert`
