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
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	schema2 "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/kube-openapi/pkg/validation/spec"

	"sigs.k8s.io/yaml"
)

func TestMutateApply(t *testing.T) {
	testdata := "../../testdata"
	schema := loadTestYaml[spec.Schema](filepath.Join(testdata, "v1schema.yaml"))

	testDir := filepath.Join(testdata, "apply", "mutate")
	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Run(e.Name(), func(t *testing.T) {
				testCase := e.Name()
				original := loadTestYaml[any](filepath.Join(testDir, testCase, "original.yaml"))
				patch := loadTestYaml[any](filepath.Join(testDir, testCase, "patch.yaml"))
				expected := loadTestYaml[any](filepath.Join(testDir, testCase, "expected.yaml"))

				merged := MutateApply(&schema, original, patch)

				if !reflect.DeepEqual(expected, merged) {
					t.Errorf("Expected:\n%s\nBut got:\n%s\n", yamlToString(expected), yamlToString(merged))
				}
			})
		}
	}
}

func TestConvertApply(t *testing.T) {
	testdata := "../../testdata"
	v1schema := loadTestYaml[spec.Schema](filepath.Join(testdata, "v1schema.yaml"))
	v1Structural := loadStructural(filepath.Join(testdata, "v1schema.yaml"))
	v2schema := loadTestYaml[spec.Schema](filepath.Join(testdata, "v2schema.yaml"))
	v2Structural := loadStructural(filepath.Join(testdata, "v2schema.yaml"))

	testDir := filepath.Join(testdata, "templates", "convert")
	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Run(e.Name(), func(t *testing.T) {
				testCase := e.Name()
				original := loadTestYaml[any](filepath.Join(testDir, testCase, "original.yaml"))
				patch := loadTestYaml[any](filepath.Join(testDir, testCase, "v1tov2.yaml"))
				reversePatch := loadTestYaml[any](filepath.Join(testDir, testCase, "v2tov1.yaml"))
				expected := loadTestYaml[any](filepath.Join(testDir, testCase, "expected.yaml"))

				merged := ConvertWithTemplate(&v1schema, &v2schema, v2Structural, original, patch)

				if !reflect.DeepEqual(expected, merged) {
					t.Errorf("Expected:\n%s\nBut got:\n%s\n", yamlToString(expected), yamlToString(merged))
				}

				merged = ConvertWithTemplate(&v2schema, &v1schema, v1Structural, expected, reversePatch)

				if !reflect.DeepEqual(original, merged) {
					t.Errorf("Expected:\n%s\nBut got:\n%s\n", yamlToString(original), yamlToString(merged))
				}
			})
		}
	}
}

func TestMutateWithTemplates(t *testing.T) {
	testdata := "../../testdata"
	schema := loadTestYaml[spec.Schema](filepath.Join(testdata, "v1schema.yaml"))

	testDir := filepath.Join(testdata, "templates", "mutate")
	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Run(e.Name(), func(t *testing.T) {
				testCase := e.Name()
				original := loadTestYaml[any](filepath.Join(testDir, testCase, "original.yaml"))
				patch := loadTestYaml[any](filepath.Join(testDir, testCase, "patch.yaml"))
				expected := loadTestYaml[any](filepath.Join(testDir, testCase, "expected.yaml"))

				merged := MutateWithTemplate(&schema, original, patch)

				if !reflect.DeepEqual(expected, merged) {
					t.Errorf("Expected:\n%s\nBut got:\n%s\n", yamlToString(expected), yamlToString(merged))
				}
			})
		}
	}
}

func TestConvertWithTemplate(t *testing.T) {
	testdata := "../../testdata"
	v1schema := loadTestYaml[spec.Schema](filepath.Join(testdata, "v1schema.yaml"))
	v1Structural := loadStructural(filepath.Join(testdata, "v1schema.yaml"))
	v2schema := loadTestYaml[spec.Schema](filepath.Join(testdata, "v2schema.yaml"))
	v2Structural := loadStructural(filepath.Join(testdata, "v2schema.yaml"))

	testDir := filepath.Join(testdata, "apply", "convert")
	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Run(e.Name(), func(t *testing.T) {
				testCase := e.Name()
				original := loadTestYaml[any](filepath.Join(testDir, testCase, "original.yaml"))
				patch := loadTestYaml[any](filepath.Join(testDir, testCase, "v1tov2.yaml"))
				reversePatch := loadTestYaml[any](filepath.Join(testDir, testCase, "v2tov1.yaml"))
				expected := loadTestYaml[any](filepath.Join(testDir, testCase, "expected.yaml"))

				merged := ConvertApply(&v1schema, &v2schema, v2Structural, original, patch)

				if !reflect.DeepEqual(expected, merged) {
					t.Errorf("Expected:\n%s\nBut got:\n%s\n", yamlToString(expected), yamlToString(merged))
				}

				merged = ConvertApply(&v2schema, &v1schema, v1Structural, expected, reversePatch)

				if !reflect.DeepEqual(original, merged) {
					t.Errorf("Expected:\n%s\nBut got:\n%s\n", yamlToString(original), yamlToString(merged))
				}
			})
		}
	}
}

func yamlToString(obj any) string {
	out, err := yaml.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return string(out)
}

func loadTestYaml[T any](file string) T {
	var original T
	bytes, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	j, err := yaml.YAMLToJSON(bytes)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(j, &original)
	if err != nil {
		panic(err)
	}
	return original
}

func loadStructural(file string) *schema2.Structural {
	v2props := loadTestYaml[v1.JSONSchemaProps](file)

	var out apiextensions.JSONSchemaProps
	err := v1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(&v2props, &out, nil)
	if err != nil {
		panic(err)
	}
	structural, err := schema2.NewStructural(&out)
	if err != nil {
		panic(err)
	}
	return structural
}
