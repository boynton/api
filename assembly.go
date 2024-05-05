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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	//"github.com/boynton/data"
	"github.com/boynton/api/model"
	"github.com/boynton/api/openapi"
	//	"github.com/boynton/api/sadl"
	"github.com/boynton/api/smithy"
	"github.com/boynton/api/swagger"
)

var ImportFileExtensions = map[string]string{
	".api":    "api",
	".smithy": "smithy",
	".sadl":   "sadl",
	".rdl":    "rdl",
}

func determineFormat(path string) string {
	ext := filepath.Ext(path)
	if f, ok := ImportFileExtensions[ext]; ok {
		if f != "" {
			return f
		}
	}
	if ext == ".json" {
		var raw map[string]interface{}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return ""
		}
		err = json.Unmarshal(data, &raw)
		if err != nil {
			return ""
		}
		if _, ok := raw["smithy"]; ok {
			return "smithy"
		}
		if _, ok := raw["openapi"]; ok {
			return "openapi"
		}
		if _, ok := raw["swagger"]; ok {
			return "swagger"
		}
		return "api"
	}
	if ext == ".yaml" {
		//openapi/swagger in yaml
	}
	return ""
}

func expandPaths(paths []string) ([]string, string, error) {
	format := ""
	var result []string
	for _, path := range paths {
		f := determineFormat(path)
		if f != "" {
			if format == "" {
				format = f
			} else {
				if format != f {
					return nil, "", fmt.Errorf("Cannot combine input model formats (found both %q and %q)", format, f)
				}
			}
			result = append(result, path)
		} else {
			fi, err := os.Stat(path)
			if err != nil {
				return nil, "", err
			}
			if fi.IsDir() {
				err = filepath.Walk(path, func(wpath string, info os.FileInfo, errIncoming error) error {
					if errIncoming != nil {
						return errIncoming
					}
					f := determineFormat(wpath)
					if f != "" {
						if format == "" {
							format = f
						} else {
							if format != f {
								return fmt.Errorf("Cannot combine input model formats (found both %q and %q)", format, f)
							}
						}
						result = append(result, wpath)
					}
					return nil
				})
				if err != nil {
					return nil, "", err
				}
			}
		}
	}
	return result, format, nil
}

func AssembleModel(paths []string, tags []string, ns string, parseOnly bool, noValidate bool) (*model.Schema, error) {
	flatPathList, format, err := expandPaths(paths)
	if err != nil {
		return nil, err
	}
	if format == "" {
		return nil, fmt.Errorf("Cannot determine acceptable input file format")
	}
	if ns == "" {
		ns = "unspecified"
	}
	var schema *model.Schema
	switch format {
	case "api":
		schema, err = model.Load(flatPathList, tags)
	case "smithy":
		schema, err = smithy.Import(flatPathList, tags, parseOnly)
	case "sadl":
		//schema, err = sadl.Import(flatPathList, tags)
		err = fmt.Errorf("sadl.Import NYI")
	case "openapi":
		schema, err = openapi.Import(flatPathList, tags, ns)
	case "swagger":
		schema, err = swagger.Import(flatPathList, tags, ns)
	case "rdl":
		err = fmt.Errorf("rdl.Import NYI")
	default:
		err = fmt.Errorf("unknown format: %q", format)
	}
	if err == nil {
		if !parseOnly && !noValidate {
			err = schema.Validate()
		}
	}
	return schema, err
}
