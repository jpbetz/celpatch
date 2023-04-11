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
	"sort"

	"github.com/golang/protobuf/ptypes/duration"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
)

// Prune is equivalent to
// PruneWithOptions(obj, s, isResourceRoot, structuralschema.UnknownFieldPathOptions{})
func Prune(obj interface{}, s *structuralschema.Structural, isResourceRoot bool) {
	PruneWithOptions(obj, s, isResourceRoot, structuralschema.UnknownFieldPathOptions{})
}

func PruneWithOptions(obj interface{}, s *structuralschema.Structural, isResourceRoot bool, opts structuralschema.UnknownFieldPathOptions) []string {
	if isResourceRoot {
		if s == nil {
			s = &structuralschema.Structural{}
		}
		if !s.XEmbeddedResource {
			clone := *s
			clone.XEmbeddedResource = true
			s = &clone
		}
	}
	prune(obj, s, &opts)
	sort.Strings(opts.UnknownFieldPaths)
	return opts.UnknownFieldPaths
}

var metaFields = map[string]bool{
	"apiVersion": true,
	"kind":       true,
	"metadata":   true,
}

// This has been hacked to also prune fields with wrong types and listType=map entries missing map keys.
func prune(x interface{}, s *structuralschema.Structural, opts *structuralschema.UnknownFieldPathOptions) any {
	if s != nil && s.XPreserveUnknownFields {
		skipPrune(x, s, opts)
		return x
	}

	origPathLen := len(opts.ParentPath)
	defer func() {
		opts.ParentPath = opts.ParentPath[:origPathLen]
	}()
	switch x := x.(type) {
	case map[string]interface{}:
		if s == nil {
			for k := range x {
				opts.RecordUnknownField(k)
				delete(x, k)
			}
			return x
		}
		for k, v := range x {
			if s.XEmbeddedResource && metaFields[k] {
				continue
			}
			prop, ok := s.Properties[k]
			if ok {
				opts.AppendKey(k)
				x[k] = prune(v, &prop, opts)
				opts.ParentPath = opts.ParentPath[:origPathLen]
			} else if s.AdditionalProperties != nil {
				opts.AppendKey(k)
				x[k] = prune(v, s.AdditionalProperties.Structural, opts)
				opts.ParentPath = opts.ParentPath[:origPathLen]
			} else {
				if !metaFields[k] || len(opts.ParentPath) > 0 {
					opts.RecordUnknownField(k)
				}
				delete(x, k)
			}
		}
		return x
	case []interface{}:
		replacement := make([]interface{}, 0, len(x))
		if s == nil {
			for i, v := range x {
				opts.AppendIndex(i)
				replacement = append(replacement, prune(v, nil, opts))
				opts.ParentPath = opts.ParentPath[:origPathLen]
			}
			return replacement
		}

		for i, v := range x {
			if s.XListType != nil && *s.XListType == "map" {
				skipped := false
				m, ok := v.(map[string]interface{})
				if !ok {
					skipped = true
				} else {
					for _, k := range s.XListMapKeys {
						if _, found := m[k]; !found {
							skipped = true
							break
						}
					}
				}
				if skipped {
					opts.AppendIndex(i)
					//replacement = append(replacement, prune(v, nil, opts))
					opts.ParentPath = opts.ParentPath[:origPathLen]
					continue
				}
			}
			opts.AppendIndex(i)
			replacement = append(replacement, prune(v, s.Items, opts))
			opts.ParentPath = opts.ParentPath[:origPathLen]
		}
		return replacement
	default:
		// scalars, do nothing
		if s != nil {
			// TODO: handle all cases and properly drop fields
			switch x.(type) {
			case int, int32, int64:
				if s.Type != "integer" {
					return nil
				}
			case float32, float64:
				if s.Type != "number" {
					return nil
				}
			case string:
				if s.Type != "string" {
					return nil
				}
			case duration.Duration:
				if s.Type != "string" || s.ValueValidation.Format != "duration" {
					return nil
				}
			}
		}
		return x
	}
}

func skipPrune(x interface{}, s *structuralschema.Structural, opts *structuralschema.UnknownFieldPathOptions) {
	if s == nil {
		return
	}
	origPathLen := len(opts.ParentPath)
	defer func() {
		opts.ParentPath = opts.ParentPath[:origPathLen]
	}()

	switch x := x.(type) {
	case map[string]interface{}:
		for k, v := range x {
			if s.XEmbeddedResource && metaFields[k] {
				continue
			}
			if prop, ok := s.Properties[k]; ok {
				opts.AppendKey(k)
				prune(v, &prop, opts)
				opts.ParentPath = opts.ParentPath[:origPathLen]
			} else if s.AdditionalProperties != nil {
				opts.AppendKey(k)
				prune(v, s.AdditionalProperties.Structural, opts)
				opts.ParentPath = opts.ParentPath[:origPathLen]
			}
		}
	case []interface{}:
		for i, v := range x {
			opts.AppendIndex(i)
			skipPrune(v, s.Items, opts)
			opts.ParentPath = opts.ParentPath[:origPathLen]
		}
	default:
		// scalars, do nothing
	}
}
