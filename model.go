package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boynton/api/openapi"
	"github.com/boynton/api/sadl"
	"github.com/boynton/smithy"
)

var ImportFileExtensions = map[string][]string{
	".smithy": []string{"smithy"},
	".json":   []string{"smithy", "openapi"},
	".sadl":   []string{"sadl"},
}

func expandPaths(paths []string) ([]string, error) {
	var result []string
	for _, path := range paths {
		ext := filepath.Ext(path)
		if _, ok := ImportFileExtensions[ext]; ok {
			result = append(result, path)
		} else {
			fi, err := os.Stat(path)
			if err != nil {
				return nil, err
			}
			if fi.IsDir() {
				err = filepath.Walk(path, func(wpath string, info os.FileInfo, errIncoming error) error {
					if errIncoming != nil {
						return errIncoming
					}
					ext := filepath.Ext(wpath)
					if _, ok := ImportFileExtensions[ext]; ok {
						result = append(result, wpath)
					}
					return nil
				})
			}
		}
	}
	return result, nil
}

func AssembleModel(paths []string, tags []string) (*smithy.AST, error) {
	flatPathList, err := expandPaths(paths)
	if err != nil {
		return nil, err
	}
	assembly := &smithy.AST{
		Smithy: "1.0",
	}
	for _, path := range flatPathList {
		var ast *smithy.AST
		var err error
		ext := filepath.Ext(path)
		switch ext {
		case ".smithy":
			ast, err = smithy.Parse(path)
		case ".json":
			ast, err = smithy.LoadAST(path)
			if err != nil {
				ast, err = openapi.Import(path)
			}
		case ".sadl":
			ast, err = sadl.Import(path)
		default:
			return nil, fmt.Errorf("parse for file type %q not implemented", ext)
		}
		if err != nil {
			return nil, err
		}
		err = assembly.Merge(ast)
		if err != nil {
			return nil, err
		}
	}
	if len(tags) > 0 {
		assembly.Filter(tags)
	}
	err = assembly.Validate()
	if err != nil {
		return nil, err
	}
	return assembly, nil
}
