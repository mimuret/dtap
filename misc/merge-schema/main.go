package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Doc struct {
	Title string `json:"title"`
	Id    string `json:"$id"`
}

type Ref struct {
	Ref string `json:"$ref"`
}

func main() {
	if len(os.Args) < 2 {
		panic("missing argment")
	}
	files, err := filepath.Glob(os.Args[1] + "/*/config-schema.json")
	if err != nil {
		panic(err)
	}
	var oneOf = struct {
		Doc
		OneOf []Ref `json:"oneOf"`
	}{}
	oneOf.Title = "Plugins"
	oneOf.Id = os.Args[2]
	for _, file := range files {
		bs, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		doc := &Doc{}
		if err := json.Unmarshal(bs, doc); err != nil {
			panic(err)
		}
		oneOf.OneOf = append(oneOf.OneOf, Ref{Ref: doc.Id})
	}
	out, err := json.Marshal(oneOf)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
}
