/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cel

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"

	"jpbetz.github.com/celpatch/pkg/apply/cel/types"
)

type Merger interface {
	Merge(obj, patch, removals ref.Val) ref.Val
}

func Objects(merger Merger) cel.EnvOption {
	return cel.Lib(celObjects{merger: merger})
}

const (
	objectsNamespace = "objects"
	applyMacro       = "apply"
)

type celObjects struct {
	merger Merger
}

func (celObjects) LibraryName() string {
	return "cel.lib.ext.cel.bindings"
}

func ApplyStructType() *cel.Type {
	return cel.ObjectType("applystruct")
}

func (c celObjects) CompileOptions() []cel.EnvOption {
	paramTypeV := cel.TypeParamType("V")
	arrayStructType := ApplyStructType()
	return []cel.EnvOption{
		cel.Types(types.ApplyStructType),
		types.ApplyTypes(),
		cel.Macros(
			cel.NewReceiverMacro(applyMacro, 2, celApply),
		),
		cel.Function("apply_filter",
			cel.Overload("apply_filter_object_object", []*cel.Type{paramTypeV, arrayStructType}, paramTypeV,
				cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
					apply := rhs.(*types.ApplyStruct)
					return c.merger.Merge(lhs, apply.GetObject(), apply.GetRemovals())
				}))),
	}
}

func (celObjects) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{}
}

func macroTargetMatchesNamespace(ns string, target *exprpb.Expr) bool {
	switch target.GetExprKind().(type) {
	case *exprpb.Expr_IdentExpr:
		if target.GetIdentExpr().GetName() != ns {
			return false
		}
		return true
	default:
		return false
	}
}

func celApply(meh cel.MacroExprHelper, target *exprpb.Expr, args []*exprpb.Expr) (*exprpb.Expr, *common.Error) {
	if !macroTargetMatchesNamespace(objectsNamespace, target) {
		return nil, nil
	}
	object := args[0]
	varApplyConfig := args[1]
	switch varApplyConfig.GetExprKind().(type) {
	case *exprpb.Expr_StructExpr:
		removals := findRemovals(meh, varApplyConfig, "")

		return meh.GlobalCall("apply_filter",
			object,
			meh.NewObject("applystruct",
				meh.NewObjectFieldInit("object", varApplyConfig, false),
				meh.NewObjectFieldInit("removals", meh.NewList(removals...), false))), nil
	default:
		return nil, &common.Error{
			Message:  "objects.apply()'s second argument must be an object creation expression",
			Location: meh.OffsetLocation(varApplyConfig.GetId()),
		}
	}
}

// TODO: eliminate string concat with pathPrefix
func findRemovals(meh cel.MacroExprHelper, literal *exprpb.Expr, pathPrefix string) []*exprpb.Expr {
	var removals []*exprpb.Expr
	switch literal.GetExprKind().(type) {
	case *exprpb.Expr_StructExpr:
		st := literal.GetStructExpr() // map or object
		for _, e := range st.Entries {
			if e.OptionalEntry {
				switch e.Value.GetExprKind().(type) {
				case *exprpb.Expr_CallExpr:
					if e.Value.GetCallExpr().GetFunction() == "none" { // TODO: check target is "optional" as well
						if key, ok := getPathKey(e); ok {
							removals = append(removals, meh.LiteralString(pathPrefix+key))
						}
					}
				}
			}
			if key, ok := getPathKey(e); ok {
				removals = append(removals, findRemovals(meh, e.Value, pathPrefix+key+".")...)
			}
		}
	case *exprpb.Expr_ListExpr:
		// TODO: Support removals from listType=maps. Must identify the removal using the key fields.
	}

	return removals
}

func getPathKey(e *exprpb.Expr_CreateStruct_Entry) (string, bool) {
	switch e.GetKeyKind().(type) {
	case *exprpb.Expr_CreateStruct_Entry_FieldKey:
		return e.GetFieldKey(), true
	case *exprpb.Expr_CreateStruct_Entry_MapKey:
		switch v := e.GetMapKey().GetExprKind().(type) {
		case *exprpb.Expr_ConstExpr:
			switch vs := v.ConstExpr.GetConstantKind().(type) {
			case *exprpb.Constant_StringValue:
				return vs.StringValue, true
			}
		}
	}
	return "", false
}
