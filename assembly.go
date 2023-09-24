/*
Copyright 2022 Lee R. Boynton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"

	//"github.com/boynton/data"
	"github.com/boynton/api/model"
	//	"github.com/boynton/api/openapi"
	//	"github.com/boynton/api/sadl"
	"github.com/boynton/api/smithy"
	//	"github.com/boynton/api/swagger"
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

func AssembleModel(paths []string, tags []string, ns string) (*model.Schema, error) {
	flatPathList, err := expandPaths(paths)
	if err != nil {
		return nil, err
	}
	assembly := &model.Schema{}
	for _, path := range flatPathList {
		var model *model.Schema
		var err error
		ext := filepath.Ext(path)
		switch ext {
		case ".smithy":
			model, err = smithy.Import(path)
		case ".json":
			model, err = smithy.Import(path)
			/*
					if err != nil {
						model, err = openapi.Import(path, ns)
						if err != nil {
							model, err = swagger.Import(path, ns)
						}
					}
				case ".sadl":
					model, err = sadl.Import(path, ns)
			*/
		default:
			return nil, fmt.Errorf("parse for file type %q not implemented", ext)
		}
		if err != nil {
			return nil, err
		}
		err = assembly.Merge(model)
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
