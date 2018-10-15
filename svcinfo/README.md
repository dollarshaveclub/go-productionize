# Service Info (svcinfo)

The service info library stores data that is compiled in at build of the binary.

We currently store the following information about the service:

* Commit SHA
* Version
* Build Date

There are currently two ways to add the information:
* compile time
* environment variables

## Compile Time

We need to update how we build our binaries to give us the information needed:

```bash
go build -ldflags '-X github.com/dollarshaveclub/go-productionize/svcinfo.CommitSHA=${SHORT_SHA} -X github.com/dollarshaveclub/go-productionize/svcinfo.Version=${BUILD_VERSION} -X github.com/dollarshaveclub/go-productionize/svcinfo.BuildDate=${BUILD_DATE}'
```

If dependencies are vendored, you will need to append the import path of the service. For example, if we wanted to set the values for the package `github.com/dollarshaveclub/example` we'd change the above to:

```bash
go build -ldflags '-X github.com/dollarshaveclub/example/vendor/github.com/dollarshaveclub/go-productionize/svcinfo.CommitSHA=${SHORT_SHA} -X github.com/dollarshaveclub/example/vendor/github.com/dollarshaveclub/go-productionize/svcinfo.Version=${BUILD_VERSION} -X github.com/dollarshaveclub/example/vendor/github.com/dollarshaveclub/go-productionize/svcinfo.BuildDate=${BUILD_DATE}'
```

## Environment Variables

Add the following environment variables to your Dockerfile with the information you want to used:

* COMMIT_SHA
* BUILD_DATE
* VERSION

These values will be automatically pulled at service startup if no values were added at compile time.

## Usage by Services

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