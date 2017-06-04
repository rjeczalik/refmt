package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rjeczalik/refmt/object"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/printer"
	yaml "gopkg.in/yaml.v1"
)

var f = &Format{
	Type: flag.String("t", "", "Output format type."),
}

var m = map[string]struct {
	marshal   func(interface{}) ([]byte, error)
	unmarshal func([]byte) (interface{}, error)
}{
	"json": {
		marshal: jsonMarshal,
		unmarshal: func(p []byte) (v interface{}, _ error) {
			if err := json.Unmarshal(p, &v); err != nil {
				return nil, err
			}
			return v, nil
		},
	},
	"yaml": {
		marshal: yaml.Marshal,
		unmarshal: func(p []byte) (v interface{}, _ error) {
			if err := yaml.Unmarshal(p, &v); err != nil {
				return nil, err
			}
			return object.FixYAML(v), nil
		},
	},
	"hcl": {
		marshal: func(v interface{}) ([]byte, error) {
			p, err := jsonMarshal(v)
			if err != nil {
				return nil, err
			}
			nd, err := hcl.Parse(string(p))
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			if err := printer.Fprint(&buf, nd); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		},
		unmarshal: func(p []byte) (v interface{}, _ error) {
			if err := hcl.Unmarshal(p, &v); err != nil {
				return nil, err
			}
			object.FixHCL(v)
			return v, nil
		},
	},
}

func typ(file string) string {
	ext := filepath.Base(file)
	if i := strings.LastIndex(ext, "."); i != -1 {
		ext = ext[i+1:]
	}
	switch ext = strings.ToLower(ext); ext {
	case "yml":
		return "yaml"
	case "json", "yaml", "hcl":
		return ext
	default:
		return ""
	}
}

var autoTryOrder = []string{"hcl", "json", "yaml"}

type Format struct {
	Type   *string   // autodetect if nil or empty
	Stdin  io.Reader // os.Stdin if nil
	Stdout io.Writer // os.Stdout if nil
	Stderr io.Writer // os.Stderr if nil
}

func (f *Format) Refmt(in, out string) error {
	v, err := f.unmarshal(in)
	if err != nil {
		return err
	}
	return f.marshal(v, out)
}

func (f *Format) stdin() io.Reader {
	if f.Stdin != nil {
		return f.Stdin
	}
	return os.Stdin
}

func (f *Format) stdout() io.Writer {
	if f.Stdout != nil {
		return f.Stdout
	}
	return os.Stdout
}

func (f *Format) stderr() io.Writer {
	if f.Stderr != nil {
		return f.Stderr
	}
	return os.Stderr
}

func (f *Format) unmarshal(file string) (v interface{}, err error) {
	p, err := f.read(file)
	if err != nil {
		return nil, err
	}
	if t := typ(file); t != "" {
		return m[t].unmarshal(p)
	}
	for _, t := range autoTryOrder {
		v, err = m[t].unmarshal(p)
		if err == nil {
			return v, nil
		}
	}
	return nil, err
}

func (f *Format) marshal(v interface{}, file string) error {
	t := typ(file)
	if t == "" && f.Type != nil {
		t = strings.ToLower(*f.Type)
	}
	if _, ok := m[t]; !ok {
		return fmt.Errorf("unknown output format: %q", t)
	}
	p, err := m[t].marshal(v)
	if err != nil {
		return err
	}
	return f.write(p, file)
}

func (f *Format) read(file string) ([]byte, error) {
	switch file {
	case "":
		return nil, errors.New("no file specified")
	case "-":
		return ioutil.ReadAll(f.stdin())
	default:
		return ioutil.ReadFile(file)
	}
}

func (f *Format) write(p []byte, file string) error {
	switch file {
	case "":
		return errors.New("no file specified")
	case "-":
		_, err := f.stdout().Write(p)
		return err
	default:
		return ioutil.WriteFile(file, p, 0644)
	}
}

func jsonMarshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Refmt(in, out string) error { return f.Refmt(in, out) }
