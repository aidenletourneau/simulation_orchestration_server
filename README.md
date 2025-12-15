# Cloud-Native Multi-Simulation Orchestration Server (Go)

## 1. Overview

Department of Energy (DoE) training exercises frequently rely on multiple independent simulation systems, such as cyber ranges, VR/AR emergency response environments, and physical facility or sensor simulations. While standards like HLA, DIS, and TENA enable data-level interoperability, they do not provide a simple, reusable mechanism for **scenario-level orchestration**—i.e., defining how events in one simulation should trigger coordinated actions in others.

This project proposes a **cloud-hosted orchestration server written in Go** that acts as a central “scenario brain.” The server ingests real-time events from connected simulations, evaluates them against a declarative YAML scenario file, and dispatches commands to other simulations accordingly. The system is designed to be cloud-native, extensible, and security-aware, making it suitable as a prototype for future DoE multi-domain training environments.

---

## 2. Goals and Non-Goals

### Goals
- Enable **scenario-level orchestration** across heterogeneous simulations.
- Provide **real-time event-driven coordination** using a lightweight communication layer.
- Support **pluggable simulation clients** (cyber, VR, physical/sensor).
- Be **cloud deployable** using Docker and AWS.
- Demonstrate awareness of **DoE security constraints** (Zero Trust, least privilege).

### Non-Goals
- Full integration with HLA/DIS/TENA (out of scope for Winterim).
- Hard real-time or frame-level time synchronization across simulations.
- Handling classified or Controlled Unclassified Information (CUI).
- Achieving production-level ATO or full DoE cyber compliance.

---

## 3. High-Level Architecture

### Core Components
1. **Ingress API / Client Gateway**
   - Accepts real-time connections from simulation clients.
   - Supports WebSockets (recommended for demo) or gRPC streaming.

2. **Simulation Registry**
   - Tracks connected simulations, their identities, capabilities, and current status.

3. **Event Bus (Internal)**
   - Normalizes incoming events and routes them to the scenario engine.
   - Initially implemented with in-memory channels; extensible to NATS or Redis Streams.

4. **Scenario Engine**
   - Loads and validates YAML scenario files.
   - Maintains scenario state and phases.
   - Evaluates trigger rules and determines which actions to execute.

5. **Dispatcher**
   - Sends commands to targeted simulations based on rule evaluation.

6. **Audit Log**
   - Records all events, rule firings, and dispatched actions for after-action review.

---

## 4. Technology Choices and Rationale

### Language: Go
- Strong concurrency primitives (goroutines, channels).
- High performance and small memory footprint.
- Widely used for cloud-native and distributed systems.

### HTTP Routing: `chi`
- Lightweight, idiomatic Go router.
- Well-suited for REST APIs and middleware-based services.

### Configuration Management: `spf13/viper`
- Supports YAML, environment variables, and 12-factor app principles.
- Useful for switching between local/dev/cloud configurations.

### YAML Parsing: `gopkg.in/yaml.v3`
- Canonical YAML parsing library for Go.
- Supports strict decoding and schema validation.

### Real-Time Communication
- **WebSockets** (primary choice): easy integration with Unity and Godot clients.
- **gRPC streaming** (future option): strong typing and service-to-service integration.

### Deployment
- Docker for packaging.
- AWS EC2 or ECS for hosting.
- Reverse proxy (e.g., Traefik) for TLS termination and routing.

---

## 5. External Interfaces

### 5.1 Simulation Client Protocol (WebSocket)

**Register**
```json
{
  "type": "register",
  "simulation_id": "cyber-1",
  "tags": ["cyber"],
  "emits": ["cyber_intrusion_detected"],
  "accepts": ["inject_followup"]
}
