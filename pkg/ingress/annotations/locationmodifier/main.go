/*
Licensed Materials - Property of IBM
cfc
@ Copyright IBM Corp. 2018 All Rights Reserved
US Government Users Restricted Rights - Use, duplication or disclosure
restricted by GSA ADP Schedule Contract with IBM Corp.
*/

package locationmodifier

import (
	extensions "k8s.io/api/extensions/v1beta1"

	"github.com/open-cluster-management/management-ingress/pkg/ingress/annotations/parser"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/errors"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/resolver"
)

type at struct {
	r resolver.Resolver
}

// NewParser creates a new secure upstream annotation parser
func NewParser(r resolver.Resolver) parser.IngressAnnotation {
	return at{r}
}

// Parse parses the annotations contained in the ingress
// rule used to indicate if the upstream servers should use SSL
func (a at) Parse(ing *extensions.Ingress) (interface{}, error) {
	ca, _ := parser.GetStringAnnotation("location-modifier", ing)
	if ca != "~" && ca != "=" && ca != "~*" {
		return "", errors.Errorf("Location modifier type %v is not supported", ca)
	}
	return ca, nil
}
