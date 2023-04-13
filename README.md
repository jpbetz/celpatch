Proof of concept: Use CEL templates to mutate Kubernetes resources.
=========

This uses CEL expressions embedded in YAML to create an partial object
which is then merged into a Kubernetes resource using the same "merge" operation used
by server side apply.

For example, given an original object:

```yaml
apiVersion: group.example.com/v1
kind: Example
metadata:
  name: "alpha"
spec:
  replicas: 1
  list:
    - "a"
    - "b"
  listMap:
    - key: "k1"
      value: "1"
    - key: "k2"
      value: "2"
status:
  availableReplicas: 0
```

The following CEL expression creates an apply configuration:

```cel
Object{
  spec: Object.spec{
    deploymentName: oldObject.metadata.name + '-deployment',
    list: oldObject.spec.list.filter(e, e != 'b'),
    listMap: [
      Object.spec.listMap.item{
        key: "k2",
        value: string(4 * 50)
      }
    ]
  }
}
```

Which merges into the original object resulting in:

```yaml
apiVersion: group.example.com/v1
kind: Example
metadata:
  name: alpha
spec:
  deploymentName: alpha-deployment
  list:
    - a
  listMap:
    - key: k1
      value: "1"
    - key: k2
      value: "200"
  replicas: 1
status:
  availableReplicas: 0
```

Note that Kubernetes merge rules such as `x-kubernetes-list-type: map` are respected.

CEL has always supported object construction like the `Object{}` in the above examples.
However, we have not enabled CEL object construction is Kubernetes for validation features, so we
would need to enable it and define how type names are represented in CEL for OpenAPI. This example
uses "Object" name as the root type and then uses the path in the OpenAPIv3 schema to identify nested
types.

This repo also contains examples that perform version conversion and that use different approaches.

For example, the `mutate-templates` directory shows an approach where templates containing CEL
expressions are embedded in YAML.

TODO:

- [ ] Try adding a "merge" function to CEL that calls SSA merge directly.
- [ ] Experiment with Guided APIs, in particular, using field paths to specify which field to modify.

Mutation cases tested:

- [ ] Sidecar injection
- [ ] Auto-population of fields (AlwaysPullImages)
- [ ] Inject environment variable
- [ ] Modify args
- [ ] Inject readiness/liveness probes
- [ ] Clear a field
- [x] Inject labels/annotations
- [ ] Add if not present
- 

Conversion cases tested:

- [x] Rename a field
- [x] Change type of a field
- [ ] Move a field to a different location in the object
- [ ] Move field from annotations to object
- [x] Re-key a listType=map
- [ ] Split a field into two fields (e.g. field="a/b" becomes field1="a", field2="b")
- [ ] Convert from scalar field to a list of scalars, (e.g. field="a" becomes field1=["a"])
- [ ] Conditionally convert (if apply the transformation, else do nothing)
- [ ] Complex type instantiation (e.g. spec.x,spec.y,spec.z becomes spec.subobj.x, spec.subobj.y, spec.subobj.z)
- [ ] convert from an annotation to a field?
- [ ] Deletions (unsetting a object property, removing a list element or a map entry)
- [ ] Apply a transform to all the objects in a list (e.g. rename a field in a list)