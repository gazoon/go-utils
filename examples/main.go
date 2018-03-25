package main

import (
	"fmt"

	"github.com/gazoon/go-utils"
)

type Config struct {
	Settings struct {
		Foo int
		Bar string
	}
	Params struct {
		Tmp string
		Var float64
	}
	Storage struct {
		Time struct {
			Prev int
		}
	}
}

func main() {
	conf := &Config{}
	err := utils.ParseConfig("examples/config", conf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", conf)
}
