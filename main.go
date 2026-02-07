package main

import (
	"errors"
	"fmt"

	"github.com/ophel1ac/kv-exercise/kv"
)

func main() {
	db := kv.New[string, string]()

	db.Set("test", "answer")

	v, err := db.Get("test")
	fmt.Println("test=", v, err)

	v, err = db.Get("no")
	fmt.Println("no=", v, err)
	fmt.Println("error is", errors.Is(err, &kv.ErrNotFound[string]{}))

	db.Delete("test")
	db.Delete("no")

	v, err = db.Get("test")
	fmt.Println("after delete test=", v, err)

	v, err = db.Get("no")
	fmt.Println("after delete no=", v, err)

	_, err = db.Get("no")
	if err != nil {
		fmt.Println("значение не найдено")
	}
}
