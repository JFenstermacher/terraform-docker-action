package main

import (
	"fmt"

	"github.com/JFenstermacher/terraform-docker-action/internal/parser"
)

func main() {
	hclParser := parser.HCLSecretsParser{}

	errors, ok := hclParser.ParseDirectory(".")

	if !ok {
		fmt.Println(errors)
	}
}
