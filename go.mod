module jpbetz.github.com/celpatch

go 1.20

require (
	github.com/golang/protobuf v1.5.3
	github.com/google/cel-go v0.13.0
	k8s.io/apiextensions-apiserver v0.0.0-00010101000000-000000000000
	k8s.io/apimachinery v0.0.0
	k8s.io/apiserver v0.0.0
	k8s.io/kube-openapi v0.0.0-20230308215209-15aac26d736a
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.1 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/term v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20221027153422-115e99e71e1c // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.0.0 // indirect
	k8s.io/client-go v0.0.0 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
	k8s.io/utils v0.0.0-20230209194617-a36077c30491 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
)

replace (
	k8s.io/api => github.com/jpbetz/kubernetes/staging/src/k8s.io/api v0.0.0-20230412153445-b37cca8618db
	k8s.io/apiextensions-apiserver => github.com/jpbetz/kubernetes/staging/src/k8s.io/apiextensions-apiserver v0.0.0-20230412153445-b37cca8618db
	k8s.io/apimachinery => github.com/jpbetz/kubernetes/staging/src/k8s.io/apimachinery v0.0.0-20230412153445-b37cca8618db
	k8s.io/apiserver => github.com/jpbetz/kubernetes/staging/src/k8s.io/apiserver v0.0.0-20230412153445-b37cca8618db
	k8s.io/client-go => github.com/jpbetz/kubernetes/staging/src/k8s.io/client-go v0.0.0-20230412153445-b37cca8618db
	k8s.io/code-generator => github.com/jpbetz/kubernetes/staging/src/k8s.io/code-generator v0.0.0-20230412153445-b37cca8618db
	k8s.io/component-base => github.com/jpbetz/kubernetes/staging/src/k8s.io/component-base v0.0.0-20230412153445-b37cca8618db
	k8s.io/kms => github.com/jpbetz/kubernetes/staging/src/k8s.io/kms v0.0.0-20230412153445-b37cca8618db
)
