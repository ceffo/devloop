# build devloop binary
build:
  go build -o bin/devloop cmd/devloop/main.go

# run linter
lint:
  golangci-lint run ./...

# run tests
test:
  go test ./...

# install devloop binary to GOPATH/bin
install:
  go install ./cmd/devloop

# todo command to add items to the todo list
todo:
  echo "This command will accept a todo item and append it to a todo list file. Example usage: just todo 'Refactor the codebase'"

# process the todo list and execute tasks
process:
  echo "This command will process the todo list: make tasks and launch the devloop binary. Example usage: just process"