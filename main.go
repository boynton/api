package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/boynton/api/openapi"
	"github.com/boynton/api/sadl"
	"github.com/boynton/data"
	"github.com/boynton/smithy"
)

func main() {
	conf := data.NewObject()
	pVersion := flag.Bool("v", false, "Show api tool version and exit")
	pList := flag.Bool("l", false, "List the entities in the model")
	pForce := flag.Bool("f", false, "Force overwrite if output file exists")
	pGen := flag.String("g", "sadl", "The generator for output")
	pOutdir := flag.String("o", "", "The directory to generate output into (defaults to stdout)")
	var params Params
	flag.Var(&params, "a", "Additional named arguments for a generator")
	var tags Tags
	flag.Var(&tags, "t", "Tag of entities to include")

	flag.Parse()
	if *pVersion {
		fmt.Printf("API tool %s [%s]\n", smithy.ToolVersion, "https://github.com/boynton/api")
		os.Exit(0)
	}
	gen := *pGen
	outdir := *pOutdir
	files := flag.Args()
	if len(files) == 0 {
		fmt.Println("usage: smithy [-v] [-o outfile] [-g generator] [-a key=val]* file ...")
		flag.PrintDefaults()
		os.Exit(1)
	}
	ast, err := AssembleModel(files, tags)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	if *pList {
		for _, n := range ast.ShapeNames() {
			fmt.Println(n)
		}
		os.Exit(0)
	}
	conf.Put("outdir", outdir)
	conf.Put("force", *pForce)
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
		err = generator.Generate(ast, conf)
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

func Generator(genName string) (smithy.Generator, error) {
	switch genName {
	case "smithy-ast":
		return new(smithy.AstGenerator), nil
	case "smithy":
		return new(smithy.IdlGenerator), nil
	case "sadl":
		return new(sadl.IdlGenerator), nil
	case "openapi":
		return new(openapi.Generator), nil
	default:
		return nil, fmt.Errorf("Unknown generator: %q", genName)
	}
}
