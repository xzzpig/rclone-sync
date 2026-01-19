package model

// HookConfigInputToModel converts a GraphQL HookConfigInput to a database HookConfig model.
func HookConfigInputToModel(input *HookConfigInput) *HookConfig {
	if input == nil {
		return nil
	}
	return &HookConfig{
		URL:     input.URL,
		Method:  input.Method,
		Headers: input.Headers,
		Body:    input.Body,
		Command: input.Command,
		WorkDir: input.WorkDir,
		Timeout: input.Timeout,
	}
}

// ToUpdateInput converts a HookInput to an UpdateHookInput.
func (h *HookInput) ToUpdateInput() UpdateHookInput {
	return UpdateHookInput{
		Enabled:  h.Enabled,
		Priority: h.Priority,
		Event:    &h.Event,
		Type:     &h.Type,
		OnError:  h.OnError,
		Config:   h.Config,
	}
}
