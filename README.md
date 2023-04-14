Proof of concept: Use CEL templates to mutate Kubernetes resources.
=========

This uses CEL expressions to create "apply configurations" of objects,
which are then merged into a Kubernetes resource using the "merge" operation of
server side apply.

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

Note that because a server side apply merging is used,
schema merge directives added to OpenAPI such as `x-kubernetes-list-type: map` are respected.

Because there is no field manager used for the merge, the merge is applied as if a never-before-used
field manager is used to perform the apply. This means that we must do something special to make it
possible to unset fields.

We will use CEL's optional type feature:

```
Object{
    spec: Object.spec{
        ?fieldToRemove: optional.none() # plz remove
    }
}
```

Sometimes the current state of the object will be needed. This is available via the
`oldObject` variable. For example, to update all containers in a pod to use the "Always"
imagePullPolicy:

```
Object{
    spec: Object.spec{
        containers: oldObject.spec.containers.map(c,
            Object.spec.containers.item{
                name: c.name,
                imagePullPolicy: "Always"
            })
        )
    }
}
```

On rare occasions, it may be necessary to perform apply directly on part of an object. For example,
imagine that the field "widgets" is a list of objects, but the field is not a listType=map, 
and so there is no way to merge in an added field to each widget using server side apply
directly like was done with the above imagePullPolicy example.

A workaround is to recreate all the widgets list, but use apply() on each widget to merge
in a single field change:

```
Object{
    spec: Object.spec{
        containers: oldObject.spec.widgets.map(oldWidget,
            objects.apply(oldWidget, Object.spec.widgets.item{
                part: "xyz"
            })
        )
    }
}
```

Note that such fields are very rare in the Kubernetes API since they don't work well with server
side apply. But this may come in handy with CRDs when the CRD author fails to use
`x-kubernetes-list-type: map`.

Notes
-----

CEL has always supported object construction like the `Object{}` expression in the above examples.
However, we do not need CEL object construction is Kubernetes for validation features, so we had
never implemented the `NewValue` function for Kubernetes types that is required for CEL expressions
to actually use object construction. Would need to enable it and define how type names are represented 
in CEL for OpenAPI. This example uses "Object" name as the root type and then uses the path in the
OpenAPIv3 schema to identify nested types.

This repo also contains examples that perform version conversion and that use different approaches.

For example, the `mutate-templates` directory shows an approach where templates containing CEL
expressions are embedded in YAML.

TODO
----

- [ ] Support listType=map for apply() function's field removal
- [ ] Experiment with Guided APIs, in particular, using field paths to specify which field to modify.

Mutation cases to test:

- [ ] Sidecar injection
- [ ] Auto-population of fields (AlwaysPullImages)
- [ ] Inject environment variable
- [ ] Modify args
- [ ] Inject readiness/liveness probes
- [ ] Clear a field
- [x] Inject labels/annotations
- [ ] Add if not present

Conversion cases to test:

- [x] Rename a field
- [x] Change the type of a field
- [ ] Move a field to a different location in the object
- [ ] Move field from annotations to object (the annotation should be unset in the version with the field)
- [x] Re-key a listType=map
- [x] Split a field into two fields (e.g. field="a/b" becomes field1="a", field2="b")
- [ ] Convert from scalar field to a list of scalars, (e.g. field="a" becomes field1=["a"])
- [ ] Conditionally convert (if apply the transformation, else do nothing)
- [ ] Complex type instantiation (e.g. spec.x,spec.y,spec.z becomes spec.subobj.x, spec.subobj.y, spec.subobj.z)
- [ ] convert from an annotation to a field?
- [ ] Deletions (unsetting a object property, removing a list element or a map entry)
- [ ] Apply a transform to all the objects in a list (e.g. rename a field in a list)