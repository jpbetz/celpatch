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

package apply

import (
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/interpreter"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/cel/common"
	"k8s.io/apiserver/pkg/cel/library"
	"k8s.io/apiserver/pkg/cel/openapi"
	"k8s.io/kube-openapi/pkg/schemaconv"
	"k8s.io/kube-openapi/pkg/validation/spec"
	smdschema "sigs.k8s.io/structured-merge-diff/v4/schema"
	"sigs.k8s.io/structured-merge-diff/v4/typed"

	cel2 "jpbetz.github.com/celpatch/pkg/apply/cel"
)

const (
	templateVar       = "$"
	objectTypeName    = "Object" // com.example.group.v1.Example if we want to fully qualify
	oldObjectTypeName = "OldObject"
	oldSelfVar        = "oldSelf"
	oldObjectVar      = "oldObject"
)

// ConvertWithTemplate performs a version conversion using the patch.
// TODO: Remove schema.Structural from arguments and introduce a more efficient alternative to the prune
// operation.
func ConvertWithTemplate(fromVersionSchema, toVersionSchema *spec.Schema, toVersionStructuralSchema *schema.Structural, fromObject, patch any) any {
	oldOpenAPISchema := &openapi.Schema{Schema: fromVersionSchema}
	newOpenAPISchema := &openapi.Schema{Schema: toVersionSchema}
	// Conversion Flow:
	// 1. Do template variable substitution
	ac := Substitute(oldOpenAPISchema, newOpenAPISchema, fromObject, patch, true)
	// 2. Start converting the v1 object to v2 and pruning: (a) any fields not in v2,
	//    (b) any fields with incorrect types (c) any listType=map entries with missing keys.
	// TODO: This prune is probably better handled by checking differences between schemas
	// and only keeping what is compatible.
	pruned := runtime.DeepCopyJSON(fromObject.(map[string]any))
	Prune(pruned, toVersionStructuralSchema, true)
	// 3. Merge the patch with the pruned object
	return Merge(toVersionSchema, pruned, ac, true)
}

func ConvertApply(fromVersionSchema, toVersionSchema *spec.Schema, toVersionStructuralSchema *schema.Structural, fromObject, patch any) any {
	oldOpenAPISchema := &openapi.Schema{Schema: fromVersionSchema}
	newOpenAPISchema := &openapi.Schema{Schema: toVersionSchema}
	// Conversion Flow:
	// 1. Build the apply configuration
	expression := patch.(map[string]any)["mutation"].(string)
	ac := Eval(oldOpenAPISchema, newOpenAPISchema, fromObject, expression, true)
	// 2. Start converting the v1 object to v2 and pruning: (a) any fields not in v2,
	//    (b) any fields with incorrect types (c) any listType=map entries with missing keys.
	// TODO: This prune is probably better handled by checking differences between schemas
	// and only keeping what is compatible.
	pruned := runtime.DeepCopyJSON(fromObject.(map[string]any))
	Prune(pruned, toVersionStructuralSchema, true)
	// 3. Merge the patch with the pruned object
	return Merge(toVersionSchema, pruned, ac, true)
}

// MutateWithTemplate applies the patch to the object.
func MutateWithTemplate(schema *spec.Schema, obj, patch any) any {
	s := &openapi.Schema{Schema: schema}
	applyConfiguration := Substitute(s, s, obj, patch, false)
	return Merge(schema, obj, applyConfiguration, false)
}

func MutateApply(schema *spec.Schema, obj any, patch any) any {
	expression := patch.(map[string]any)["mutation"].(string)
	openAPISchema := &openapi.Schema{Schema: schema}
	applyConfiguration := Eval(openAPISchema, openAPISchema, obj, expression, false)
	return Merge(schema, obj, applyConfiguration, false)
}

func MutateDirect(schema *spec.Schema, obj any, patch any) any {
	expression := patch.(map[string]any)["mutation"].(string)
	openAPISchema := &openapi.Schema{Schema: schema}
	result := Eval(openAPISchema, openAPISchema, obj, expression, false)
	return result
}

