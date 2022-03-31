module github.com/smartwalle/dbc/examples

require (

	github.com/smartwalle/dbc v0.0.0
	github.com/smartwalle/nmap v0.0.0
)

replace (
	github.com/smartwalle/dbc => ../
 	github.com/smartwalle/nmap => /Users/yang/Desktop/smartwalle/nmap
 )

go 1.18
