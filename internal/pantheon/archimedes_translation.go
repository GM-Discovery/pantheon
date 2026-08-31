package pantheon

import (
	"encoding/json"
	"fmt"
	"strings"
)

// archimedesActionIntent is the only frontier response shape that may enter
// Pantheon's action-draft seam. It is still a suggestion: createActionDraft
// keeps it at Future until a human accepts it.
type archimedesActionIntent struct {
	Kind          string         `json:"kind"`
	Summary       string         `json:"summary,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
	Uncertainties []string       `json:"uncertainties,omitempty"`
}

type archimedesFrontierEnvelope struct {
	Answer        string                   `json:"answer"`
	Actions       []archimedesActionIntent `json:"actions,omitempty"`
	Uncertainties []string                 `json:"uncertainties,omitempty"`
}

type archimedesTranslatedResponse struct {
	Answer           string                   `json:"answer"`
	Actions          []archimedesActionIntent `json:"actions,omitempty"`
	Uncertainties    []string                 `json:"uncertainties,omitempty"`
	TranslationState string                   `json:"translation_state"`
}

func archimedesJSONContent(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "```") && strings.HasSuffix(value, "```") {
		lines := strings.Split(value, "\n")
		if len(lines) >= 3 {
			return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}
	return value
}

func translateArchimedesResponse(answer string) (archimedesTranslatedResponse, error) {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return archimedesTranslatedResponse{}, fmt.Errorf("frontier response contained no answer")
	}
	content := archimedesJSONContent(answer)
	if !strings.HasPrefix(content, "{") {
		return archimedesTranslatedResponse{Answer: answer, TranslationState: "plain_text_no_actions"}, nil
	}
	var envelope archimedesFrontierEnvelope
	if err := json.Unmarshal([]byte(content), &envelope); err != nil {
		return archimedesTranslatedResponse{Answer: answer, TranslationState: "malformed_structured_response", Uncertainties: []string{"The frontier response looked structured but could not be translated into typed action proposals."}}, nil
	}
	if strings.TrimSpace(envelope.Answer) == "" {
		return archimedesTranslatedResponse{}, fmt.Errorf("structured frontier response contained no answer")
	}
	for index := range envelope.Actions {
		action := &envelope.Actions[index]
		action.Kind = strings.TrimSpace(strings.ToLower(action.Kind))
		action.Summary = strings.TrimSpace(action.Summary)
		if action.Payload == nil {
			action.Payload = map[string]any{}
		}
		if action.Kind == "" {
			return archimedesTranslatedResponse{}, fmt.Errorf("structured frontier action %d has no kind", index+1)
		}
	}
	return archimedesTranslatedResponse{Answer: strings.TrimSpace(envelope.Answer), Actions: envelope.Actions, Uncertainties: envelope.Uncertainties, TranslationState: "typed"}, nil
}

func rehydrateArchimedesValue(value any, aliases []archimedesPrivateAlias) any {
	switch typed := value.(type) {
	case string:
		for _, alias := range aliases {
			typed = strings.ReplaceAll(typed, alias.Alias, alias.Value)
		}
		return typed
	case []any:
		result := make([]any, len(typed))
		for index := range typed {
			result[index] = rehydrateArchimedesValue(typed[index], aliases)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = rehydrateArchimedesValue(item, aliases)
		}
		return result
	default:
		return value
	}
}

func rehydrateArchimedesActions(actions []archimedesActionIntent, aliases []archimedesPrivateAlias) []archimedesActionIntent {
	result := make([]archimedesActionIntent, len(actions))
	for index, action := range actions {
		result[index] = action
		result[index].Summary, _ = rehydrateArchimedesValue(action.Summary, aliases).(string)
		result[index].Payload, _ = rehydrateArchimedesValue(action.Payload, aliases).(map[string]any)
		if result[index].Payload == nil {
			result[index].Payload = map[string]any{}
		}
		for uncertaintyIndex := range result[index].Uncertainties {
			value, _ := rehydrateArchimedesValue(result[index].Uncertainties[uncertaintyIndex], aliases).(string)
			result[index].Uncertainties[uncertaintyIndex] = value
		}
	}
	return result
}

func (d *daemon) routeArchimedesResponseToApollo(exchange archimedesExchange, resultPath string, response archimedesTranslatedResponse) ([]actionDraft, error) {
	if len(response.Actions) == 0 {
		return nil, nil
	}
	reportID := "archimedes-" + exchange.ExchangeID
	citations := []string{resultPath}
	drafts := make([]actionDraft, 0, len(response.Actions))
	for _, action := range response.Actions {
		kind := action.Kind
		if kind != actionKindCalendar && kind != actionKindEmail {
			kind = actionKindClarification
			action.Payload = map[string]any{"requested_action": action.Summary, "frontier_kind": action.Kind}
		}
		payload, err := json.Marshal(action.Payload)
		if err != nil {
			return drafts, fmt.Errorf("marshal frontier action %q: %w", action.Kind, err)
		}
		state, reason := actionDraftState(kind, payload)
		if len(action.Uncertainties) > 0 {
			reason = strings.TrimSpace(strings.Join(action.Uncertainties, "; "))
			if state == actionStateProposed && reason != "" {
				state = actionStateClarification
			}
		}
		draft, err := d.createActionDraft(actionDraft{
			Kind: kind, State: state, SourceReportID: reportID, SourceReportPath: resultPath,
			Summary: actionDraftSummary(kind, payload, action.Summary), Payload: payload,
			SourceCitations: citations, LastError: reason,
		})
		if err != nil {
			return drafts, fmt.Errorf("persist Apollo action handoff for %q: %w", action.Kind, err)
		}
		drafts = append(drafts, draft)
	}
	return drafts, nil
}
