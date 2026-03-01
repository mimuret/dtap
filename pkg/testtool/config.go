package testtool

import "github.com/mimuret/dtap/v3/pkg/config"

func MustInputBlock(pType, name, hclData string) *config.InputBlock {
	block, err := config.NewInputBlock(pType, name, hclData)
	if err != nil {
		panic(err)
	}
	return block
}

func MustFilterBlock(pType, name, hclData string) *config.FilterBlock {
	block, err := config.NewFilterBlock(pType, name, hclData)
	if err != nil {
		panic(err)
	}
	return block
}
func MustOutputBlock(pType, name, hclData string) *config.OutputBlock {
	block, err := config.NewOutputBlock(pType, name, hclData)
	if err != nil {
		panic(err)
	}
	return block
}
