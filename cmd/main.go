package main

import (
	"fmt"

	"github.com/jackparsonss/vertex/cmd/generated"
)

func main() {
	go generated.StartServer()

	product := generated.GetProduct(0)
	fmt.Println("product", product)

	products := generated.GetProducts()
	fmt.Println("products", products)

	select {}
}
