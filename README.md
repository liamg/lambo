# Lambo

<p align="center">
	<img height="890px" src="lamb.png">
</p>

Test API Gateway wrapped lambda functions locally.

Lambo can also be used to test API GW lambdas in CI without needing docker-in-docker. It will take all HTTP requests and route them to a local invocation of your lambda function.

It comes after I had great difficulty getting sam-cli working via DinD in CI.

<p align="center">
	<img src="demo.gif">
</p>


## Usage

### Binary

```bash
lambo --listen-addr 127.0.0.1:3000 ./my-lambda
```

### Docker

```bash
docker run -it -p "3000:3000" -v `pwd`:/app ghcr.io/liamg/lambo:latest /app/my-lambda
```

### Further configuration

```
Usage:
  lambo [lambda-path] [flags]

Flags:
  -e, --env-var stringArray   Add environment variable to expose to the lambda
  -h, --help                  help for lambo
  -l, --listen-addr string    The server will listen for requests on this address and route them to your local lambda function. (default "127.0.0.1:3000")
  -t, --timeout duration      Maximum duration to allow a single invocation of the lambda to run for. (default 30s)
```
