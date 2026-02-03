This project demonstrates a multi-device, edge-first AI companion system built around a “pick up and orchestrate” model: **whichever device the user is currently using becomes the orchestrator**, and the other devices dynamically switch into helper roles (controller, sensor, or compute worker).

We are prototyping the full workflow on **Windows and macOS first** so it runs end to end on common hardware. During the hackathon, we will **enable Snapdragon-specific acceleration** by swapping the AI execution layer to Qualcomm’s NPU stack, while keeping the same control and media interfaces.

We will check if its qualcom, then use their NPU and their sdks. (For windows we will use execution along with copilot+ pc features)
The system enables:

* **Reliable remote execution and AI task orchestration via gRPC** across devices
* **Instant, low-latency screen, video, and audio streaming via WebRTC**, activated only when needed
* **Seamless switching between command-only mode and live visual interaction**
* **Fully local, edge-first operation with no cloud dependency**

The core idea is a dual-plane architecture:

* **gRPC is the always-on control plane** for discovery, sessions, commands, AI task routing, file transfer, telemetry, and WebRTC signaling
* **WebRTC is the on-demand media plane** for real-time screen and audio when visual context is required

At runtime, devices advertise their capabilities (CPU/GPU/NPU availability, battery, bandwidth). The current orchestrator routes tasks accordingly: light tasks run locally, heavier tasks are delegated to the best available helper device, and WebRTC is only activated when visual context is required. During the hackathon, Snapdragon devices will serve as high-efficiency AI workers through NPU acceleration.
