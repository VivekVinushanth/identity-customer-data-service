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

package executor

import (
	"encoding/json"
	"fmt"

	"github.com/wso2/identity-customer-data-service/internal/orchestration/matcher"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/model"
	profileProvider "github.com/wso2/identity-customer-data-service/internal/profile/provider"
	profileStore "github.com/wso2/identity-customer-data-service/internal/profile/store"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	"github.com/wso2/identity-customer-data-service/internal/system/workers"
)

func init() {
	Register(constants.ActionTypeProfileUpdate, &profileUpdateExecutor{})
	Register(constants.ActionTypeProfileSearch, &profileSearchExecutor{})
	Register(constants.ActionTypeProfileMerge, &profileMergeExecutor{})
}

// -----------------------------------------------------------------------
// profile.update — patches the target profile. Config:
//
//	{ "profile_id": "{{event.profile_id}}" (optional, defaults to the
//	  triggering event's profile_id), "patch": { "<dotted.path>": <value> } }
//
// String values inside "patch" may reference "{{event.*}}", "{{profile.*}}"
// or "{{search_result.*}}" placeholders.
// -----------------------------------------------------------------------

type profileUpdateExecutor struct{}

func (e *profileUpdateExecutor) Execute(execCtx *ExecutionContext, action model.Action) error {

	profileId, _ := action.Config["profile_id"].(string)
	if profileId == "" {
		profileId = execCtx.Event.ProfileId
	} else {
		profileId = matcher.ResolveTemplate(profileId, execCtx.Data)
	}
	if profileId == "" {
		return fmt.Errorf("profile.update: no profile_id available (neither configured nor on the triggering event)")
	}

	patch, ok := action.Config["patch"].(map[string]interface{})
	if !ok || len(patch) == 0 {
		return fmt.Errorf("profile.update: config.patch is required and must be a non-empty object")
	}
	resolvedPatch := matcher.ResolveTemplatesInConfig(patch, execCtx.Data)

	profilesService := profileProvider.NewProfilesProvider().GetProfilesService()
	_, err := profilesService.PatchProfile(profileId, execCtx.OrgHandle, resolvedPatch)
	return err
}

// -----------------------------------------------------------------------
// profile.search — looks up a profile and stores it under "search_result" in
// the execution context for later actions in the same rule. Config:
//
//	{ "by": "profile_id" | "user_id" (default "profile_id"),
//	  "value": "{{event.profile_id}}" (default: the triggering event's profile_id) }
// -----------------------------------------------------------------------

type profileSearchExecutor struct{}

func (e *profileSearchExecutor) Execute(execCtx *ExecutionContext, action model.Action) error {

	by, _ := action.Config["by"].(string)
	if by == "" {
		by = "profile_id"
	}
	value, _ := action.Config["value"].(string)
	if value == "" {
		value = execCtx.Event.ProfileId
	} else {
		value = matcher.ResolveTemplate(value, execCtx.Data)
	}
	if value == "" {
		return fmt.Errorf("profile.search: no lookup value available")
	}

	profilesService := profileProvider.NewProfilesProvider().GetProfilesService()

	var result interface{}
	var err error
	switch by {
	case "user_id":
		result, err = profilesService.FindProfileByUserId(value)
	case "profile_id":
		result, err = profilesService.GetProfile(value)
	default:
		return fmt.Errorf("profile.search: unsupported \"by\" value %q (expected \"profile_id\" or \"user_id\")", by)
	}
	if err != nil {
		return err
	}

	asMap, err := toMap(result)
	if err != nil {
		return fmt.Errorf("profile.search: failed to store search result: %w", err)
	}
	execCtx.SetContextValue("search_result", asMap)
	return nil
}

// -----------------------------------------------------------------------
// profile.merge — re-enqueues a profile onto the unification pipeline
// (internal/system/workers.EnqueueProfileForProcessing) instead of
// duplicating merge logic. This runs the same matching/merging path a normal
// profile create/update triggers, evaluated against the org's configured
// unification rules. Config:
//
//	{ "profile_id": "{{event.profile_id}}" (optional, defaults to the
//	  triggering event's profile_id) }
// -----------------------------------------------------------------------

type profileMergeExecutor struct{}

func (e *profileMergeExecutor) Execute(execCtx *ExecutionContext, action model.Action) error {

	profileId, _ := action.Config["profile_id"].(string)
	if profileId == "" {
		profileId = execCtx.Event.ProfileId
	} else {
		profileId = matcher.ResolveTemplate(profileId, execCtx.Data)
	}
	if profileId == "" {
		return fmt.Errorf("profile.merge: no profile_id available (neither configured nor on the triggering event)")
	}

	profile, err := profileStore.GetProfile(profileId)
	if err != nil {
		return err
	}
	if profile == nil {
		return fmt.Errorf("profile.merge: profile %q not found", profileId)
	}

	workers.EnqueueProfileForProcessing(*profile)
	return nil
}

// toMap round-trips v through JSON to obtain a plain map[string]interface{}
// suitable for the matcher's dotted-path resolution, regardless of v's
// concrete struct type.
func toMap(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return map[string]interface{}{}, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
