package main

import (
	"os"
	"path/filepath"

	"k8s.io/gengo/args"

	"yunion.io/x/code-generator/pkg/swagger-gen/generators"
	"yunion.io/x/log"
)

func Packages(ctx *generator.Context, arguments *args.GeneratorArgs) generator.Packages {
	boilerplate, err := arguments.LoadGoBoilerplate()
	if err != nil {
		log.Fatalf("Failed loading boilerplate: %v", err)
	}
	pkgs := generator.Packages{}
	inputs := sets.NewString(ctx.Inputs...)
	header := append([]byte(fmt.Sprintf("// +build !%s\n\n", arguments.GeneratedBuildTag)), boilerplate...)

	outPkgName := strings.Split(filepath.Base(arguments.OutputPackagePath), ".")[0]
	pkgPath := arguments.OutputPackagePath
	svcName := outPkgName
	pkgs = append(pkgs, NewDocPackage(outPkgName, pkgPath, header, svcName))
	for i := range inputs {
		pkg := ctx.Universe[i]
		if pkg == nil {
			continue
		}
		log.Infof("Considering pkg %q", pkg.Path)
		pkgs = append(pkgs,
			&generator.DefaultPackage{
				PackageName: outPkgName,
				PackagePath: pkgPath,
				HeaderText:  header,
				GeneratorFunc: func(c *generator.Context) []generator.Generator {
					return []generator.Generator{
						// Generate swagger code by model.
						generators.NewSwaggerGen(arguments.OutputFileBaseName, pkg.Path, ctx.Order),
					}
				},
				FilterFunc: func(c *generator.Context, t *types.Type) bool {
					return t.Name.Package == pkg.Path
				},
			},
		)
	}
	return pkgs
}

type generator struct {
	generator.Generator
}

func NewGenerator(gen generator.Generator) generator.Generator {
	return &generator{
		Generator: gen,
	}
}

func (g *generator) Imports(c *generator.Context) []string {
	return []string{
		"yunion.io/x/yunion-kube/pkg/apis",
	}
}
