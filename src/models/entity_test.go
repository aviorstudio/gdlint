package models

import "testing"

func TestGodotVirtualCallbacksAreProtectedWithoutCalls(t *testing.T) {
	for _, name := range []string{"_get_configuration_warnings", "_get_property_list", "_property_can_revert", "_property_get_revert", "_get", "_set", "_to_string", "_ready", "_on_pressed"} {
		if !(&Entity{Type: EntityFunction, Name: name}).IsProtected() {
			t.Errorf("engine callback %s must survive unused-code cleanup", name)
		}
	}
	for _, entity := range []*Entity{{Type: EntityFunction, Name: "_get_configuration_warnings_extra"}, {Type: EntityFunction, Name: "_helper"}, {Type: EntitySignal, Name: "_get_configuration_warnings"}} {
		if entity.IsProtected() {
			t.Errorf("ordinary entity must still require a use: %+v", entity)
		}
	}
}
