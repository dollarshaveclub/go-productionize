# Go Productionize

Go Productionize is a set of libraries that can make building a production ready environment easier. It provides a set of libraries that can be used separately, but are more powerful when combined together.

## Service Info (svcinfo)

The service info library stores data that is compiled in at build of the binary.

[Service Info](svcinfo/README.md)

## Reporter

The reporter library reports different runtime metrics to the DataDog service. This library sends metrics constantly, with a delay of the provided period for each report.

[Reporter](reporter/README.md)

## Healthz

Healthz is an HTTP endpoint that provides health information of the application along with some helpful information to the developer such as profiling through pprof. Healthz should be placed on its own HTTP Serve Mux as it contains administrative actions that could be harmful if provided to outside access.

[Healthz](healthz/README.md)
