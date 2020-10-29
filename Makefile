code-gen:
#	@bash ./hack/code-generator/generate-groups.sh "deepcopy,client,lister" \
#      github.com/laik/yce-cloud-extensions/pkg/client github.com/laik/yce-cloud-extensions/pkg/apis \
#      yamecloud:v1
	@bash ./hack/code-generator/generate-groups.sh "deepcopy" \
      github.com/laik/yce-cloud-extensions/pkg/client github.com/laik/yce-cloud-extensions/pkg/apis \
      yamecloud:v1