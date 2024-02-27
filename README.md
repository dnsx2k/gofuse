![gofuse.png](docs%2Fgofuse.png)

### To do:
- Configuration for known hosts (long-pooling option, rps limit etc) + default when no configured
- Respect max rps feature
- Metrics

# [WIP] GoFuse - circuit-breaker sidecar

![circuit-breaker.drawio.png](docs%2Fcircuit-breaker.drawio.png)

GoFuse is an open-source project aimed at providing a versatile circuit breaker proxy written in Go, designed to seamlessly integrate with any microservice architecture. The primary objective of GoFuse is to enhance reliability and resilience by controlling the flow of requests based on the health and availability of upstream services.

Project Description: GoFuse - Circuit Breaker Sidecar Proxy

GoFuse is an open-source project aimed at providing a versatile circuit breaker proxy written in Go, designed to seamlessly integrate with any microservice architecture. The primary objective of GoFuse is to enhance reliability and resilience by controlling the flow of requests based on the health and availability of upstream services.

Key Features:

- Versatile Sidecar Integration: GoFuse functions as a sidecar proxy, capable of integration with any microservice without requiring significant changes to existing codebases.
- HTTP Proxy Configuration: Leveraging Go's built-in proxy capabilities, GoFuse streamlines HTTP client configuration, simplifying the setup process for developers.
- Packet Transport via HTTP: All traffic between the host and the proxy is directed via HTTP, bypassing the need for SSL certificates and minimizing latency. Outbound communications from the proxy service are enforced to use HTTPS for secure transmission.
- Long-Polling Feature: In scenarios where the circuit breaker is open, GoFuse offers a long-polling feature to hold incoming requests for a specified period, allowing the upstream service a chance to recover before responding with a "Service Unavailable" status.
- Custom Round-Tripper: To overcome limitations in reaching services via HTTP due to the absence of TLS support, GoFuse provides a custom round-tripper. This solution enforces HTTP communication from client to proxy, ensuring compatibility and enhancing functionality.