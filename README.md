# Vertex

Turns any function call into a client/server call

## Installation

```bash
go install github.com/jackparsonss/vertex
```

## How to use

### 1. In your Go code, annotate functions or methods with `@server` comments

```go
package serviceA

// GetUser fetches a user by ID
// @server path=/api/users method=GET
func GetUser(id int) *User {
    // Your implementation here
    return &User{ID: id, Name: "Test User"}
}

// SaveUser creates a new user
// @server path=/api/users method=POST
func SaveUser(name string) *User {
    // Your implementation here
    return &User{ID: 1, Name: name}
}

// For methods on structs, make sure you have a constructor function:
func NewUserService() *UserService {
    return &UserService{}
}

// GetUsers retrieves all users
// @server path=/api/users/all method=GET
func (s *UserService) GetUsers() []User {
    // Your implementation here
    return []User{{ID: 1, Name: "User 1"}, {ID: 2, Name: "User 2"}}
}
```

### 2. Run the generator

```bash
vertex run
```

### 3. Include the generated code in your application

```go
package serviceB

import "vertex"

// @server path=/api/do-something
func doSomething() {
  u := vertex.GetUser(1)

  return u.Name
}

// curl http://localhost:8080/api/do-something
// {name: "Test User"}
```

## Features

- Generates both server and client code
- Handles both standalone functions and methods on structs
- Supports GET and POST HTTP methods
- Automatic service instance creation for struct methods
- Type-safe client wrappers that match the original function signatures
- Customizable server port and client endpoint URL