// Merge performs a server side apply style merge of the patch (apply configuration) to the
// obj.  The schema of the object is also required. If preserveUnknownFields is true, the
// patch may add unrecognized fields, otherwise adding unrecognized fields will result in an error.
func Merge(s *spec.Schema, obj, patch any, preserveUnknownFields bool) any {
	specSchema, err := schemaconv.ToSchemaFromOpenAPI(map[string]*spec.Schema{"root": s}, preserveUnknownFields)
	if err != nil {
		panic(err)
	}
	parser := typed.Parser{Schema: smdschema.Schema{Types: specSchema.Types}}
	t := parser.Type("root")
	objT, err := t.FromUnstructured(obj)
	if err != nil {
		panic(err)
	}
	patchT, err := t.FromUnstructured(patch)
	if err != nil {
		panic(err)
	}
	result, err := objT.Merge(patchT)
	if err != nil {
		panic(err)
	}
	return result.AsValue().Unstructured()
}

// Substitute applies template variable substitution to the patch. All `{$: "<CEL expression>}`
// directives will be evaluated and the result will be inlined into the patch.
// obj provides the "oldObject" variable that is accessible in CEL expressions.
// oldObjectSchema provides the schema of obj, and patchSchema provides the schema of the patch.
// These schemas may be the same (e.g. for mutating admission) or may differ (e.g. for CRD conversion).
func Substitute(oldObjectSchema, patchSchema common.Schema, obj, patch any, isConversion bool) any {
	a := &applier{patchSchema: patchSchema, oldObjectSchema: oldObjectSchema, oldObject: obj, isConvertion: isConversion}
	return a.applyTemplate(patchSchema, patch, obj)
}

func Eval(oldObjectSchema, patchSchema common.Schema, obj any, expression string, isConversion bool) any {
	a := &applier{patchSchema: patchSchema, oldObjectSchema: oldObjectSchema, oldObject: obj, isConvertion: isConversion}
	return a.evaluateSubstitution(nil, nil, expression, isConversion)
}

type applier struct {
	patchSchema     common.Schema
	oldObjectSchema common.Schema
	oldObject       any
	isConvertion    bool
}

// applyTemplate applies any template substitutions at the current schema level
// and then traverses to the next level of schema depth, if any.
func (a *applier) applyTemplate(schema common.Schema, patchValue, oldValue any) any {
	if m, ok := patchValue.(map[string]any); ok {
		if v, ok := m[templateVar]; ok {
			return a.evaluateSubstitution(schema, oldValue, v.(string), a.isConvertion)
		}
	}
	if schema.Properties() != nil {
		m, ok := patchValue.(map[string]any)
		if !ok {
			panic("expected map")
		}
		objM, _ := oldValue.(map[string]any)

		result := map[string]any{}
		for fieldName, propSchema := range schema.Properties() {
			if v, ok := m[fieldName]; ok {
				var objField any
				if objM != nil {
					objField = objM[fieldName]
				}
				result[fieldName] = a.applyTemplate(propSchema, v, objField)
			}
		}
		return result
	} else if schema.AdditionalProperties() != nil {
		m, ok := patchValue.(map[string]any)
		if !ok {
			panic("expected map")
		}
		objM, _ := oldValue.(map[string]any)

		schema := schema.AdditionalProperties().Schema()
		result := map[string]any{}
		for k, v := range m {
			var objField any
			if objM != nil {
				objField = objM[k]
			}
			result[k] = a.applyTemplate(schema, v, objField)
		}
		return result
	} else if schema.Items() != nil {
		l, ok := patchValue.([]any)
		if !ok {
			panic("expected slice")
		}

		result := make([]any, len(l))
		for i, el := range l {
			var objEl any
			// TODO: correlate

			result[i] = a.applyTemplate(schema.Items(), el, objEl)
		}
		return result
	} else {
		return patchValue
	}
}

// TODO: This is not right. The schema needs to be the right one for whatever object "apply()"
// was called on, which is not necessarily the root schema.
type merger struct {
	schema        *spec.Schema
	openAPISchema common.Schema
}

func (m *merger) Merge(obj, patch, removals ref.Val) ref.Val {
	objVal := valueToUnstructured(obj)
	patchval := valueToUnstructured(patch)

	result := Merge(m.schema, objVal, patchval, false)
	resultObj := result.(map[string]any)
	iter := removals.(traits.Iterable).Iterator()
	for iter.HasNext() == types.True {
		removal := iter.Next().Value().(string)
		delete(resultObj, removal)
	}
	return common.UnstructuredToVal(resultObj, m.openAPISchema)
}

