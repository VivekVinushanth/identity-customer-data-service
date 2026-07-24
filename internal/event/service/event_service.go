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

package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/wso2/identity-customer-data-service/internal/event/model"
	"github.com/wso2/identity-customer-data-service/internal/event/store"
	"github.com/wso2/identity-customer-data-service/internal/orchestration/worker"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
)

type EventServiceInterface interface {
	AddEvent(event model.EventRequest, orgHandle string) (*model.Event, error)
	GetEvents(orgHandle string, limit int) ([]model.Event, error)
	GetEvent(eventId string) (*model.Event, error)
	GetEventsByProfile(orgHandle, profileId string, limit int) ([]model.Event, error)
}

// EventService is the default implementation of the EventServiceInterface.
type EventService struct{}

// GetEventService creates a new instance of EventService.
func GetEventService() EventServiceInterface {

	return &EventService{}
}

// AddEvent validates and persists an incoming event, then hands it off to the
// orchestration worker for asynchronous rule matching. Persistence failures
// are returned to the caller; orchestration dispatch is best-effort and never
// fails the write (the event is already durably stored).
func (es *EventService) AddEvent(req model.EventRequest, orgHandle string) (*model.Event, error) {

	if req.EventType == "" || req.EventName == "" {
		return nil, errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.EVENT_VALIDATION.Code,
			Message:     errors2.EVENT_VALIDATION.Message,
			Description: "event_type and event_name are required",
		}, http.StatusBadRequest)
	}

	now := time.Now().UTC()
	event := model.Event{
		EventId:        uuid.New().String(),
		OrgHandle:      orgHandle,
		ProfileId:      req.ProfileId,
		ApplicationId:  req.ApplicationId,
		EventType:      req.EventType,
		EventName:      req.EventName,
		EventTimestamp: req.EventTimestamp,
		Properties:     req.Properties,
		Context:        req.Context,
		CreatedAt:      now,
	}
	if event.EventTimestamp == 0 {
		event.EventTimestamp = now.Unix()
	}

	if err := store.AddEvent(event); err != nil {
		return nil, err
	}

	worker.EnqueueEventForOrchestration(event)
	return &event, nil
}

// GetEvents fetches the most recent events for the org.
func (es *EventService) GetEvents(orgHandle string, limit int) ([]model.Event, error) {

	if limit <= 0 {
		limit = constants.DefaultLimit
	}
	return store.GetEvents(orgHandle, limit)
}

// GetEventsByProfile fetches the most recent events for a single profile.
func (es *EventService) GetEventsByProfile(orgHandle, profileId string, limit int) ([]model.Event, error) {

	if limit <= 0 {
		limit = constants.DefaultLimit
	}
	return store.GetEventsByProfile(orgHandle, profileId, limit)
}

// GetEvent fetches a specific event by Id.
func (es *EventService) GetEvent(eventId string) (*model.Event, error) {

	event, err := store.GetEvent(eventId)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.EVENT_NOT_FOUND.Code,
			Message:     errors2.EVENT_NOT_FOUND.Message,
			Description: fmt.Sprintf("Event: '%s' not found", eventId),
		}, http.StatusNotFound)
	}
	return event, nil
}
