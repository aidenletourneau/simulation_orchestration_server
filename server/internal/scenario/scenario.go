package scenario

import (
	"fmt"
	"log"
	"os"

	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/models"
	"gopkg.in/yaml.v3"
)

// ScenarioManager handles loading and matching scenario rules
type ScenarioManager struct {
	scenario *models.Scenario
}

// NewScenarioManager creates a new scenario manager
func NewScenarioManager() *ScenarioManager {
	return &ScenarioManager{}
}

// LoadScenario loads a scenario from a YAML file
func (sm *ScenarioManager) LoadScenario(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read scenario file: %w", err)
	}

	return sm.LoadScenarioFromBytes(data)
}

// LoadScenarioFromBytes loads a scenario from YAML bytes
func (sm *ScenarioManager) LoadScenarioFromBytes(data []byte) error {
	var scenarioFile models.ScenarioFile
	if err := yaml.Unmarshal(data, &scenarioFile); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	sm.scenario = &scenarioFile.Scenario
	log.Printf("Loaded scenario: %s with %d rules", scenarioFile.Scenario.Name, len(scenarioFile.Scenario.Rules))
	return nil
}

// GetCurrentScenario returns information about the currently loaded scenario
func (sm *ScenarioManager) GetCurrentScenario() *models.Scenario {
	return sm.scenario
}

// ProcessEvent checks if an event matches any rules and returns actions to execute
func (sm *ScenarioManager) ProcessEvent(event models.Event) []models.Action {
	if sm.scenario == nil {
		return nil
	}

	var actions []models.Action

	for _, rule := range sm.scenario.Rules {
		// Check if event type matches
		if rule.When.EventType != event.EventType {
			continue
		}

		// Check if source matches (if specified in rule)
		if rule.When.From != "" && rule.When.From != event.Source {
			continue
		}

		// Rule matches! Add all actions
		log.Printf("Rule matched! Event: %s from %s", event.EventType, event.Source)
		actions = append(actions, rule.Then...)
	}

	return actions
}
