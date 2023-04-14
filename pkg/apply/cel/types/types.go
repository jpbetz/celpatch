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

package types

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

var (
	// ApplyStructType indicates the runtime type of an optional value.
	ApplyStructType = types.NewTypeValue("applystruct")
)

// ApplyStruct value which points to a value if non-empty.
type ApplyStruct struct {
	object   ref.Val
	removals ref.Val
}

func (o *ApplyStruct) GetObject() ref.Val {
	return o.object
}

func (o *ApplyStruct) GetRemovals() ref.Val {
	return o.removals
}

// ConvertToNative implements the ref.Val interface method.
func (o *ApplyStruct) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return o.object.ConvertToNative(typeDesc)
}

// ConvertToType implements the ref.Val interface method.
func (o *ApplyStruct) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case ApplyStructType:
		return o
	case types.TypeType:
		return ApplyStructType
	}
	return types.NewErr("type conversion error from '%s' to '%s'", ApplyStructType, typeVal)
}

// Equal determines whether the values contained by two ApplyStruct values are equal.
func (o *ApplyStruct) Equal(other ref.Val) ref.Val {
	otherOpt, isOpt := other.(*ApplyStruct)
	if !isOpt {
		return types.False
	}
	if o.object.Equal(otherOpt.object) == types.True { // TODO: check removals..
		return types.True
	}
	return types.False
}

func (o *ApplyStruct) String() string {
	return fmt.Sprintf("ApplyStruct(object: %v, removals: %v)", o.object, o.removals)
}

// Type implements the ref.Val interface method.
func (o *ApplyStruct) Type() ref.Type {
	return ApplyStructType
}

// Value returns the underlying 'Value()' of the wrapped value, if present.
func (o *ApplyStruct) Value() any {
	return o.object.Value() // TODO
}

func newTypeProvider(ta ref.TypeAdapter, tp ref.TypeProvider) *applyTypeProvider {
	return &applyTypeProvider{ta, tp}
}

func ApplyTypes() cel.EnvOption {
	return func(env *cel.Env) (*cel.Env, error) {
		tp := newTypeProvider(env.TypeAdapter(), env.TypeProvider())
		env, err := cel.CustomTypeAdapter(tp)(env)
		if err != nil {
			return nil, err
		}
		return cel.CustomTypeProvider(tp)(env)
	}
}

type applyTypeProvider struct {
	baseAdapter  ref.TypeAdapter
	baseProvider ref.TypeProvider
}

// EnumValue proxies to the ref.TypeProvider configured at the times the NativeTypes
// option was configured.
func (tp *applyTypeProvider) EnumValue(enumName string) ref.Val {
	return tp.baseProvider.EnumValue(enumName)
}

// FindIdent looks up natives type instances by qualified identifier, and if not found
// proxies to the composed ref.TypeProvider.
func (tp *applyTypeProvider) FindIdent(typeName string) (ref.Val, bool) {
	return tp.baseProvider.FindIdent(typeName)
}

// FindType looks up CEL type-checker type definition by qualified identifier, and if not found
// proxies to the composed ref.TypeProvider.
func (tp *applyTypeProvider) FindType(typeName string) (*exprpb.Type, bool) {
	if typeName == ApplyStructType.TypeName() {
		return decls.NewTypeType(decls.NewObjectType(typeName)), true
	}
	return tp.baseProvider.FindType(typeName)
}

// FindFieldType looks up a native type's field definition, and if the type name is not a native
// type then proxies to the composed ref.TypeProvider
func (tp *applyTypeProvider) FindFieldType(typeName, fieldName string) (*ref.FieldType, bool) {
	if typeName == ApplyStructType.TypeName() {
		if fieldName == "object" {
			return &ref.FieldType{
				Type: decls.NewTypeParamType("K"),
				IsSet: func(obj any) bool {
					return true // TODO
				},
				GetFrom: func(obj any) (any, error) {
					return nil, nil // TODO
				},
			}, true
		}
		if fieldName == "removals" {
			return &ref.FieldType{
				Type: decls.NewListType(decls.String),
				IsSet: func(obj any) bool {
					return true // TODO
				},
				GetFrom: func(obj any) (any, error) {
					return nil, nil // TODO
				},
			}, true
		}
	}
	return tp.baseProvider.FindFieldType(typeName, fieldName)
}

// NewValue implements the ref.TypeProvider interface method.
func (tp *applyTypeProvider) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	if typeName != ApplyStructType.TypeName() {
		return tp.baseProvider.NewValue(typeName, fields)
	}
	return &ApplyStruct{fields["object"], fields["removals"]}
}

func (tp *applyTypeProvider) NativeToValue(val any) ref.Val {
	// TODO
	return tp.baseAdapter.NativeToValue(val)
}
