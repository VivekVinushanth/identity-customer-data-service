/*
 * Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package services

import (
	"net/http"
	"strings"

	"github.com/wso2/identity-customer-data-service/internal/event/handler"
	"github.com/wso2/identity-customer-data-service/internal/system/constants"
)

type EventService struct {
	eventHandler *handler.EventHandler
	mux          *http.ServeMux
}

func NewEventService(mux *http.ServeMux) *EventService {
	s := &EventService{
		eventHandler: handler.NewEventHandler(),
		mux:          mux,
	}

	const base = constants.ApiBasePath + "/v1"
	s.mux.HandleFunc("POST "+base+"/events", s.eventHandler.AddEvent)
	s.mux.HandleFunc("GET "+base+"/events", s.eventHandler.GetEvents)
	s.mux.HandleFunc("GET "+base+"/events/{eventId}", s.eventHandler.GetEvent)
	s.mux.HandleFunc("GET "+base+"/profiles/{profileId}/events", s.eventHandler.GetEventsForProfile)

	return s
}

// Route handles all tenant-aware events endpoints
func (s *EventService) Route(w http.ResponseWriter, r *http.Request) {
	if trimmed := strings.TrimSuffix(r.URL.Path, "/"); trimmed != "" {
		r.URL.Path = trimmed
	}
	s.mux.ServeHTTP(w, r)
}
