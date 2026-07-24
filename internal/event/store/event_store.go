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

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wso2/identity-customer-data-service/internal/event/model"
	"github.com/wso2/identity-customer-data-service/internal/system/database/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/database/scripts"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
	"github.com/wso2/identity-customer-data-service/internal/system/log"
)

// AddEvent persists a new event.
func AddEvent(event model.Event) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get database client for adding event: %s", event.EventId)
		logger.Debug(errorMsg, log.Error(err))
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.ADD_EVENT.Code,
			Message:     errors2.ADD_EVENT.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()

	propertiesJSON, err := json.Marshal(event.Properties)
	if err != nil {
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.ADD_EVENT.Code,
			Message:     errors2.ADD_EVENT.Message,
			Description: "Failed to serialize event properties",
		}, err)
	}
	contextJSON, err := json.Marshal(event.Context)
	if err != nil {
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.ADD_EVENT.Code,
			Message:     errors2.ADD_EVENT.Message,
			Description: "Failed to serialize event context",
		}, err)
	}

	query := scripts.InsertEvent[provider.NewDBProvider().GetDBType()]
	_, err = dbClient.ExecuteQuery(query, event.EventId, event.OrgHandle, nullableString(event.ProfileId),
		nullableString(event.ApplicationId), event.EventType, event.EventName, event.EventTimestamp,
		string(propertiesJSON), string(contextJSON), event.CreatedAt)
	if err != nil {
		errorMsg := fmt.Sprintf("Error occurred while adding event: %s", event.EventId)
		logger.Debug(errorMsg, log.Error(err))
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.ADD_EVENT.Code,
			Message:     errors2.ADD_EVENT.Message,
			Description: errorMsg,
		}, err)
	}

	logger.Info(fmt.Sprintf("Event '%s' recorded successfully", event.EventId))
	return nil
}

// GetEvents fetches the most recent events for an org, newest first.
func GetEvents(orgHandle string, limit int) ([]model.Event, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get database client for fetching events for organization: %s", orgHandle)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.GET_EVENT.Code,
			Message:     errors2.GET_EVENT.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()

	query := scripts.GetEvents[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, orgHandle, limit)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed in fetching events for organization: %s", orgHandle)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.GET_EVENT.Code,
			Message:     errors2.GET_EVENT.Message,
			Description: errorMsg,
		}, err)
	}

	events := make([]model.Event, 0, len(results))
	for _, row := range results {
		event, err := rowToEvent(row)
		if err != nil {
			logger.Debug("Failed to parse event row", log.Error(err))
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

// GetEventsByProfile fetches the most recent events for a single profile
// within an org, newest first.
func GetEventsByProfile(orgHandle, profileId string, limit int) ([]model.Event, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get database client for fetching events for profile: %s", profileId)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.GET_EVENT.Code,
			Message:     errors2.GET_EVENT.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()

	query := scripts.GetEventsByProfile[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, orgHandle, profileId, limit)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed in fetching events for profile: %s", profileId)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.GET_EVENT.Code,
			Message:     errors2.GET_EVENT.Message,
			Description: errorMsg,
		}, err)
	}

	events := make([]model.Event, 0, len(results))
	for _, row := range results {
		event, err := rowToEvent(row)
		if err != nil {
			logger.Debug("Failed to parse event row", log.Error(err))
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

// GetEvent fetches a single event by its Id.
func GetEvent(eventId string) (*model.Event, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get database client for fetching event: %s", eventId)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.GET_EVENT.Code,
			Message:     errors2.GET_EVENT.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()

	query := scripts.GetEvent[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, eventId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		errorMsg := fmt.Sprintf("Failed in fetching event with event_id: %s", eventId)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.GET_EVENT.Code,
			Message:     errors2.GET_EVENT.Message,
			Description: errorMsg,
		}, err)
	}
	if len(results) == 0 {
		return nil, nil
	}

	event, err := rowToEvent(results[0])
	if err != nil {
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.GET_EVENT.Code,
			Message:     errors2.GET_EVENT.Message,
			Description: "Failed to parse stored event",
		}, err)
	}
	return &event, nil
}

func rowToEvent(row map[string]interface{}) (model.Event, error) {

	var event model.Event
	event.EventId, _ = row["event_id"].(string)
	event.OrgHandle, _ = row["org_handle"].(string)
	event.ProfileId, _ = row["profile_id"].(string)
	event.ApplicationId, _ = row["application_id"].(string)
	event.EventType, _ = row["event_type"].(string)
	event.EventName, _ = row["event_name"].(string)
	if ts, ok := row["event_timestamp"].(int64); ok {
		event.EventTimestamp = ts
	}
	if createdAt, ok := row["created_at"].(time.Time); ok {
		event.CreatedAt = createdAt
	}

	if raw, ok := row["properties"].(string); ok && raw != "" {
		if err := json.Unmarshal([]byte(raw), &event.Properties); err != nil {
			return event, fmt.Errorf("failed to unmarshal properties: %w", err)
		}
	}
	if raw, ok := row["context"].(string); ok && raw != "" {
		if err := json.Unmarshal([]byte(raw), &event.Context); err != nil {
			return event, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}
	return event, nil
}

// nullableString converts an empty string to nil so it is stored as SQL NULL
// rather than an empty string.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
