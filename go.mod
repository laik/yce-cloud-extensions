module github.com/laik/yce-cloud-extensions
go 1.15

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
	golang.org/x/net v0.0.0-20201027133719-8eef5233e2a1 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	k8s.io/api v0.19.3 // indirect
	k8s.io/client-go v0.0.0
	k8s.io/utils v0.0.0-20201027101359-01387209bb0d // indirect
)

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.1
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
	k8s.io/api => k8s.io/api v0.18.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.6
	k8s.io/client-go => k8s.io/client-go v0.18.0
)
