package main

import (
	"fmt"

	"github.com/jackparsonss/platform-x/generated"
)

//go:generate go run ../generator/generator.go

// SayHello is a simple function that will be transformed to make HTTP calls
// @server path=/hello method=GET
func SayHello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

// Add two numbers and return the result
// @server path=/add method=POST
func Add(a, b int) int {
	return a + b
}

func main() {
	go generated.StartServer()

	result1 := generated.SayHello("World")
	fmt.Println("SayHello result:", result1)

	// result2 := generated.Add(5, 10)
	// fmt.Println("Add result:", result2)

	select {}
}
