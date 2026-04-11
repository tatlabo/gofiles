package cred

import _ "embed"

//go:embed cert.pem
var Cert string

//go:embed key.pem
var Key string
