# proto/

Reserved for future gRPC contracts. The current v1 architecture uses REST
(inter-service, synchronous) and RabbitMQ/JSON (events), so this directory
is intentionally empty today — see
[docs/architecture.md](../docs/architecture.md).

Candidates for protobuf definitions later:

- A future high-throughput internal API (e.g. Flight Prediction Service
  batch scoring) where REST/JSON overhead matters.
- Sharing strongly-typed event schemas across languages if a non-Go
  consumer (e.g. a Python ML service) joins the `flight.events` exchange.

Adding gRPC here must not require changes to existing REST contracts in
[docs/api-contracts/](../docs/api-contracts/) — it is additive per the
extensibility goals in architecture.md §5.
