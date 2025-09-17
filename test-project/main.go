package main

import (
	"errors"
	"fmt"
)

var AFixedWidthArray = []byte{0x02}

func main() {
	struct1 := &Struct1{field: 1}
	struct2 := &Struct2{fieldX: 2, fieldY: 3}
	result, err := check(struct1)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Result:", result)
	}
	result, _ = check(struct2) // Unused error result
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println("Result:", result)
	check(struct2) // Unused error result

	someOtherFunction() // Just to make sure that it isn't held to be unreachable
}

type Getter interface {
	Get() int
}

type Struct1 struct {
	field int
}

func (s *Struct1) Get() int {
	return s.field
}

type Struct2 struct {
	fieldX int
	fieldY int
}

func (s *Struct2) Get() int {
	return s.fieldX + s.fieldY
}

func check(getter Getter) (int, error) {
	res := getter.Get()
	if res < 0 {
		return 0, errors.New("negative result")
	}
	return res, nil
}

func someOtherFunction() {
	AFixedWidthArray = append(AFixedWidthArray, 0x03)
	AFixedWidthArray = []byte{0x04, 0x05, 0x6}
}
