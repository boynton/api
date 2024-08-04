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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/boynton/api/golang"
	"github.com/boynton/api/html"
	"github.com/boynton/api/httptrace"
	"github.com/boynton/api/markdown"
	"github.com/boynton/api/model"
	"github.com/boynton/api/openapi"
	"github.com/boynton/api/plantuml"
	"github.com/boynton/api/rdl"
	"github.com/boynton/api/sadl"
	"github.com/boynton/api/smithy"
	"github.com/boynton/data"
)

var Version string = "development version"

func main() {
	conf := data.NewObject()
	pNoValidate := flag.Bool("v", false, "Suppress validation of the assembled model")
	pQuiet := flag.Bool("q", false, "Quiet tool output, make it less verbose")
	pHelp := flag.Bool("h", false, "Show more help information")
	pList := flag.Bool("l", false, "List the entities in the model")
	pEntity := flag.String("e", "", "Show the specified entity.")
	pForce := flag.Bool("f", false, "Force overwrite if output file exists")
	pParseOnly := flag.Bool("p", false, "Parse input, display parse tree, and exit")
	pGen := flag.String("g", "smithy", "The generator for output")
	pNs := flag.String("ns", "", "The namespace to force if absent. Also used by the api generator to flatten to a single namespace")
	pOutdir := flag.String("o", "", "The directory to generate output into (defaults to stdout)")
	pWarn := flag.String("w", "show", "Warnings. 'show' or 'suppress' or 'error'. Default is 'show'")
	var params Params
	flag.Var(&params, "a", "Additional named arguments for a generator")
	var tags Tags
	flag.Var(&tags, "t", "Tag of entities to include. Prefix tag with '-' to exclude that tag")
	flag.Parse()
	if *pHelp {
		help()
		os.Exit(0)
	}
	model.MinimizeOutput = *pQuiet
	switch *pWarn {
	case "error":
		model.WarningsAreErrors = true
		model.ShowWarnings = true
	case "suppress":
		model.WarningsAreErrors = false
		model.ShowWarnings = false
	default:
		model.WarningsAreErrors = false
		model.ShowWarnings = true
	}
	gen := *pGen
	outdir := *pOutdir
	files := flag.Args()
	if len(files) == 0 {
		fmt.Printf("API tool %s [%s]\n", Version, "https://github.com/boynton/api")
		fmt.Println("usage: api [-vlfhpq] [-w warnlev] [-ns namespace] [-e entityid] [-d outdir] [-g generator] [-a key=val]* [-t tag]* file ...")
		flag.PrintDefaults()
		os.Exit(1)
	}
	schema, err := AssembleModel(files, tags, *pNs, *pParseOnly, *pNoValidate)
	if err != nil {
		model.Error("%s\n", err)
	}
	if *pParseOnly {
		os.Exit(0)
	}
	if *pList {
		if schema.Id != "" {
			fmt.Println(schema.Id + " (service)")
		}
		for _, o := range schema.Operations {
			fmt.Println(o.Id + " (operation)")
		}
		for _, n := range schema.ShapeNames() {
			fmt.Println(n)
		}
		os.Exit(0)
	} else if *pEntity != "" {
		eid := model.AbsoluteIdentifier(*pEntity)
		fmt.Println(">>>>>>>", eid, "<<<<<<")
		td := schema.GetTypeDef(eid)
		if td != nil {
			fmt.Println(td)
		} else {
			op := schema.GetOperationDef(eid)
			if op != nil {
				fmt.Println(op)
			} else {
				fmt.Println("No such entity:", eid)
				os.Exit(1)
			}
		}
		os.Exit(0)
	}
	if gen == "json" {
		fmt.Println(data.Pretty(schema))
		os.Exit(0)
	}
	conf.Put("outdir", outdir)
	if *pNs != "" {
		conf.Put("namespace", *pNs)
	}
	if *pForce {
		conf.Put("force", true)
	}
	for _, a := range params {
		kv := strings.Split(a, "=")
		if len(kv) > 1 {
			conf.Put(kv[0], kv[1])
		} else {
			conf.Put(a, true)
		}
	}
	generator, err := Generator(gen)
	if err == nil {
		err = generator.Generate(schema, conf)
	}
	if err != nil {
		fmt.Printf("*** %v\n", err)
		os.Exit(4)
	}
}

type Params []string

func (p *Params) String() string {
	return strings.Join([]string(*p), " ")
}
func (p *Params) Set(value string) error {
	*p = append(*p, strings.TrimSpace(value))
	return nil
}

type Tags []string

func (p *Tags) String() string {
	return strings.Join([]string(*p), " ")
}
func (p *Tags) Set(value string) error {
	*p = append(*p, strings.TrimSpace(value))
	return nil
}

func Generator(genName string) (model.Generator, error) {
	switch genName {
	case "summary":
		return new(model.SummaryGenerator), nil
	case "api":
		return new(model.ApiGenerator), nil
	case "markdown":
		return new(markdown.Generator), nil
	case "html":
		return new(html.Generator), nil
	case "smithy-ast":
		return new(smithy.AstGenerator), nil
	case "smithy":
		return new(smithy.IdlGenerator), nil
	case "sadl":
		return new(sadl.Generator), nil
	case "rdl":
		return new(rdl.Generator), nil
	case "openapi":
		return new(openapi.Generator), nil
	case "go", "golang":
		return new(golang.Generator), nil
	case "httptrace":
		return new(httptrace.Generator), nil
	case "plantuml":
		return new(plantuml.Generator), nil
	//case "swagger":
	//case "swagger-ui":
	//case "ts":
	default:
		return nil, fmt.Errorf("Unknown generator: %q", genName)
	}
}

func help() {
	msg := `
Supported API description formats for each input file extension:
   .api      api (the default for this tool
   .smithy   smithy
   .json     api, smithy, openapi, swagger (inferred by looking at the file contents)

The '' and 'namespace' options allow specifying those attributes for input formats
that do not require or support them. Otherwise a default is used based on the model being parsed.

Supported generators and options used from config if present
- api: Prints the native API representation to stdout. This is the default.
- json: Prints the parsed API data representation in JSON to stdout
- smithy: Prints the Smithy IDL representation to stdout.
- smithy-ast: Prints the Smithy AST representation to stdout
- openapi: Prints the OpenAPI Spec v3 representation to stdout
- plantuml: Prints the PlantUML representation of the API to stdout.
- sadl: Prints the SADL (an older format similar to api) to stdout. Useful for some additional generators.
- html: Prints html to stdout
   "-a detail-generator=api" - to generate the detail entries with "api" instead of "smithy", which is the default
- markdown: Prints markdown to stdout
   "-a detail-generator=api" - to generate the detail entries with "api" instead of "smithy", which is the default
   "-a use-html-pre-tag" - use the HTML <pre> tags instead of code fencing, allowing interior links. Not as compatible.

For any generator the following additional parameters are accepted:
- "-a sort" - causes the operations and types to be alphabetically sorted, by default the original order is preserved

`
	//not yet:
	// - java: Generate Java client code
	// - go: Generate Go client code
	// - gorilla: Generate Go code for a Gorilla-based server.
	// - jaxrs: Generate Java code for a Jersey/Jetty/Jackson based JAX-RS servier

	fmt.Println(msg)
}
