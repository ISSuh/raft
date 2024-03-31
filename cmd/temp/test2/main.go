package main

import (
	"fmt"
	"reflect"
)

func funct(a interface{}) {
	switch a.(type) {
	case int:
		fmt.Println("int")
	case *int:
		fmt.Println("*int")
	}

	t := reflect.TypeOf(a)
	
	value := a.(t.Name())
	fmt.Println(value)
}

func main() {
	a := 21

	funct(a)
	funct(&a)
}
