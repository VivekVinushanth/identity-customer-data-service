/*
 * Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package utils

import (
	"github.com/wso2/identity-customer-data-service/internal/system/authn"
	"github.com/wso2/identity-customer-data-service/internal/system/authz"
	"github.com/wso2/identity-customer-data-service/internal/system/errors"
	"net/http"
	"strings"
)

// AuthnAndAuthz performs authentication and authorization for the given HTTP request and operation.
func AuthnAndAuthz(r *http.Request, operation string) error {

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || authHeader == "" {
		clientError := errors.NewClientError(errors.ErrorMessage{
			Code:        errors.UN_AUTHORIZED.Code,
			Message:     errors.UN_AUTHORIZED.Message,
			Description: "Missing or invalid Authorization header",
		}, http.StatusUnauthorized)
		return clientError
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	//  Validate token
	claims, err := authn.ValidateAuthenticationAndReturnClaims(token)
	if err != nil {
		clientError := errors.NewClientError(errors.ErrorMessage{
			Code:        errors.UN_AUTHORIZED.Code,
			Message:     errors.UN_AUTHORIZED.Message,
			Description: "Missing or invalid Authorization header",
		}, http.StatusUnauthorized)
		return clientError
	}

	if !authz.ValidatePermission(claims["scope"].(string), operation) {
		clientError := errors.NewClientError(errors.ErrorMessage{
			Code:        errors.FORBIDDEN.Code,
			Message:     errors.FORBIDDEN.Message,
			Description: "Do not have permission to perform this operation",
		}, http.StatusForbidden)
		return clientError
	}
	return nil
}
