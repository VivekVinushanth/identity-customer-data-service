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

package model

import "time"

// Event represents an incoming behavioural event that feeds the orchestration
// engine (and, in future, analytics). OrgHandle is always resolved from the
// authenticated request path, never accepted from the client, so a caller
// cannot spoof another tenant's events.
type Event struct {
	EventId        string                 `json:"event_id" bson:"event_id"`
	OrgHandle      string                 `json:"org_handle" bson:"org_handle"`
	ProfileId      string                 `json:"profile_id,omitempty" bson:"profile_id,omitempty"`
	ApplicationId  string                 `json:"application_id,omitempty" bson:"application_id,omitempty"`
	EventType      string                 `json:"event_type" bson:"event_type" binding:"required"`
	EventName      string                 `json:"event_name" bson:"event_name" binding:"required"`
	EventTimestamp int64                  `json:"event_timestamp,omitempty" bson:"event_timestamp,omitempty"`
	Properties     map[string]interface{} `json:"properties,omitempty" bson:"properties,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty" bson:"context,omitempty"`
	CreatedAt      time.Time              `json:"created_at" bson:"created_at"`
}

// EventRequest is the client-facing payload for POST /events. OrgHandle and
// EventId are assigned by the server.
type EventRequest struct {
	ProfileId      string                 `json:"profile_id,omitempty"`
	ApplicationId  string                 `json:"application_id,omitempty"`
	EventType      string                 `json:"event_type"`
	EventName      string                 `json:"event_name"`
	EventTimestamp int64                  `json:"event_timestamp,omitempty"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}

// ToMap returns a flattened view of the event used by the orchestration
// engine's condition evaluator and templating (accessed as "event.<field>").
func (e Event) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"event_id":        e.EventId,
		"profile_id":      e.ProfileId,
		"application_id":  e.ApplicationId,
		"event_type":      e.EventType,
		"event_name":      e.EventName,
		"event_timestamp": e.EventTimestamp,
		"properties":      e.Properties,
		"context":         e.Context,
	}
}
