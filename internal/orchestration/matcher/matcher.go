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

// Package matcher evaluates orchestration rule conditions and resolves
// "{{event.*}}" / "{{profile.*}}" / "{{search_result.*}}" template
// placeholders against the execution context. It is deliberately dependency
// free (only orchestration/model and system/constants) so both the
// orchestration/executor package and the orchestration worker can depend on
// it without creating an import cycle.
package matcher

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
)

// EvaluateConditions returns true only if every condition passes (AND
// semantics) against ctx, a map with top-level keys such as "event",
// "profile" and (once a profile.search action has run earlier in the same
// rule) "search_result". An empty condition list always passes.
func EvaluateConditions(conditions []model.Condition, ctx map[string]interface{}) bool {

	for _, cond := range conditions {
		if !evaluateCondition(cond, ctx) {
			return false
		}
	}
	return true
}

func evaluateCondition(cond model.Condition, ctx map[string]interface{}) bool {

	actual, found := resolveField(cond.Field, ctx)

	switch cond.Operator {
	case constants.OperatorExists:
		return found && !isEmptyValue(actual)
	case constants.OperatorEquals:
		return found && toComparableString(actual) == cond.Value
	case constants.OperatorNotEquals:
		return !found || toComparableString(actual) != cond.Value
	case constants.OperatorContains:
		return found && containsValue(actual, cond.Value)
	case constants.OperatorGreaterThan, constants.OperatorGreaterOrEq,
		constants.OperatorLessThan, constants.OperatorLessOrEq:
		return found && compareNumeric(actual, cond.Value, cond.Operator)
	default:
		return false
	}
}

// resolveField reads a dotted path (e.g. "event.properties.value" or
// "profile.traits.plan") out of ctx. The first segment selects the top-level
// document ("event", "profile", "search_result"); remaining segments walk
// nested maps.
func resolveField(field string, ctx map[string]interface{}) (interface{}, bool) {

	segments := strings.Split(field, ".")
	if len(segments) == 0 {
		return nil, false
	}

	var current interface{} = ctx[segments[0]]
	if current == nil {
		return nil, false
	}
	for _, segment := range segments[1:] {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[segment]
		if !ok {
			return nil, false
		}
	}
	return current, current != nil
}

func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return s == ""
	}
	return false
}

func toComparableString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}

func containsValue(actual interface{}, expected string) bool {
	switch val := actual.(type) {
	case string:
		return strings.Contains(val, expected)
	case []interface{}:
		for _, item := range val {
			if toComparableString(item) == expected {
				return true
			}
		}
		return false
	default:
		return strings.Contains(toComparableString(actual), expected)
	}
}

func compareNumeric(actual interface{}, expected string, operator string) bool {

	actualFloat, ok := toFloat(actual)
	if !ok {
		return false
	}
	expectedFloat, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return false
	}

	switch operator {
	case constants.OperatorGreaterThan:
		return actualFloat > expectedFloat
	case constants.OperatorGreaterOrEq:
		return actualFloat >= expectedFloat
	case constants.OperatorLessThan:
		return actualFloat < expectedFloat
	case constants.OperatorLessOrEq:
		return actualFloat <= expectedFloat
	default:
		return false
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

var templatePlaceholder = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)

// ResolveTemplate replaces every "{{path}}" placeholder in value with the
// stringified result of resolving path against ctx. A placeholder that
// doesn't resolve to anything is left untouched so misconfiguration is
// visible in the delivered payload rather than silently blanked out.
func ResolveTemplate(value string, ctx map[string]interface{}) string {

	return templatePlaceholder.ReplaceAllStringFunc(value, func(match string) string {
		path := strings.TrimSpace(match[2 : len(match)-2])
		resolved, found := resolveField(path, ctx)
		if !found {
			return match
		}
		return toComparableString(resolved)
	})
}

// ResolveTemplatesInConfig returns a copy of config with every string value
// (recursively, including inside nested maps) template-resolved against ctx.
func ResolveTemplatesInConfig(config map[string]interface{}, ctx map[string]interface{}) map[string]interface{} {

	resolved := make(map[string]interface{}, len(config))
	for key, value := range config {
		resolved[key] = resolveTemplateValue(value, ctx)
	}
	return resolved
}

func resolveTemplateValue(value interface{}, ctx map[string]interface{}) interface{} {
	switch val := value.(type) {
	case string:
		return ResolveTemplate(val, ctx)
	case map[string]interface{}:
		return ResolveTemplatesInConfig(val, ctx)
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, item := range val {
			out[i] = resolveTemplateValue(item, ctx)
		}
		return out
	default:
		return value
	}
}
