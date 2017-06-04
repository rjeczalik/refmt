# refmt [![GoDoc](https://godoc.org/github.com/rjeczalik/refmt?status.png)](https://godoc.org/github.com/rjeczalik/refmt) [![Build Status](https://img.shields.io/travis/rjeczalik/refmt/master.svg)](https://travis-ci.org/rjeczalik/refmt "linux_amd64") [![Build status](https://img.shields.io/appveyor/ci/rjeczalik/refmt.svg)](https://ci.appveyor.com/project/rjeczalik/refmt "windows_amd64")
Reformat HCL ⇄ JSON and HCL ⇄ YAML.

### install

```
$ go get github.com/rjeczalik/refmt
```

### usage

```
usage: refmt [-f format] INPUT_FILE|"-" OUTPUT_FILE|"-"

Converts from one encoding to another. Supported formats (and their file extensions):

        - HCL (.hcl)
        - JSON (.json)
        - YAML (.yaml or .yml)

If INPUT_FILE extension is not recognized or INPUT_FILE is "-" (stdin),
refmt will try to guess input format.

If OUTPUT_FILE is "-" (stdout), destination format is required to be
passed with -f flag.
```

### examples

```
$ refmt -t yaml main.yaml -
```
```yaml
provider:
  aws:
    access_key: ${var.aws_access_key}
    secret_key: ${var.aws_secret_key}
resource:
  aws_instance:
    aws-instance:
      instance_type: t2.nano
      user_data: echo "hello world!" >> /tmp/helloworld.txt
```
```
$ refmt main.yaml main.json
```
```json
{
        "provider": {
                "aws": {
                        "access_key": "${var.aws_access_key}",
                        "secret_key": "${var.aws_secret_key}"
                }
        },
        "resource": {
                "aws_instance": {
                        "aws-instance": {
                                "instance_type": "t2.nano",
                                "user_data": "echo \"hello world!\" >> /tmp/helloworld.txt"
                        }
                }
        }
}
```
```hcl
$ refmt main.json main.hcl
```
```
provider "aws" {
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
}

resource "aws_instance" "aws-instance" {
  instance_type = "t2.nano"
  user_data = "echo \"hello world!\" >> /tmp/helloworld.txt"
}
```

#### pretty reformat in-place

```
$ refmt main.tf.json main.tf.json
```

### todo

- inline docs
- fix hcl marshaling:
  - fix excessive newlines
  - fix excessive quotes
