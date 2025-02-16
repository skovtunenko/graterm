/*
Package graterm provides a structured API for managing graceful application shutdown in response to specific [os.Signal] events.
It ensures a controlled and predictable shutdown process by allowing the registration of ordered termination hooks.

# Purpose

The graterm package simplifies application shutdown handling by introducing a centralized shutdown manager, the [Terminator].
It listens for specified [os.Signal] events and orchestrates the orderly execution of termination hooks.
Hooks allow resources to be properly cleaned up before the application exits.

# Key Concepts

  - [Terminator]: A singleton shutdown manager responsible for capturing OS termination signals and executing registered hooks.
  - [Hook]: A termination callback function that performs cleanup tasks during shutdown.
  - [Order]: A numeric priority assigned to each hook, dictating execution order. Hooks with the same [Order] execute concurrently, while hooks with a lower [Order] value complete before those with a higher [Order] value.

# Features

  - Enforced Hook Registration: Hooks must only be registered via [Terminator] methods, ensuring proper ordering and execution.
  - Ordered Execution: Hooks execute sequentially based on their assigned [Order]; those with the same priority run concurrently.
  - Configurable Timeouts: Each Hook may have an individual timeout, and a global shutdown timeout can be set to enforce an upper limit on the termination process.
  - Optional Logging: the library does not depend on a specific logging framework.
  - Traceability: Hooks can be assigned a name for logging purposes when a [Logger] is attached to the [Terminator].
  - Panic Safety: Panics inside hooks are caught, logged, and do not disrupt the overall shutdown sequence.

# Additional Considerations

  - Concurrent execution of hooks with the same [Order] requires careful management of shared resources to avoid race conditions.
  - Incorrect timeout configurations may cause delays in shutdown; ensure timeouts are set to appropriate values.
*/
package graterm
