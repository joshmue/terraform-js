package configs

import (
	"github.com/robertkrimen/otto"
	"io/ioutil"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

func execFile(path string) ([]*Resource, error) {
	vm := otto.New()
	_, err := vm.Eval(`
resources = [];
function make(rType, rName, rParams) {
  resources.push({
    type: rType,
    name: rName,
    params: rParams
  });
}`)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_, err = vm.Eval(string(data))
	if err != nil {
		return nil, err
	}
	val, err := vm.Get("resources")
	if err != nil {
		return nil, err
	}
	resourcesObj := val.Object()
	resources := []*Resource{}
	for _, key := range resourcesObj.Keys() {
		rVal, err := resourcesObj.Get(key)
		if err != nil {
			return nil, err
		}
		rObj := rVal.Object()
		rTypeVal, err := rObj.Get("type")
		if err != nil {
			return nil, err
		}
		rType, err := rTypeVal.ToString()
		if err != nil {
			return nil, err
		}
		rNameVal, err := rObj.Get("name")
		if err != nil {
			return nil, err
		}
		rName, err := rNameVal.ToString()
		if err != nil {
			return nil, err
		}
		rParamsVal, err := rObj.Get("params")
		if err != nil {
			return nil, err
		}
		rParams := rParamsVal.Object()
		params := map[string]string{}
		for _, pKey := range rParams.Keys() {
			pVal, err := rParams.Get(pKey)
			if err != nil {
				return nil, err
			}
			params[pKey], err = pVal.ToString()
			if err != nil {
				return nil, err
			}
		}
		resources = append(resources, makeResource(rType, rName, params))
	}
	return resources, nil
}

func makeResource(rType string, rName string, rParams map[string]string) *Resource {
	attrs := hclsyntax.Attributes{}
	for key, value := range rParams {
		attrs[key] = &hclsyntax.Attribute{
			Name: key,
			Expr: &hclsyntax.TemplateExpr{
				Parts: []hclsyntax.Expression{
					&hclsyntax.LiteralValueExpr{
						Val: cty.StringVal(value),
					},
				},
			},
		}
	}
	return &Resource{
		Mode: addrs.ManagedResourceMode,
		Name: rName,
		Type: rType,
		Config: &hclsyntax.Body{
			Attributes: attrs,
		},
		Managed: &ManagedResource{

		},
	}
}
