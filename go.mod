module github.com/boynton/api

go 1.18

require (
	github.com/boynton/data v0.0.4
	github.com/ghodss/yaml v1.0.0
)

replace github.com/boynton/data => ../data

require (
	github.com/shopspring/decimal v1.3.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
