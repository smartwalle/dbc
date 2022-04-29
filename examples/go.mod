module github.com/smartwalle/dbc/examples

require (
	github.com/smartwalle/dbc v0.0.0
	github.com/smartwalle/queue v0.0.1
)

replace (
	github.com/smartwalle/dbc => ../
)

go 1.12
