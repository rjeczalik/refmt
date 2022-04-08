package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"rafal.dev/refmt/object"

	"github.com/go-sql-driver/mysql"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/pkg/errors"
	"github.com/savaki/jq"
	yaml "gopkg.in/yaml.v1"
)

var f = &Format{
	Type: flag.String("t", "", "Output format type."),
}

type envCodec struct {
	prefix *string
}

func (c *envCodec) codec() codec {
	return codec{
		marshal:   c.marshal,
		unmarshal: c.unmarshal,
	}
}

type Options map[string]string

func parseOptions(key string) (string, Options) {
	opts := make(Options)

	for i, j := 0, 0; ; {
		if i = strings.IndexByte(key, '['); j == -1 {
			break
		}

		if j = strings.IndexByte(key, ']'); i == -1 {
			break
		}

		k, v := key[i+1:j], ""

		if l := strings.IndexByte(k, '='); l != -1 {
			k, v = k[:l], v[l+1:]
		}

		opts[k] = v

		key = key[:i] + key[j+1:]
	}

	return key, opts
}

func executeOptions(v interface{}, opts Options, marshal bool) interface{} {
	if _, ok := opts["b64"]; ok {
		if s, ok := v.(string); ok {
			if marshal {
				if _, err := base64.StdEncoding.DecodeString(s); err == nil {
					return s
				}

				return base64.StdEncoding.EncodeToString([]byte(s))
			}

			if p, err := base64.StdEncoding.DecodeString(s); err == nil {
				return string(p)
			}

			return s
		}
	}

	return v
}

func transform(marshal bool) func(map[string]interface{}, string) {
	return func(m map[string]interface{}, key string) {
		v := m[key]

		k, opts := parseOptions(key)
		v = executeOptions(v, opts, marshal)

		delete(m, key)
		m[k] = v
	}
}

func (c *envCodec) marshal(v interface{}) ([]byte, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, errors.New("envCoded: cannot marshal non-object value")
	}

	object.Walk(m, transform(true))

	var (
		p    = *c.prefix
		envs = object.Flatten(m, "_")
		keys = object.Keys(envs)
		buf  bytes.Buffer
	)

	for _, k := range keys {
		fmt.Fprintf(&buf, "%s%s=%s\n", p, strings.ToUpper(k), envs[k])
	}

	return buf.Bytes(), nil
}

func (c *envCodec) unmarshal([]byte) (interface{}, error) {
	return nil, errors.New("envCodec: not implemented")
}

type codec struct {
	marshal   func(interface{}) ([]byte, error)
	unmarshal func([]byte) (interface{}, error)
}

type jsonCodec struct {
	compact *bool
	b64     *bool
}

func (c *jsonCodec) codec() codec {
	return codec{
		marshal:   c.marshal,
		unmarshal: c.unmarshal,
	}
}

func (c *jsonCodec) marshal(v interface{}) ([]byte, error) {
	if m, ok := v.(map[string]interface{}); ok {
		object.Walk(m, transform(true))
	}

	if *c.compact {
		return json.Marshal(v)
	}

	return jsonMarshal(v)
}

func (c *jsonCodec) unmarshal(p []byte) (v interface{}, _ error) {
	if err := json.Unmarshal(p, &v); err != nil {
		return nil, err
	}
	if m, ok := v.(map[string]interface{}); ok {
		object.Walk(m, transform(false))
	}
	return v, nil
}

type yamlCodec struct{}

func (c *yamlCodec) codec() codec {
	return codec{
		marshal:   c.marshal,
		unmarshal: c.unmarshal,
	}
}

func (c *yamlCodec) marshal(v interface{}) ([]byte, error) {
	if m, ok := v.(map[string]interface{}); ok {
		object.Walk(m, transform(true))
	}
	return yaml.Marshal(v)
}

func (c *yamlCodec) unmarshal(p []byte) (v interface{}, _ error) {
	if err := yaml.Unmarshal(p, &v); err != nil {
		return nil, err
	}

	return object.FixYAML(v), nil
}

var m = map[string]codec{
	"json": (&jsonCodec{
		compact: flag.Bool("c", false, "One-line output for JSON format."),
	}).codec(),
	"yaml": new(yamlCodec).codec(),
	"hcl": {
		marshal: func(v interface{}) ([]byte, error) {
			if m, ok := v.(map[string]interface{}); ok {
				object.Walk(m, transform(true))
			}
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
			if m, ok := v.(map[string]interface{}); ok {
				object.Walk(m, transform(false))
			}
			return v, nil
		},
	},
	"env": (&envCodec{
		prefix: flag.String("p", "", "Prefix for keys when type is env."),
	}).codec(),
}

