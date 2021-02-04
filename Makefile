all: build-ci build-cd build-unit

code-gen:
	@bash ./hack/code-generator/generate-groups.sh "deepcopy" \
      github.com/laik/yce-cloud-extensions/pkg/client github.com/laik/yce-cloud-extensions/pkg/apis \
      yamecloud:v1 

build-ci:
	docker build -t harbor.ym/devops/ci:v0.1.6.2 -f docker/Dockerfile.ci .
	docker push harbor.ym/devops/ci:v0.1.6.2

build-cd:
	docker build -t harbor.ym/devops/cd:v0.1.2 -f docker/Dockerfile.cd .
	docker push harbor.ym/devops/cd:v0.1.0

build-unit:
	docker build -t harbor.ym/devops/unit:v0.1.2 -f docker/Dockerfile.unit .
	docker push harbor.ym/devops/unit:v0.1.2