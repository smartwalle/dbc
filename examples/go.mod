module github.com/smartwalle/dbc/examples

require (
	github.com/smartwalle/dbc v0.0.0
	github.com/smartwalle/queue v0.0.0
)

replace (
	github.com/smartwalle/dbc => ../
	github.com/smartwalle/queue => /Users/yang/Desktop/smartwalle/queue
)

go 1.12
