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

The following YAML creates an apply configuration:

```yaml
spec:
  deploymentName: {$: "object.metadata.name + '-deployment'"}
  list: {$: "self.filter(e, e != 'b')"}
  listMap:
    - key: "k2"
      value: {$: "string(4 * 50)"}
```

Which results in:

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

Note that merge rules such as `x-kubernetes-list-type: map` are respected.

CEL object construction may be useful in some cases. E.g.:

```yaml
spec:
  listMap: {$: "[Example.spec.listMap.item{key: 'first', value: 'prepended'}] + object.spec.listMap"}
```

CEL object construction is not supported in Kubernetes today, so we would need to define
how object types are represented. This example uses the Kind name as the root type and then
uses the path in the OpenAPIv3 schema to identify nested types.

Many simple mutations can be written this way without any template variables. For example, it is
trivial to set a label or inject a basic sidecar.

The use of `{$: "<cel expression>"}` is just an example. Template substitution can be
declared with any marker key we want. We don't necessarily need to use "$".

TODO:

- [ ] Figure out how to best prepend initContainers (append is easy)
- [ ] Do we need `{$if: "<condition>", $then: <more YAML>}` ? Note that `{$: "<condition> ? <cel data literal> : {} "}` is possible.
- [ ] Do we need looping constructs? `{$: "self.map(x, {somefield: x.something}) "}` is possible.

Mutation cases tested:

- [ ] Sidecar injection
- [ ] Auto-population of fields (AlwaysPullImages)
- [ ] Inject environment variable
- [ ] Modify args
- [ ] Inject readiness/liveness probes
- [ ] Clear a field
- [x] Inject labels/annotations

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