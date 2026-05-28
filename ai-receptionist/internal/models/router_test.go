package models

import (
	"testing"
)

func TestGetModel_intentClassify_envOverride(t *testing.T) {
	t.Setenv("INTENT_CLASSIFY_MODEL", "custom-intent-model")
	SetConfigModel("cfg-model")
	if got := GetModel("intent_classify"); got != "custom-intent-model" {
		t.Fatalf("got %q want custom-intent-model", got)
	}
}

func TestGetModel_intentClassify_configDefault(t *testing.T) {
	t.Setenv("INTENT_CLASSIFY_MODEL", "")
	SetConfigModel("cfg-model")
	if got := GetModel("intent_classify"); got != "cfg-model" {
		t.Fatalf("got %q want cfg-model", got)
	}
}

func TestGetModel_intentClassify_fallback(t *testing.T) {
	t.Setenv("INTENT_CLASSIFY_MODEL", "")
	SetConfigModel("")
	if got := GetModel("intent_classify"); got != "gemma4:31b-cloud" {
		t.Fatalf("got %q", got)
	}
}

func TestGetModel_otherTask_usesConfig(t *testing.T) {
	SetConfigModel("planner-model")
	if got := GetModel("planner"); got != "planner-model" {
		t.Fatalf("got %q", got)
	}
}
