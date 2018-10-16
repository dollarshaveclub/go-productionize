# Go Productoinize

Go Productionize is a set of libraries that can make building a production ready environment easier. It provides a set of libraries that can be used separately, but are more powerful when combined together.

## Service Info (svcinfo)

The service info library stores data that is compiled in at build of the binary.

We currently store the following information about the service:

* Commit SHA
* Version
* Build Date

There are currently two ways to add the information:
* compile time
* environment variables

### Compile Time

We need to update how we build our binaries to give us the information needed:

```bash
go build -ldflags '-X github.com/dollarshaveclub/go-productionize/svcinfo.CommitSHA=${SHORT_SHA} -X github.com/dollarshaveclub/go-productionize/svcinfo.Version=${BUILD_VERSION} -X github.com/dollarshaveclub/go-productionize/svcinfo.BuildDate=${BUILD_DATE}'
```

If dependencies are vendored, you will need to append the import path of the service. For example, if we wanted to set the values for the package `github.com/dollarshaveclub/example` we'd change the above to:

```bash
go build -ldflags '-X github.com/dollarshaveclub/example/vendor/github.com/dollarshaveclub/go-productionize/svcinfo.CommitSHA=${SHORT_SHA} -X github.com/dollarshaveclub/example/vendor/github.com/dollarshaveclub/go-productionize/svcinfo.Version=${BUILD_VERSION} -X github.com/dollarshaveclub/example/vendor/github.com/dollarshaveclub/go-productionize/svcinfo.BuildDate=${BUILD_DATE}'
```

### Environment Variables

Add the following environment variables to your Dockerfile with the information you want to used:

* COMMIT_SHA
* BUILD_DATE
* VERSION

These values will be automatically pulled at service startup if no values were added at compile time.

### Usage by Services

One helpful way to utilize this information is by adding the information as a set of DataDog tags to every metric. We can do this easily by setting up our DataDog client like the following example:

```go
func main() {
    ...

    dd, err := statsd.Client("datadog.addr:8125")
    if err != nil {
        // do something with the error
    }
    dd.Namespace = "service_name."

    infoTags := svcinfo.GetDDTags()
    if len(infoTags) > 0 {
        dd.Tags = append(dd.Tags, infoTags...)
    }

    ...
}
```

We now have the ability to track how stats will change and be able to better understand which version of our service is showing a given issue. It is suggested that you always add the service info DD tags as a default set so that they are added to every metric.

## Reporter

The reporter library reports different runtime metrics to the DataDog service. This library sends metrics constantly, with a delay of the provided period for each report.

Metrics include:

* Memory (runtime.MemStats)
* Runtime
* Go Information (version, arch, and OS as tags)
* Service Information (commit, version, and build date as tags)

### Usage

To use the reporter library, you will need to import the library and the run the "New" function inside of the main of your service:

```go
func main() {
    ...

    dd, err := statsd.Client("datadog.addr:8125")
    if err != nil {
        // do something with the error
    }
    dd.Namespace = "service_name."

    r := reporter.New(dd)
    ...
}
```

A reporter is returned and is helpful for usage with other services in this library.

#### Modifying Properties

There are two properties that can be modified:

* Period (how often metrics are sent)
* Default tags (tags added to every metric reported by reporter)

There are two helper functions that can be used with the "New" method as follows:

```go
func main() {
    ...

    r := reporter.New(dd,
            reporter.Period(10*time.second),
            reporter.DefaultTags([]string{"tag1:blah", "tag2:blahblah"})
        )

    ...
```