func typ(file string) string {
	ext := filepath.Base(file)
	if i := strings.LastIndex(ext, "."); i != -1 {
		ext = ext[i+1:]
	}
	switch ext = strings.ToLower(ext); ext {
	case "yml":
		return "yaml"
	case "tf":
		return "hcl"
	case "tfstate":
		return "json"
	case "json", "yaml", "hcl":
		return ext
	default:
		return ""
	}
}

var autoTryOrder = []string{"hcl", "json", "yaml", "env"}

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

func (f *Format) Merge(orig, mixin, out string) error {
	vorig, err := f.unmarshal(orig)
	if fi, e := os.Stat(orig); os.IsNotExist(e) || fi.Size() == 0 {
		vorig = make(map[string]interface{})
	} else if err != nil {
		return err
	}
	vmixin, err := f.unmarshal(mixin)
	if err != nil {
		return err
	}
	morig, ok := vorig.(map[string]interface{})
	if !ok {
		return fmt.Errorf("original object is %T, expected %T", vorig, (map[string]interface{})(nil))
	}
	mmixin, ok := vmixin.(map[string]interface{})
	if !ok {
		return fmt.Errorf("mixin object is %T, expected %T", vmixin, (map[string]interface{})(nil))
	}
	if err := object.Merge(mmixin, morig); err != nil {
		return err
	}
	return f.marshal(morig, out)
}

func (f *Format) DSN(dsn string) error {
	if dsn == "" {
		p, err := f.read("-")
		if err != nil {
			return err
		}
		dsn = string(bytes.TrimSpace(p))
	}
	c, err := mysql.ParseDSN(dsn)
	if err != nil {
		return err
	}
	// --user=root --password=101202 --port=5506 --host=127.0.0.1 --database=scylla_dbaas
	var buf bytes.Buffer
	if c.User != "" {
		buf.WriteString("--user=")
		buf.WriteString(c.User)
		buf.WriteRune(' ')
	}
	if c.Passwd != "" {
		buf.WriteString("--password=")
		buf.WriteString(c.Passwd)
		buf.WriteRune(' ')
	}
	if c.Addr != "" {
		if host, port, err := net.SplitHostPort(c.Addr); err == nil {
			buf.WriteString("--host=")
			buf.WriteString(host)
			buf.WriteString(" --port=")
			buf.WriteString(port)
		} else {
			buf.WriteString("--host=")
			buf.WriteString(c.Addr)
		}
		buf.WriteRune(' ')
	}
	if c.DBName != "" {
		buf.WriteString("--database=")
		buf.WriteString(c.DBName)
		buf.WriteRune(' ')
	}
	buf.WriteRune('\n')
	return f.write(buf.Bytes(), "-")
}

func (f *Format) Set(in, key, value string) error {
	v, err := f.unmarshal(in)
	if fi, e := os.Stat(in); os.IsNotExist(e) || fi.Size() == 0 {
		v = make(map[string]interface{})
	} else if err != nil {
		return err
	}
	vobj, ok := v.(map[string]interface{})
	if !ok {
		return fmt.Errorf("original object is %T, expected %T", v, (map[string]interface{})(nil))
	}
	if err := object.SetFlatKeyValue(vobj, key, value); err != nil {
		return fmt.Errorf("unable to set %s=%s: %s", key, value, err)
	}
	return f.marshal(vobj, in)
}

var funcs = map[string]interface{}{
	"jq": func(expr string, v interface{}) (interface{}, error) {
		op, err := jq.Parse(expr)
		if err != nil {
			return nil, errors.Wrap(err, "jq.Parse")
		}
		p, err := json.Marshal(v)
		if err != nil {
			return nil, errors.Wrap(err, "json.Marshal")
		}
		q, err := op.Apply(p)
		if err != nil {
			return nil, errors.Wrap(err, "op.Apply")
		}
		var vv interface{}
		if err := json.Unmarshal(q, &vv); err != nil {
			return nil, errors.Wrap(err, "json.Unmarshal")
		}
		return vv, nil
	},
}

func (f *Format) Template(tmpl, data, out string) error {
	p, err := f.read(tmpl)
	if err != nil {
		return err
	}
	t, err := template.New("").Funcs(funcs).Parse(string(p))
	if err != nil {
		return err
	}
	vdata, err := f.unmarshal(data)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vdata); err != nil {
		return err
	}
	return f.write(buf.Bytes(), out)
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
	if q, err := base64.StdEncoding.DecodeString(string(p)); err == nil {
		p = q
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

func Refmt(in, out string) error            { return f.Refmt(in, out) }
func Merge(orig, mixin, out string) error   { return f.Merge(orig, mixin, out) }
func DSN(dsn string) error                  { return f.DSN(dsn) }
func Set(in, key, value string) error       { return f.Set(in, key, value) }
func Template(tmpl, data, out string) error { return f.Template(tmpl, data, out) }
