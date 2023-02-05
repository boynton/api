module github.com/boynton/api

go 1.17

require (
	github.com/boynton/data v0.0.1
	github.com/boynton/sadl v1.8.4
	github.com/boynton/smithy v0.5.0
	github.com/ghodss/yaml v1.0.0
)

require gopkg.in/yaml.v2 v2.4.0 // indirect

//replace github.com/boynton/sadl => ../sadl
