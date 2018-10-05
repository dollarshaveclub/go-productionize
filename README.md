# Go Productoinize

Go Productionize is a set of libraries that can make building a production ready environment easier. It provides a set of libraries that can be used separately, but are more powerful when combined together.

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