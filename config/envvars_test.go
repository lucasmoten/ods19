package config_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/config"
)

func TestEnvVars(t *testing.T) {

	ff, _ := os.Open("envvars.go")
	// DIMEODS-1262 - ensure file closed if not nil
	if ff != nil {
		defer ff.Close()
	}
	src, _ := ioutil.ReadAll(ff)
	fset := token.NewFileSet() // positions are relative to fset
	f, err := parser.ParseFile(fset, "envvars.go", string(src), 0)
	if err != nil {
		panic(err)
	}

	// Gather the const identifiers as into a []string
	var constants []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.CONST {
				// This is the one-big-declaration for const
				for _, spec := range x.Specs {
					if vspec, ok := spec.(*ast.ValueSpec); ok {
						constants = append(constants, vspec.Names[0].Name)
					}
				}
			}
		}
		return true
	})

	// Compare to the length of the exported slice.
	if len(constants) != len(config.Vars) {

		t.Errorf("Go AST parser found %v const declarations, but Vars array contains %v. You may need to add a declared const to the Vars array", len(constants), len(config.Vars))
	}

}