// evaluateSubstitution a template variable substitution CEL expression.
func (a *applier) evaluateSubstitution(selfSchema common.Schema, self any, expression string, isConversion bool) any {
	//selfDecl := common.SchemaDeclType(selfSchema, selfSchema == a.rootSchema)
	//selfVal := common.UnstructuredToVal(self, selfSchema)

	objVal := common.UnstructuredToVal(a.oldObject, a.oldObjectSchema)

	m := &merger{schema: a.oldObjectSchema.(*openapi.Schema).Schema, openAPISchema: a.oldObjectSchema}
	baseEnv, err := buildBaseEnv(m)
	if err != nil {
		panic(err)
	}

	var rt *common.OpenAPITypeProvider
	var oldObjectCelType *cel.Type
	if isConversion {
		patchDecl := common.SchemaDeclType(a.patchSchema, true).MaybeAssignTypeName(objectTypeName)
		oldObjectDecl := common.SchemaDeclType(a.oldObjectSchema, true).MaybeAssignTypeName(oldObjectTypeName)
		rt, err = common.NewOpenAPITypeProvider(patchDecl, oldObjectDecl)
		if err != nil {
			panic(err)
		}
		oldObjectCelType = oldObjectDecl.CelType()
	} else {
		objectDecl := common.SchemaDeclType(a.patchSchema, true).MaybeAssignTypeName(objectTypeName)
		rt, err = common.NewOpenAPITypeProvider(objectDecl)
		if err != nil {
			panic(err)
		}
		oldObjectCelType = objectDecl.CelType()
	}

	opts, err := rt.EnvOptions(baseEnv.TypeProvider())
	if err != nil {
		panic(err)
	}
	opts = append(opts,
		//cel.Variable(oldSelfVar, selfDecl.CelType()),
		cel.Variable(oldObjectVar, oldObjectCelType),
	)
	env, err := baseEnv.Extend(opts...)
	if err != nil {
		panic(err)
	}
	ast, issues := env.Compile(expression)
	if issues != nil {
		panic(issues)
	}
	// TODO: check return type matches schema type
	prog, err := env.Program(ast)
	if err != nil {
		panic(err)
	}
	activation := &evaluationActivation{self: nil, object: objVal}
	v, _, err := prog.Eval(activation)
	if err != nil {
		panic(err)
	}
	return valueToUnstructured(v)
}

// valueToUnstructured strips away all ref.Val and replaces them with unstructured equivalents.
func valueToUnstructured(o any) any {
	// TODO: this is a mess. Essentially, I need a way to convert data back out of CEL and
	// because of data literals, the data can be a mix of ref.Vals and Go scalars...
	switch a := o.(type) {
	case ref.Val:
		switch t := a.Value().(type) {
		case map[ref.Val]ref.Val:
			result := make(map[string]any, len(t))
			for k, v := range t {
				result[k.Value().(string)] = valueToUnstructured(v)
			}
			return result
		case []ref.Val:
			result := make([]any, len(t))
			for i, e := range t {
				result[i] = valueToUnstructured(e)
			}
			return result
		case time.Duration:
			return t.String() // TODO: what other types should be handled here?
		default:
			return valueToUnstructured(t)
		}
	case map[string]any:
		result := make(map[string]any, len(a))
		for k, v := range a {
			result[k] = valueToUnstructured(v)
		}
		return result
	case []any:
		result := make([]any, len(a))
		for i, e := range a {
			result[i] = valueToUnstructured(e)
		}
		return result
	case time.Duration:
		return a.String()
	}
	return o
}

func baseOpts(merger cel2.Merger) []cel.EnvOption {
	var opts []cel.EnvOption
	opts = append(opts, cel.HomogeneousAggregateLiterals())
	// Validate function declarations once during base env initialization,
	// so they don't need to be evaluated each time a CEL rule is compiled.
	// This is a relatively expensive operation.
	//opts = append(opts, cel.EagerlyValidateDeclarations(true)) // TODO: enabble when I figure what is going wrong with optional types
	opts = append(opts, cel.DefaultUTCTimeZone(true))
	opts = append(opts, cel.OptionalTypes())
	opts = append(opts, cel2.Objects(merger))
	opts = append(opts, library.ExtensionLibs...)
	return opts
}
func buildBaseEnv(merger cel2.Merger) (*cel.Env, error) {
	return cel.NewEnv(baseOpts(merger)...)
}

type evaluationActivation struct {
	object, self any
}

// ResolveName returns a value from the activation by qualified name, or false if the name
// could not be found.
func (a *evaluationActivation) ResolveName(name string) (interface{}, bool) {
	switch name {
	case oldSelfVar:
		return a.self, true
	case oldObjectVar:
		return a.object, true
	default:
		return nil, false
	}
}

func (a *evaluationActivation) Parent() interpreter.Activation {
	return nil
}
