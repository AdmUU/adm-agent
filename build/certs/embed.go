package certs

import (
	_ "embed"
)

//go:embed client.pem
var ShareCertPem []byte 

//go:embed client.key
var ShareCertKey []byte 

func GetCertPem() []byte {
    return ShareCertPem
}

func GetCertKey() []byte {
    return ShareCertKey
}