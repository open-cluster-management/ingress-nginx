/*
Licensed Materials - Property of IBM
cfc
@ Copyright IBM Corp. 2018 All Rights Reserved
US Government Users Restricted Rights - Use, duplication or disclosure
restricted by GSA ADP Schedule Contract with IBM Corp.
*/

package secureupstream

import (
	"fmt"

	"github.com/pkg/errors"
	extensions "k8s.io/api/extensions/v1beta1"

	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations/parser"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/resolver"
)

// Config describes SSL backend configuration
type Config struct {
	Secure       bool                 `json:"secure"`
	CACert       resolver.AuthSSLCert `json:"caCert"`
	ClientCACert resolver.AuthSSLCert `json:"clientCACert"`
}

type su struct {
	r resolver.Resolver
}

// NewParser creates a new secure upstream annotation parser
func NewParser(r resolver.Resolver) parser.IngressAnnotation {
	return su{r}
}

// Parse parses the annotations contained in the ingress
// rule used to indicate if the upstream servers should use SSL
func (a su) Parse(ing *extensions.Ingress) (interface{}, error) {
	s, _ := parser.GetBoolAnnotation("secure-backends", ing)
	ca, _ := parser.GetStringAnnotation("secure-verify-ca-secret", ing)
	clientca, _ := parser.GetStringAnnotation("secure-client-ca-secret", ing)

	_caCert := resolver.AuthSSLCert{}
	_clientCACert := resolver.AuthSSLCert{}

	secure := &Config{
		Secure:       s,
		CACert:       _caCert,
		ClientCACert: _clientCACert,
	}
	if !s && ca != "" {
		return secure,
			errors.Errorf("trying to use CA from secret %v/%v on a non secure backend", ing.Namespace, ca)
	}
	if !s && clientca != "" {
		return secure,
			errors.Errorf("trying to use Client CA from secret %v/%v on a non secure backend", ing.Namespace, clientca)
	}

	if ca != "" {
		caCert, err := a.r.GetAuthCertificate(fmt.Sprintf("%v/%v", ing.Namespace, ca))
		if err != nil {
			return secure, errors.Wrap(err, "error obtaining certificate")
		}
		if caCert != nil {
			_caCert = *caCert
		}
	}

	if clientca != "" {
		caCert, err := a.r.GetAuthCertificate(fmt.Sprintf("%v/%v", ing.Namespace, clientca))
		if err != nil {
			return secure, errors.Wrap(err, "error obtaining client certificate")
		}
		if caCert != nil {
			_clientCACert = *caCert
		}
	}
	return &Config{
		Secure:       s,
		CACert:       _caCert,
		ClientCACert: _clientCACert,
	}, nil
}
