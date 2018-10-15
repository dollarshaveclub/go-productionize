# Healthz

Healthz is an HTTP endpoint that provides health information of the application along with some helpful information to the developer such as profiling through pprof. Healthz should be placed on its own HTTP Serve Mux as it contains administrative actions that could be harmful if provided to outside access.

## Usage

Below is an example for usage of the healthz library.

```go
func main() {
  // Create a separate mux because the net/pprof package likes to place things
  // which we don't want
  healthMux := &http.ServeMux{}

  // Setup healthz and use the option to set it to be alive right away
  //
  // healthz.SetAlive() is an option that can be passed along with others that allow
  // you to configure different aspects of healthz. Other configuration functions are
  // available for different configurations and can be chained to the end of the
  // healthz.New() function.
  h, err := healthz.New(healthMux, healthz.SetAlive())
  if err != nil {
    log.Println(err)
  }

  // Start healthz on its own port in the background
  go http.ListenAndServe(":8888", healthMux)

  // Create a separate mux for the actual application to use to prevent leaking
  // pprof endpoints to the public
  appMux := &http.ServeMux{}

  // Initialize whatever you need to for your app
  // ...

  // Once initialized, mark the app as "ready" and start the app
  h.Ready()

  http.ListenAndServe(":8080", appMux)
}
```

### Updating Kubernetes Configuration

Your Kubernetes configuration will need to be updated when adding the Healthz endpoints. You will need to update your `liveliness` and `readiness` configuration to use the new `/healthz` endpoints as well as the port that Healthz is running on.

### Acessing Healthz through Kubernetes

Healthz is place on a different port than what you application will use to accept its traffic. The healthz port will should not be exported to the public since it provides methods for terminating or causing a large amount of work for the application to perform.

So, we will want to use the `kubectl port-forward` command for a specific pod to allow us access as it is required. You will need to pick a Kubernetes pod to use with your port forwarding as well as know the port you want to use.

After the `kubectl port-forward` command is running, you will access the endpoint through your browser using the `localhost` address and the port for Healthz.

### Extensions

You can also add a Reporter along with a DataDog client to extend the information provided back by the `/healthz/stats` endpoint. If these are not provided, there will be fewer stats that are provided back to the user.

## Endpoints

Below is a list of endpoints provided by the healthz library for you to utilize:

* `/healthz/`: List of links to the available endpoints.
* `/healthz/ready`: Readiness endpoint for usage with Kubernetes. This endpoint is basic as it will be accessed many times a minute.
* `/healthz/alive`: Liveliness endpoint for usage with Kubernetes. This endpoint is basic as it will be accessed many times a minute.
* `/healthz/stats`: A set of stats that can be looked up by the user provided by the reporter lib of `go-productoinize`.
* `/healthz/abortabortabort`: Set the app to unhealthy on the `/healthz/ready` endpoint so that Kubernetes will restart the pod.
* `/healthz/diediedie`: Will kill the process with an exit code of 255.
* `/healthz/pprof`: Provides the endpoints provided by the `net/pprof` package. Check that package to see what is available.
