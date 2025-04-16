package product

import "github.com/google/uuid"

//go:generate go run ../../engine/engine.go
type Product struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Price string    `json:"price"`
}

type ProductService struct {
	Products []Product
}

func NewProductService() *ProductService {
	return &ProductService{
		Products: []Product{
			{ID: uuid.New(), Name: "Product 1", Price: "$10"},
			{ID: uuid.New(), Name: "Product 2", Price: "$20"},
			{ID: uuid.New(), Name: "Product 3", Price: "$30"},
		},
	}
}

// @server path=/hello method=GET
func (p *ProductService) GetProducts() []Product {
	return p.Products
}

// @server path=/hello method=GET
func (p *ProductService) GetProduct(i int) Product {
	return p.Products[i]
}
