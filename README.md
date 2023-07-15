# pika
ORM-like SQL builder

# Features
* Any feature supported by sqlx
* Support for [AIP-160](https://google.aip.dev/160) filtering
* Utilities to help with [AIP-132](https://google.aip.dev/132) for List calls
* Support for determining filters based on Protobuf messages
* Automatically selecting columns in struct
* Count, Get, All, Create, Update, Delete and more.
* Support for simple joins

# Example

### Simple connect and Get
```go
package main

import (
	"go.ciq.dev/pika"
	"log"
)

type User struct {
	PikaTableName string `pika:"users"`
	ID            int64  `db:"id" pika:"omitempty"`
	Name          string `db:"name"`
}

func main() {
	psql, _ := pika.NewPostgreSQL("postgres://postgres:postgres@localhost:5432/test")
	args := pika.NewArgs()
	args.Set("id", 1)
	qs := pika.Q[User](psql).Filter("id=:id").Args(args)
	user, _ := qs.Get()
	
	log.Println(user)
}
```

### AIP-160
```go
package main

import (
	"go.ciq.dev/pika"
	"log"
)

type Article struct {
	PikaTableName string    `pika:"users"`
	ID            int64     `db:"id" pika:"omitempty"`
	CreatedAt     time.Time `db:"created_at" pika:"omitempty"`
	Title         string    `db:"title"`
	Body          string    `db:"body"`
}

func main() {
	psql, _ := pika.NewPostgreSQL("postgres://postgres:postgres@localhost:5432/test")

	qs := pika.Q[Article](s.db)
	// Get the following articles
	//   * The title MUST contain Hello and the body MUST contain World
	//   * If the above is not match, the article MUST be created before 2023-07-30
	qs, err := qs.AIP160(`(title:"Hello" AND body:"World") OR (created_at < 2023-07-30T00:00:00Z)`, pika.AIPFilterOptions{})
	if err != nil {
		return nil, err
	}

	rows, err := qs.All()
	if err != nil {
		return nil, err
	}

	log.Println(rows)
}
```