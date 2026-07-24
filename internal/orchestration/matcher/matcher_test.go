/*
 * Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package matcher

import (
	"testing"

	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
)

func testContext() map[string]interface{} {
	return map[string]interface{}{
		"event": map[string]interface{}{
			"event_type": "track",
			"event_name": "add_to_cart",
			"properties": map[string]interface{}{
				"value":       49.65,
				"object_name": "Educational #2",
			},
		},
		"profile": map[string]interface{}{
			"traits": map[string]interface{}{
				"plan":  "free",
				"tags":  []interface{}{"vip", "beta"},
				"email": "",
			},
		},
	}
}

func TestEvaluateConditions(t *testing.T) {
	ctx := testContext()

	tests := []struct {
		name string
		cond model.Condition
		want bool
	}{
		{"eq match", model.Condition{Field: "profile.traits.plan", Operator: constants.OperatorEquals, Value: "free"}, true},
		{"eq mismatch", model.Condition{Field: "profile.traits.plan", Operator: constants.OperatorEquals, Value: "pro"}, false},
		{"neq match", model.Condition{Field: "profile.traits.plan", Operator: constants.OperatorNotEquals, Value: "pro"}, true},
		{"neq missing field", model.Condition{Field: "profile.traits.missing", Operator: constants.OperatorNotEquals, Value: "pro"}, true},
		{"gt numeric pass", model.Condition{Field: "event.properties.value", Operator: constants.OperatorGreaterThan, Value: "10"}, true},
		{"gt numeric fail", model.Condition{Field: "event.properties.value", Operator: constants.OperatorGreaterThan, Value: "100"}, false},
		{"gte boundary", model.Condition{Field: "event.properties.value", Operator: constants.OperatorGreaterOrEq, Value: "49.65"}, true},
		{"lt numeric", model.Condition{Field: "event.properties.value", Operator: constants.OperatorLessThan, Value: "50"}, true},
		{"contains string", model.Condition{Field: "event.properties.object_name", Operator: constants.OperatorContains, Value: "Educational"}, true},
		{"contains array member", model.Condition{Field: "profile.traits.tags", Operator: constants.OperatorContains, Value: "vip"}, true},
		{"contains array miss", model.Condition{Field: "profile.traits.tags", Operator: constants.OperatorContains, Value: "gold"}, false},
		{"exists true", model.Condition{Field: "profile.traits.plan", Operator: constants.OperatorExists}, true},
		{"exists empty string false", model.Condition{Field: "profile.traits.email", Operator: constants.OperatorExists}, false},
		{"exists missing false", model.Condition{Field: "profile.traits.missing", Operator: constants.OperatorExists}, false},
		{"unknown operator", model.Condition{Field: "profile.traits.plan", Operator: "unknown", Value: "free"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateConditions([]model.Condition{tt.cond}, ctx)
			if got != tt.want {
				t.Errorf("EvaluateConditions(%+v) = %v, want %v", tt.cond, got, tt.want)
			}
		})
	}
}

func TestEvaluateConditionsANDSemantics(t *testing.T) {
	ctx := testContext()
	conditions := []model.Condition{
		{Field: "profile.traits.plan", Operator: constants.OperatorEquals, Value: "free"},
		{Field: "event.properties.value", Operator: constants.OperatorGreaterThan, Value: "10"},
	}
	if !EvaluateConditions(conditions, ctx) {
		t.Fatal("expected all conditions to pass")
	}

	conditions = append(conditions, model.Condition{
		Field: "profile.traits.plan", Operator: constants.OperatorEquals, Value: "pro",
	})
	if EvaluateConditions(conditions, ctx) {
		t.Fatal("expected condition list to fail once one condition fails")
	}
}

func TestEvaluateConditionsEmptyListPasses(t *testing.T) {
	if !EvaluateConditions(nil, testContext()) {
		t.Fatal("an empty condition list should always pass")
	}
}

func TestResolveTemplate(t *testing.T) {
	ctx := testContext()

	got := ResolveTemplate("plan={{profile.traits.plan}} event={{event.event_name}}", ctx)
	want := "plan=free event=add_to_cart"
	if got != want {
		t.Errorf("ResolveTemplate() = %q, want %q", got, want)
	}
}

func TestResolveTemplateUnresolvedPlaceholderLeftUntouched(t *testing.T) {
	ctx := testContext()
	got := ResolveTemplate("value={{profile.traits.does_not_exist}}", ctx)
	want := "value={{profile.traits.does_not_exist}}"
	if got != want {
		t.Errorf("ResolveTemplate() = %q, want %q", got, want)
	}
}

func TestResolveTemplatesInConfigNested(t *testing.T) {
	ctx := testContext()
	config := map[string]interface{}{
		"subject": "Cart update for {{profile.traits.plan}} plan",
		"nested": map[string]interface{}{
			"value": "{{event.properties.value}}",
		},
		"list": []interface{}{"{{event.event_type}}", "static"},
		"num":  42,
	}

	resolved := ResolveTemplatesInConfig(config, ctx)

	if resolved["subject"] != "Cart update for free plan" {
		t.Errorf("subject = %v", resolved["subject"])
	}
	nested, ok := resolved["nested"].(map[string]interface{})
	if !ok || nested["value"] != "49.65" {
		t.Errorf("nested.value = %v", resolved["nested"])
	}
	list, ok := resolved["list"].([]interface{})
	if !ok || list[0] != "track" || list[1] != "static" {
		t.Errorf("list = %v", resolved["list"])
	}
	if resolved["num"] != 42 {
		t.Errorf("num = %v, want unchanged 42", resolved["num"])
	}
}
