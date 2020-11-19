

docker:
    docker build -t yametech/ci:latest -f docker/Dockerfile.ci .
    docker build -t yametech/cd:latest -f docker/Dockerfile.ci .

code-gen:
#	@bash ./hack/code-generator/generate-groups.sh "deepcopy,client,lister" \
#      github.com/laik/yce-cloud-extensions/pkg/client github.com/laik/yce-cloud-extensions/pkg/apis \
#      yamecloud:v1
	@bash ./hack/code-generator/generate-groups.sh "deepcopy" \
      github.com/laik/yce-cloud-extensions/pkg/client github.com/laik/yce-cloud-extensions/pkg/apis \
      yamecloud:v1

