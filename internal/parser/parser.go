package parser

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type HCLSecretsParser struct {
	ProviderSecrets []string
}

func (hsp *HCLSecretsParser) ParseDirectory(dir string) ([]error, bool) {
	f, err := os.Stat(dir)

	var errors []error
	if err != nil {
		return []error{err}, false
	}

	if !f.IsDir() {
		err := fmt.Errorf("Provided filename, %s, is not a directory.", dir)

		return []error{err}, false
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if hsp.quickFileCheck(path) {
				errs, _ := hsp.ParseFile(path)
				errors = append(errors, errs...)
			}
		}

		return nil
	})

	if err != nil {
		errors = append(errors, err)
		return errors, false
	}

	return errors, len(errors) == 0
}

func (hsp *HCLSecretsParser) ParseFile(path string) ([]error, bool) {
	bytes, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatalf("Failed to read file: %s\n", path)
	}

	return hsp.ParseContent(path, bytes)
}

func (hsp *HCLSecretsParser) ParseContent(path string, bytes []byte) ([]error, bool) {
	file, diags := hclsyntax.ParseConfig(bytes, path, hcl.Pos{Line: 1, Column: 1})

	if diags.HasErrors() {
		return diags.Errs(), false
	}

	if errs := hsp.parseFile(file); len(errs) > 0 {
		return errs, false
	}

	return nil, true
}

func (hsp *HCLSecretsParser) parseFile(f *hcl.File) []error {
	body, ok := f.Body.(*hclsyntax.Body)
	if !ok {
		return []error{fmt.Errorf("Error while converting to hclsyntax.Body")}
	}

	if errs := hsp.parseBody(body); len(errs) > 0 {
		return errs
	}

	return nil
}

func (hsp *HCLSecretsParser) parseBody(body *hclsyntax.Body) []error {
	var errors []error
	for _, block := range body.Blocks {
		if errs := hsp.parseBlock(block); len(errs) > 0 {
			return errs
		}
	}

	for _, value := range body.Attributes {
		if errs := hsp.parseExpression(value.Expr); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	return errors
}

func (hsp *HCLSecretsParser) parseBlock(block *hclsyntax.Block) []error {
	if block.Type == "provider" {
		return nil
	}

	return hsp.parseBody(block.Body)
}

func (hsp *HCLSecretsParser) parseExpression(expr hclsyntax.Expression) []error {
	var errors []error
	traversals := hclsyntax.Variables(expr)

	for _, traversal := range traversals {
		for _, traverser := range traversal {
			attr, ok := traverser.(hcl.TraverseAttr)
			if !ok {
				continue
			}

			for _, secret := range hsp.ProviderSecrets {
				if attr.Name == secret {
					errors = append(errors, fmt.Errorf("Provider secret value '%s', %s, found outside of provider configuration", secret, attr.SourceRange()))
				}
			}
		}
	}

	return errors
}

func (hsp *HCLSecretsParser) quickFileCheck(path string) bool {
	if !(strings.HasSuffix(path, ".tf.json") || strings.HasSuffix(path, ".tf")) {
		return false
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return true // Allow parser failures to catch
	}

	// There's probably a faster method for finding if any string in ProviderSecrets exist in file
	// I think I remember reading regexp is slow in golang?
	group := fmt.Sprintf("(%s)", strings.Join(hsp.ProviderSecrets, "|"))

	found, err := regexp.Match(group, bytes)
	if err != nil {
		return true
	}

	return found
}
