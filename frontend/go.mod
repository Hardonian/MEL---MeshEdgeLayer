// Isolates the JS toolchain from the root Go module so ./... and go vet do not
// traverse npm packages that ship incidental .go files (e.g. flatted).
module github.com/mel-project/mel/frontend

go 1.24
