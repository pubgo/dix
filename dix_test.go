package dix

import (
	"fmt"
	"testing"
)

func TestModel(t *testing.T) {
	type ss struct {
		Model
	}

	var s interface{} = ss{}

	param1, ok := s.(dixData)
	fmt.Println(s,ok)
	if ok {
		fmt.Println(param1)
		s = checkDixDataType(param1)
	}

	fmt.Println(s)
}
