package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

func exampleToml() {
	var tml map[string]toml.Primitive
	var intKey int64

	meta, _ := toml.DecodeFile("file", &tml)

	fmt.Printf("Meta: %+v\n", meta)
	fmt.Printf("IsDefined: %+v\n", meta.IsDefined("key1"))
	fmt.Printf("keys: %+v\n", meta.Keys())
	fmt.Printf("Undecoded: %+v\n", meta.Undecoded())
	fmt.Printf("type: %+v\n", meta.Type("level2.foo"))
	fmt.Printf("prim: %+v\n", meta.PrimitiveDecode(tml["int_key"], &intKey))
	fmt.Printf("decoded: %+v\n", intKey)
	fmt.Printf("tml: %+v\n", tml)
}
