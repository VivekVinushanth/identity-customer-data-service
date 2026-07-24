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

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	adminConfigService "github.com/wso2/identity-customer-data-service/internal/admin_config/service"
	"github.com/wso2/identity-customer-data-service/internal/event/model"
	"github.com/wso2/identity-customer-data-service/internal/event/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
	"github.com/wso2/identity-customer-data-service/internal/system/security"
	"github.com/wso2/identity-customer-data-service/internal/system/utils"
)

type EventHandler struct{}

func NewEventHandler() *EventHandler {

	return &EventHandler{}
}

// AddEvent handles POST /events: records the event and hands it off to the
// orchestration engine for asynchronous rule matching.
func (eh *EventHandler) AddEvent(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "events:create"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	var req model.EventRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		clientError := errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.BAD_REQUEST.Code,
			Message:     errors2.BAD_REQUEST.Message,
			Description: utils.HandleDecodeError(err, "event"),
		}, http.StatusBadRequest)
		utils.WriteErrorResponse(w, clientError)
		return
	}

	eventProvider := provider.NewEventProvider()
	eventService := eventProvider.GetEventService()
	event, err := eventService.AddEvent(req, orgHandle)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusCreated, event, constants.EventResource)
}

// GetEvents handles GET /events.
func (eh *EventHandler) GetEvents(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "events:view"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	limit := constants.DefaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	eventProvider := provider.NewEventProvider()
	eventService := eventProvider.GetEventService()
	events, err := eventService.GetEvents(orgHandle, limit)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, events, constants.EventResource)
}

// GetEventsForProfile handles GET /profiles/{profileId}/events.
func (eh *EventHandler) GetEventsForProfile(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "events:view"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	profileId := r.PathValue("profileId")
	if profileId == "" {
		utils.HandleError(w, errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.EVENT_NOT_FOUND.Code,
			Message:     errors2.EVENT_NOT_FOUND.Message,
			Description: "Invalid path for profile event retrieval",
		}, http.StatusNotFound))
		return
	}

	limit := constants.DefaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	eventProvider := provider.NewEventProvider()
	eventService := eventProvider.GetEventService()
	events, err := eventService.GetEventsByProfile(orgHandle, profileId, limit)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, events, constants.EventResource)
}

// GetEvent handles GET /events/{eventId}.
func (eh *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {

	if err := security.AuthnAndAuthz(r, "events:view"); err != nil {
		utils.HandleError(w, err)
		return
	}

	orgHandle := utils.ExtractOrgHandleFromPath(r)
	if !isCDSEnabled(orgHandle) {
		utils.HandleError(w, cdsNotEnabledError())
		return
	}

	eventId := r.PathValue("eventId")
	if eventId == "" {
		utils.HandleError(w, errors2.NewClientError(errors2.ErrorMessage{
			Code:        errors2.EVENT_NOT_FOUND.Code,
			Message:     errors2.EVENT_NOT_FOUND.Message,
			Description: "Invalid path for event retrieval",
		}, http.StatusNotFound))
		return
	}

	eventProvider := provider.NewEventProvider()
	eventService := eventProvider.GetEventService()
	event, err := eventService.GetEvent(eventId)
	if err != nil {
		utils.HandleError(w, err)
		return
	}
	utils.RespondJSON(w, http.StatusOK, event, constants.EventResource)
}

func isCDSEnabled(orgHandle string) bool {
	return adminConfigService.GetAdminConfigService().IsCDSEnabled(orgHandle)
}

func cdsNotEnabledError() *errors2.ClientError {
	return errors2.NewClientError(errors2.ErrorMessage{
		Code:        errors2.CDS_NOT_ENABLED.Code,
		Message:     errors2.CDS_NOT_ENABLED.Message,
		Description: errors2.CDS_NOT_ENABLED.Description,
	}, http.StatusBadRequest)
}
