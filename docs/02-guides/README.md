# Guides

Practical tutorials and step-by-step guides for using Zenith Operator.

## Contents

### [HTTP Functions](http-functions.md)
Learn how to create synchronous HTTP functions that respond to REST requests. Ideal for APIs, webhooks, and microservices.

**Topics covered:**
- Function code structure
- Function CR configuration
- Build monitoring
- Access via public URL
- Environment variables and advanced settings

### [Event Functions](event-functions.md)
Create asynchronous event-driven functions using Knative Eventing. Perfect for asynchronous processing and event-driven workflows.

**Topics covered:**
- Event-driven architecture
- Broker and Trigger configuration
- CloudEvents event filters
- Event sending and processing
- Advanced patterns (DLQ, fan-out)

### [Function Communication](function-communication.md)
Implement HTTP communication between multiple functions to create complex microservices architectures.

**Topics covered:**
- Service URL patterns
- Synchronous Request-response
- Asynchronous Fire-and-forget
- Service discovery
- Timeout, retry, and circuit breaker
- Integration with Dapr

### [Git Authentication](git-authentication.md)
Configure authentication for private Git repositories using HTTPS or SSH.

**Topics covered:**
- HTTPS with GitHub Token
- SSH with Deploy Keys
- How authentication works
- Common issues troubleshooting
- Support for GitLab, Bitbucket, and private servers

### [Observability](observability.md)
Configure distributed tracing and observability for your functions using OpenTelemetry.

**Topics covered:**
- Tracing configuration
- OpenTelemetry Collector integration
- Trace visualization
- Sampling rates
- Integration with Dapr

## Next Steps

After mastering the practical guides:

- **[Concepts](../03-concepts/)** - Dive deeper into operator architecture
- **[Reference](../04-reference/)** - Consult the complete API specification
- **[Operations](../05-operations/)** - Configure production environment
