# YAML Scenario Language Documentation

## Overview

The YAML Scenario Language is used to define event-driven orchestration rules for coordinating multiple simulations. Scenarios describe how events from one simulation should trigger commands to other simulations, enabling complex multi-simulation workflows.

## Table of Contents

- [Basic Structure](#basic-structure)
- [Scenario Definition](#scenario-definition)
- [Rules](#rules)
- [When Conditions](#when-conditions)
- [Actions](#actions)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Basic Structure

A scenario file is a YAML document with the following top-level structure:

```yaml
scenario:
  name: "Scenario Name"
  rules:
    - when: ...
      then: ...
```

## Scenario Definition

### `scenario` (root)

The root element of every scenario file.

**Required**: Yes

**Type**: Object

**Properties**:
- `name` (string, required): A descriptive name for the scenario
- `rules` (array, required): List of event-driven rules

**Example**:
```yaml
scenario:
  name: "Disaster Response Coordination"
  rules: [...]
```

## Rules

Rules define the event-driven behavior of the scenario. Each rule consists of a condition (`when`) and a set of actions (`then`) to execute when the condition is met.

### Rule Structure

```yaml
- when:
    event_type: "event.type"
    from: "simulation_id"  # optional
  then:
    - send_to: "target_sim"
      command: "command_name"
      params:
        key: value
```

**Properties**:
- `when` (object, required): Condition that triggers the rule
- `then` (array, required): List of actions to execute when condition is met

**Behavior**:
- Rules are evaluated in order when an event arrives
- Multiple rules can match the same event
- All matching rules will execute their actions
- Actions within a rule execute sequentially

## When Conditions

The `when` block defines the conditions that must be met for a rule to fire.

### `when` Object

**Required**: Yes

**Type**: Object

**Properties**:
- `event_type` (string, required): The type of event that triggers this rule
- `from` (string, optional): The ID of the simulation that must send the event

### Event Type Matching

Event types are matched exactly. Use dot notation for hierarchical event types (e.g., `attack.detected`, `sensor.triggered`).

**Examples**:
```yaml
# Match any event of this type from any simulation
when:
  event_type: "emergency.activated"

# Match only events from a specific simulation
when:
  event_type: "attack.detected"
  from: "cyber_sim"
```

### Event Type Patterns

- Use descriptive, hierarchical names: `category.action` or `category.subcategory.action`
- Examples:
  - `attack.detected`
  - `sensor.triggered`
  - `disaster.detected`
  - `quality.critical`
  - `fire.alarm`

## Actions

The `then` block contains a list of actions to execute when the rule condition is met.

### Action Structure

```yaml
then:
  - send_to: "target_simulation_id"
    command: "command_name"
    params:
      param1: value1
      param2: value2
    compensate_command: "rollback_command"  # optional
    compensate_params:                       # optional
      param1: value1
```

### Action Properties

#### `send_to` (required)

**Type**: String

The ID of the target simulation that will receive the command. This must match the simulation ID used when the simulation registers with the server.

**Example**:
```yaml
send_to: "vr_sim"
send_to: "cyber_sim"
send_to: "emergency_sim"
```

#### `command` (required)

**Type**: String

The command name to send to the target simulation. This is a string identifier that the target simulation should recognize and handle.

**Example**:
```yaml
command: "show_alert"
command: "activate_protocol"
command: "emergency_stop"
```

#### `params` (required)

**Type**: Object (key-value pairs)

A map of parameters to send with the command. The structure and values depend on what the target simulation expects.

**Supported Value Types**:
- Strings: `"text"`
- Numbers: `42`, `3.14`
- Booleans: `true`, `false`
- Arrays: `["item1", "item2"]`
- Objects: `{key: value}`
- Special values: `"auto"`, `"all"`, `"now"`, etc. (interpreted by target simulation)

**Example**:
```yaml
params:
  message: "Cyber attack detected!"
  severity: 8
  active: true
  channels: ["all"]
  timestamp: "now"
```

#### `compensate_command` (optional)

**Type**: String

A command to execute if this action needs to be rolled back (used in saga patterns for distributed transactions).

**Example**:
```yaml
compensate_command: "cancel_alert"
```

#### `compensate_params` (optional)

**Type**: Object

Parameters for the compensation command.

**Example**:
```yaml
compensate_params:
  alert_id: "auto"
  reason: "rollback"
```

## Examples

### Simple Rule

```yaml
scenario:
  name: "Simple Test Scenario"
  rules:
    - when:
        event_type: "attack.detected"
        from: "cyber_sim"
      then:
        - send_to: "vr_sim"
          command: "show_alert"
          params:
            message: "Cyber attack detected!"
            severity: 8
```

### Multiple Actions

```yaml
scenario:
  name: "Multi-Action Rule"
  rules:
    - when:
        event_type: "sensor.triggered"
        from: "sensor_sim"
      then:
        - send_to: "cyber_sim"
          command: "log_event"
          params:
            event: "sensor_triggered"
        - send_to: "vr_sim"
          command: "update_status"
          params:
            status: "sensor_active"
```

### Complex Scenario

```yaml
scenario:
  name: "Disaster Response Coordination"
  rules:
    # Natural disaster detection
    - when:
        event_type: "disaster.detected"
      then:
        - send_to: "emergency_sim"
          command: "activate_protocol"
          params:
            protocol: "disaster_response"
            severity: "high"
        - send_to: "communication_sim"
          command: "broadcast_alert"
          params:
            channels: ["all"]
            message: "Disaster detected - emergency protocols activated"
        - send_to: "resource_sim"
          command: "allocate_resources"
          params:
            priority: "critical"
    
    # Evacuation order
    - when:
        event_type: "evacuation.ordered"
        from: "emergency_sim"
      then:
        - send_to: "vr_sim"
          command: "display_evacuation_route"
          params:
            routes: "all_available"
            update_interval: 30
        - send_to: "navigation_sim"
          command: "calculate_routes"
          params:
            avoid_areas: "hazard_zones"
```

### Rule Without Source Filter

```yaml
scenario:
  name: "Generic Emergency Rule"
  rules:
    # This rule matches the event type from any simulation
    - when:
        event_type: "emergency.activated"
      then:
        - send_to: "vr_sim"
          command: "emergency_mode"
          params:
            active: true
        - send_to: "cyber_sim"
          command: "emergency_mode"
          params:
            active: true
```

## Best Practices

### 1. Naming Conventions

- **Event Types**: Use hierarchical dot notation (`category.action` or `category.subcategory.action`)
  - Good: `attack.detected`, `sensor.temperature.critical`, `disaster.detected`
  - Avoid: `attackDetected`, `sensor_triggered` (inconsistent)

- **Command Names**: Use snake_case or kebab-case
  - Good: `show_alert`, `activate-protocol`, `emergency_stop`
  - Avoid: `showAlert` (camelCase), `ShowAlert` (PascalCase)

- **Simulation IDs**: Use consistent, descriptive identifiers
  - Good: `cyber_sim`, `vr_sim`, `emergency_sim`
  - Avoid: `sim1`, `sim2`, `cyberSim`

### 2. Rule Organization

- Group related rules together with comments
- Order rules by priority (most specific first, then general)
- Use comments to explain complex rules

```yaml
rules:
  # Critical alerts - highest priority
  - when:
      event_type: "emergency.critical"
    then: [...]
  
  # General alerts
  - when:
      event_type: "alert.general"
    then: [...]
```

### 3. Parameter Design

- Use descriptive parameter names
- Provide default values when possible (handled by target simulation)
- Use arrays for multiple values: `channels: ["all"]`
- Use objects for structured data when needed

### 4. Error Handling

- Design rules to be idempotent when possible
- Consider using compensation commands for critical operations
- Test scenarios with missing or disconnected simulations

### 5. Documentation

- Add comments explaining complex rules
- Document the purpose of each rule
- Note any dependencies between rules

```yaml
rules:
  # When cyber sim detects attack, alert VR sim
  # This rule triggers the primary alert system
  - when:
      event_type: "attack.detected"
      from: "cyber_sim"
    then: [...]
```

## Validation

The server validates scenario files when they are loaded:

- **YAML Syntax**: Must be valid YAML
- **Structure**: Must have `scenario.name` and `scenario.rules`
- **Rules**: Each rule must have `when` and `then`
- **When Conditions**: Must have `event_type`
- **Actions**: Each action must have `send_to`, `command`, and `params`

Invalid scenarios will be rejected with an error message.

## File Format

- **File Extension**: `.yaml` or `.yml`
- **Encoding**: UTF-8
- **Indentation**: Use spaces (2 or 4 spaces per level, be consistent)

## Integration with Server

1. **Upload**: Scenarios can be uploaded via the `/api/scenarios/upload` endpoint
2. **Loading**: The server loads scenarios at startup or when uploaded
3. **Matching**: When an event arrives, all rules are checked in order
4. **Execution**: Matching rules execute their actions sequentially
5. **Delivery**: Commands are sent to target simulations via WebSocket

## See Also

- [README.md](./README.md) - General server documentation
- [TESTING.md](./TESTING.md) - Testing scenarios
- Example scenarios in `../scenarios/` directory
