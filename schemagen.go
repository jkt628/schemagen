package schemagen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	schemaregistry "github.com/Landoop/schema-registry"
	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
	"github.com/actgardner/gogen-avro/generator"
	"github.com/actgardner/gogen-avro/types"
)

const (
	kindAvro = "Avro"
)

// SchemaConfig describes schemas to be downloaded.
type SchemaConfig struct {
	Subject   string `yaml:"subject" valid:"-"`
	Version   string `yaml:"version" valid:"-"`
	Package   string `yaml:"package" valid:"required"`
	LocalPath string `yaml:"localPath" valid:"-"`
}

// Config is the global application config.
type Config struct {
	Kind      string         `yaml:"kind" valid:"required"`
	Registry  string         `yaml:"registry" valid:"required"`
	Compile   bool           `yaml:"compile" valid:"required"`
	OutputDir string         `yaml:"outputDir" valid:"required"`
	Schemas   []SchemaConfig `yaml:"schemas" valid:"required"`

	NoFetch bool `yaml:"noFetch"`
}

// Run uses the Config to download schemas and to compile them.
func Run(ctx context.Context, cfg Config) error {
	if _, err := govalidator.ValidateStruct(cfg); err != nil {
		return err
	}

	switch cfg.Kind {
	case kindAvro:
		return generateAvro(ctx, cfg)
	default:
		return fmt.Errorf("kind %q is not supported", cfg.Kind)
	}
}

func generateAvro(ctx context.Context, cfg Config) error {
	client, err := schemaregistry.NewClient(cfg.Registry)
	if err != nil {
		return err
	}

	if err := os.Mkdir(cfg.OutputDir, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	if !cfg.NoFetch {
		for _, s := range cfg.Schemas {
			if err := os.Mkdir(path.Join(cfg.OutputDir, s.Package), 0755); err != nil && !os.IsExist(err) {
				return err
			}

			var schema string

			if s.Version == "latest" {
				sch, err := client.GetLatestSchema(s.Subject)
				if err != nil {
					return err
				}
				schema = sch.Schema
			} else {
				v, err := strconv.Atoi(s.Version)
				if err != nil {
					return fmt.Errorf("version %q is not valid: %v", s.Version, err)
				}

				sch, err := client.GetSchemaBySubject(s.Subject, v)
				if err != nil {
					return err
				}

				schema = sch.Schema
			}

			var b bytes.Buffer
			if err := json.Indent(&b, []byte(schema), "", "    "); err != nil {
				return err
			}

			if err := ioutil.WriteFile(path.Join(cfg.OutputDir, s.Package, s.Package+".avsc"), b.Bytes(), 0755); err != nil {
				return err
			}
		}
	}

	data, err := ioutil.ReadDir(cfg.OutputDir)
	if err != nil {
		return err
	}

	for _, d := range data {
		if !d.IsDir() {
			continue
		}

		files, err := filepath.Glob(filepath.Join(cfg.OutputDir, d.Name(), "/*.avsc"))
		if err != nil {
			return errors.Wrapf(err, "unable to find Avro schemas in %q", d.Name())
		}

		var schemas []string
		for _, f := range files {
			data, err := ioutil.ReadFile(f)
			if err != nil {
				return errors.Wrapf(err, "unable to read file %q", f)
			}

			schemas = append(schemas, string(data))
		}

		if err := CompileAvroSchema(d.Name(), cfg.OutputDir, schemas...); err != nil {
			return err
		}
	}

	return nil
}

// CompileAvroSchema compiles single Avro schema to Go code, using gopkg as a Go package name
// and out as target directory for compiled code.
func CompileAvroSchema(gopkg, out string, schemas ...string) error {
	pkg := generator.NewPackage(gopkg)
	namespace := types.NewNamespace(false)

	for _, schema := range schemas {
		_, err := namespace.TypeForSchema([]byte(schema))
		if err != nil {
			return err
		}

		for _, v := range namespace.Definitions {
			rec, ok := v.(*types.RecordDefinition)
			if !ok {
				continue
			}

			filename := generator.ToSnake(rec.Name()) + ".go"

			generateGoka(filename, rec, pkg)
		}

		if err := namespace.AddToPackage(pkg, codegenComment([]string{gopkg + ".avsc"}), false); err != nil {
			return err
		}

		if err := pkg.WriteFiles(path.Join(out, gopkg)); err != nil {
			return err
		}
	}

	return nil
}

// codegenComment generates a comment informing readers they are looking at
// generated code and lists the source avro files used to generate the code
//
// invariant: sources > 0
func codegenComment(sources []string) string {
	const fileComment = `// Code generated by github.com/burdiyan/schemagen. DO NOT EDIT.
/*
 * %s
 */`
	var sourceBlock []string
	if len(sources) == 1 {
		sourceBlock = append(sourceBlock, "SOURCE:")
	} else {
		sourceBlock = append(sourceBlock, "SOURCES:")
	}

	for _, source := range sources {
		_, fName := filepath.Split(source)
		sourceBlock = append(sourceBlock, fmt.Sprintf(" *     %s", fName))
	}

	return fmt.Sprintf(fileComment, strings.Join(sourceBlock, "\n"))
}
