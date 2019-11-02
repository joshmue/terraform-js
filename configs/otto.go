package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/addrs"
	"github.com/robertkrimen/otto"
	"github.com/zclconf/go-cty/cty"
	"strings"
)

func (p *Parser) execFile(path string) ([]*Resource, error) {
	vm := otto.New()
	_, err := vm.Eval(`
resources = [];
function make(rType, rName, rParams, rDeps) {
  resources.push({
    type: rType,
    name: rName,
    params: rParams || {},
    deps: rDeps || []
  });
}`)
	if err != nil {
		return nil, err
	}
	data, err := p.fs.ReadFile(path)
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
		rDepsVal, err := rObj.Get("deps")
		if err != nil {
			return nil, err
		}
		rDeps := rDepsVal.Object()
		deps := []string{}
		for _, dKey := range rDeps.Keys() {
			dVal, err := rDeps.Get(dKey)
			if err != nil {
				return nil, err
			}
			dep, err := dVal.ToString()
			if err != nil {
				return nil, err
			}
			deps = append(deps, dep)
		}
		resources = append(resources, makeResource(rType, rName, params, deps))
	}
	return resources, nil
}

func makeResource(rType string, rName string, rParams map[string]string, deps []string) *Resource {
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
	travs := []hcl.Traversal{}
	for _, dep := range deps {
		trav := hcl.Traversal{}
		pos := 0
		for _, part := range strings.Split(dep, ".") {
			if pos == 0 {
				trav = append(trav, hcl.TraverseRoot{
					Name: part,
					SrcRange: hcl.Range{
						Start: hcl.Pos{
							Line:   1,
							Column: pos + 1,
							Byte:   pos,
						},
						End: hcl.Pos{
							Line:   1,
							Column: pos + len(part) + 1,
							Byte:   pos + len(part),
						},
					},
				})
			} else {
				trav = append(trav, hcl.TraverseAttr{
					Name: part,
					SrcRange: hcl.Range{
						Start: hcl.Pos{
							Line:   1,
							Column: pos + 1,
							Byte:   pos,
						},
						End: hcl.Pos{
							Line:   1,
							Column: pos + len(part) + 1,
							Byte:   pos + len(part),
						},
					},
				})
			}
			pos += len(part)
		}
		travs = append(travs, trav)
	}
	return &Resource{
		Mode:      addrs.ManagedResourceMode,
		Name:      rName,
		Type:      rType,
		DependsOn: travs,
		Config: &hclsyntax.Body{
			Attributes: attrs,
		},
		Managed: &ManagedResource{},
	}
}
