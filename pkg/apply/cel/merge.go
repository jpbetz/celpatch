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
			// cel.bind(<object>, <patchObject>)
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
		creator := varApplyConfig.GetStructExpr()
		var removals []*exprpb.Expr
		for _, e := range creator.Entries {
			if e.OptionalEntry {
				switch e.Value.GetExprKind().(type) {
				case *exprpb.Expr_CallExpr:
					if e.Value.GetCallExpr().GetFunction() == "none" { // TODO: check target is "optional" as well
						removals = append(removals, meh.LiteralString(e.GetFieldKey()))
					}
				}
			}
		}

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
