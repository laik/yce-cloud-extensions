all: build-ci build-cd

code-gen:
	@bash ./hack/code-generator/generate-groups.sh "deepcopy" \
      github.com/laik/yce-cloud-extensions/pkg/client github.com/laik/yce-cloud-extensions/pkg/apis \
      yamecloud:v1 

build-ci:
	docker build -t yametech/ci:v0.1.0 -f docker/Dockerfile.ci .
	docker push yametech/ci:v0.1.0

build-cd:
	docker build -t yametech/cd:v0.1.0 -f docker/Dockerfile.cd .
	docker push yametech/cd:v0.1.0