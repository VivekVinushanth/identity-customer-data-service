/*
 * Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package store

import (
	"fmt"
	"github.com/lib/pq"
	model "github.com/wso2/identity-customer-data-service/internal/consent/model"
	"github.com/wso2/identity-customer-data-service/internal/system/database/provider"
	"github.com/wso2/identity-customer-data-service/internal/system/database/scripts"
	errors2 "github.com/wso2/identity-customer-data-service/internal/system/errors"
	"github.com/wso2/identity-customer-data-service/internal/system/log"
	"strings"
)

// AddConsentCategory inserts a new consent category into the database.
func AddConsentCategory(category model.ConsentCategory) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get db client for inserting consent category: %s", category.CategoryIdentifier)
		logger.Debug(errorMsg, log.Error(err))
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.ADD_CONSENT_CATEGORY.Code,
			Message:     errors2.ADD_CONSENT_CATEGORY.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()
	query := scripts.InsertConsentCategory[provider.NewDBProvider().GetDBType()]
	tx, err := dbClient.BeginTx()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to begin transaction for inserting consent category: %s", category.CategoryIdentifier)
		logger.Debug(errorMsg, log.Error(err))
		serverError := errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.ADD_CONSENT_CATEGORY.Code,
			Message:     errors2.ADD_CONSENT_CATEGORY.Message,
			Description: errorMsg,
		}, err)
		return serverError
	}
	_, err = tx.Exec(query, category.CategoryName, category.CategoryIdentifier, category.OrgHandle, category.Purpose, pq.Array(category.Destinations))
	if err != nil {
		errRollback := tx.Rollback()
		if errRollback != nil {
			errorMsg := fmt.Sprintf("Failed to rollback inserting consent category: %s", category.CategoryIdentifier)
			logger.Debug(errorMsg, log.Error(errRollback))
			return errors2.NewServerError(errors2.ErrorMessage{
				Code:        errors2.ADD_CONSENT_CATEGORY.Code,
				Message:     errors2.ADD_CONSENT_CATEGORY.Message,
				Description: errorMsg,
			}, errRollback)
		}
		errorMsg := fmt.Sprintf("Failed to execute query for inserting consent category: %s", category.CategoryIdentifier)
		logger.Debug(errorMsg, log.Error(err))
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.ADD_CONSENT_CATEGORY.Code,
			Message:     errors2.ADD_CONSENT_CATEGORY.Message,
			Description: errorMsg,
		}, err)
	}

	attrQuery := scripts.InsertConsentCategoryAttribute[provider.NewDBProvider().GetDBType()]
	for _, attr := range category.Attributes {
		appId := attr.AppId
		_, err = tx.Exec(attrQuery, category.CategoryIdentifier, attr.Scope, attr.AttributeId, appId)
		if err != nil {
			_ = tx.Rollback()
			errorMsg := fmt.Sprintf("Failed to insert attribute %s for consent category: %s", attr.AttributeId, category.CategoryIdentifier)
			logger.Debug(errorMsg, log.Error(err))
			return errors2.NewServerError(errors2.ErrorMessage{
				Code:        errors2.ADD_CONSENT_CATEGORY.Code,
				Message:     errors2.ADD_CONSENT_CATEGORY.Message,
				Description: errorMsg,
			}, err)
		}
	}

	logger.Info(fmt.Sprintf("Successfully inserted consent category: %s", category.CategoryIdentifier))
	return tx.Commit()
}

// GetAllConsentCategories retrieves all consent categories from the database.
func GetAllConsentCategories() ([]model.ConsentCategory, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := "Failed to get db client for fetching consent categories."
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()

	query := scripts.GetAllConsentCategories[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query)
	if err != nil {
		errorMsg := "Failed to execute query for fetching consent categories."
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: "Failed to fetch consent categories.",
		}, err)
	}

	categories := make([]model.ConsentCategory, 0, len(results))
	categoryIds := make([]string, 0, len(results))
	for _, row := range results {
		id := row["category_identifier"].(string)
		categories = append(categories, model.ConsentCategory{
			CategoryName:       row["category_name"].(string),
			CategoryIdentifier: id,
			OrgHandle:          row["org_handle"].(string),
			Purpose:            row["purpose"].(string),
			Destinations:       parseStringArray(row["destinations"]),
		})
		categoryIds = append(categoryIds, id)
	}
	if len(categories) == 0 {
		logger.Debug("No consent categories found")
		return nil, nil
	}

	attrsByCategory, err := getAttributesByCategoryIds(dbClient, categoryIds)
	if err != nil {
		return nil, err
	}
	for i := range categories {
		categories[i].Attributes = attrsByCategory[categories[i].CategoryIdentifier]
	}

	logger.Info(fmt.Sprintf("Successfully fetched %d consent categories", len(categories)))
	return categories, nil
}

// GetConsentCategoryByID retrieves a consent category by its ID.
func GetConsentCategoryByID(id string) (*model.ConsentCategory, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get db client for fetching consent category: %s", id)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()

	query := scripts.GetConsentCategoryById[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, id)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute query for fetching consent category: %s", id)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: errorMsg,
		}, err)
	}

	if len(results) == 0 {
		logger.Debug(fmt.Sprintf("Consent category not found for id: %s", id))
		return nil, nil
	}
	row := results[0]
	category := model.ConsentCategory{
		CategoryName:       row["category_name"].(string),
		CategoryIdentifier: row["category_identifier"].(string),
		OrgHandle:          row["org_handle"].(string),
		Purpose:            row["purpose"].(string),
		Destinations:       parseStringArray(row["destinations"]),
	}

	attrsByCategory, err := getAttributesByCategoryIds(dbClient, []string{id})
	if err != nil {
		return nil, err
	}
	category.Attributes = attrsByCategory[id]

	return &category, nil
}

// GetConsentCategoryByName retrieves a consent category by its ID.
func GetConsentCategoryByName(name string) (*model.ConsentCategory, error) {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get db client for fetching consent category: %s", name)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()

	query := scripts.GetConsentCategoryByName[provider.NewDBProvider().GetDBType()]
	results, err := dbClient.ExecuteQuery(query, name)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute query for fetching consent category: %s", name)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: errorMsg,
		}, err)
	}

	if len(results) == 0 {
		logger.Debug(fmt.Sprintf("Consent category not found for name: %s", name))
		return nil, nil
	}
	row := results[0]
	category := model.ConsentCategory{
		CategoryName:       row["category_name"].(string),
		CategoryIdentifier: row["category_identifier"].(string),
		OrgHandle:          row["org_handle"].(string),
		Purpose:            row["purpose"].(string),
		Destinations:       parseStringArray(row["destinations"]),
	}
	return &category, nil
}

// UpdateConsentCategory updates an existing consent category in the database.
func UpdateConsentCategory(category model.ConsentCategory) error {

	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get db client for updating consent category: %s", category.CategoryIdentifier)
		logger.Debug(errorMsg, log.Error(err))
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.UPDATE_CONSENT_CATEGORY.Code,
			Message:     errors2.UPDATE_CONSENT_CATEGORY.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()
	tx, err := dbClient.BeginTx()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to begin transaction for updating consent category: %s",
			category.CategoryIdentifier)
		logger.Debug(errorMsg, log.Error(err))
		serverError := errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.UPDATE_CONSENT_CATEGORY.Code,
			Message:     errors2.UPDATE_CONSENT_CATEGORY.Message,
			Description: errorMsg,
		}, err)
		return serverError
	}

	query := scripts.UpdateConsentCategory[provider.NewDBProvider().GetDBType()]
	_, err = tx.Exec(query, category.CategoryName, category.Purpose, pq.Array(category.Destinations), category.CategoryIdentifier)
	if err != nil {
		_ = tx.Rollback()
		logger.Debug("Failed to update consent category", log.Error(err))
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.UPDATE_CONSENT_CATEGORY.Code,
			Message:     errors2.UPDATE_CONSENT_CATEGORY.Message,
			Description: "Failed to update consent category.",
		}, err)
	}

	deleteAttrQuery := scripts.DeleteConsentCategoryAttributesByCategoryId[provider.NewDBProvider().GetDBType()]
	_, err = tx.Exec(deleteAttrQuery, category.CategoryIdentifier)
	if err != nil {
		_ = tx.Rollback()
		errorMsg := fmt.Sprintf("Failed to delete attributes for consent category: %s", category.CategoryIdentifier)
		logger.Debug(errorMsg, log.Error(err))
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.UPDATE_CONSENT_CATEGORY.Code,
			Message:     errors2.UPDATE_CONSENT_CATEGORY.Message,
			Description: errorMsg,
		}, err)
	}

	insertAttrQuery := scripts.InsertConsentCategoryAttribute[provider.NewDBProvider().GetDBType()]
	for _, attr := range category.Attributes {
		_, err = tx.Exec(insertAttrQuery, category.CategoryIdentifier, attr.Scope, attr.AttributeId, attr.AppId)
		if err != nil {
			_ = tx.Rollback()
			errorMsg := fmt.Sprintf("Failed to insert attribute %s for consent category: %s", attr.AttributeId, category.CategoryIdentifier)
			logger.Debug(errorMsg, log.Error(err))
			return errors2.NewServerError(errors2.ErrorMessage{
				Code:        errors2.UPDATE_CONSENT_CATEGORY.Code,
				Message:     errors2.UPDATE_CONSENT_CATEGORY.Message,
				Description: errorMsg,
			}, err)
		}
	}

	return tx.Commit()
}

func DeleteConsentCategory(categoryId string) error {
	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get db client for deleting consent category: %s", categoryId)
		logger.Debug(errorMsg, log.Error(err))
		serverError := errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.UPDATE_CONSENT_CATEGORY.Code,
			Message:     errors2.UPDATE_CONSENT_CATEGORY.Message,
			Description: errorMsg,
		}, err)
		return serverError
	}
	defer dbClient.Close()

	tx, err := dbClient.BeginTx()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to begin transaction for deleting consent category: %s", categoryId)
		logger.Debug(errorMsg, log.Error(err))
		serverError := errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.UPDATE_CONSENT_CATEGORY.Code,
			Message:     errors2.UPDATE_CONSENT_CATEGORY.Message,
			Description: errorMsg,
		}, err)
		return serverError
	}

	query := scripts.DeleteConsentCategory[provider.NewDBProvider().GetDBType()]
	_, err = tx.Exec(query, categoryId)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to execute query for deleting consent category: %s", categoryId)
		logger.Debug(errMsg, log.Error(err))
		return errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.UPDATE_CONSENT_CATEGORY.Code,
			Message:     errors2.UPDATE_CONSENT_CATEGORY.Message,
			Description: errMsg,
		}, err)
	}
	return tx.Commit()
}

// GetConsentedCategoryAttributesByProfileId returns the allowed attribute sets for each
// consented category. It only returns attributes for categories the profile has actively consented to.
func GetConsentedCategoryAttributesByProfileId(profileId string, categoryIds []string) (map[string][]model.ConsentAttribute, error) {
	dbClient, err := provider.NewDBProvider().GetDBClient()
	logger := log.GetLogger()
	if err != nil {
		errorMsg := "Failed to get db client for fetching consented category attributes"
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: errorMsg,
		}, err)
	}
	defer dbClient.Close()

	// Fetch which categories the profile has consented to (consent_status = true)
	consentQuery := scripts.GetProfileConsentsByProfileId[provider.NewDBProvider().GetDBType()]
	consentResults, err := dbClient.ExecuteQuery(consentQuery, profileId)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to fetch consents for profile: %s", profileId)
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: errorMsg,
		}, err)
	}

	consentedSet := make(map[string]bool)
	for _, row := range consentResults {
		if status, ok := row["consent_status"].(bool); ok && status {
			consentedSet[row["category_id"].(string)] = true
		}
	}

	// Filter requested categoryIds to only those the profile consented to
	consentedIds := make([]string, 0, len(categoryIds))
	for _, id := range categoryIds {
		if consentedSet[id] {
			consentedIds = append(consentedIds, id)
		}
	}

	if len(consentedIds) == 0 {
		return make(map[string][]model.ConsentAttribute), nil
	}

	return getAttributesByCategoryIds(dbClient, consentedIds)
}

// getAttributesByCategoryIds is an internal helper that fetches attributes for a list of category IDs
// using the provided db client (avoids opening a second connection).
func getAttributesByCategoryIds(dbClient interface {
	ExecuteQuery(query string, args ...interface{}) ([]map[string]interface{}, error)
}, categoryIds []string) (map[string][]model.ConsentAttribute, error) {
	logger := log.GetLogger()

	result := make(map[string][]model.ConsentAttribute)
	if len(categoryIds) == 0 {
		return result, nil
	}

	ids := make([]interface{}, len(categoryIds))
	placeholders := make([]string, len(categoryIds))
	for i, id := range categoryIds {
		ids[i] = id
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	inQuery := fmt.Sprintf(
		"SELECT category_id, scope, attribute_id, app_id FROM consent_category_attributes WHERE category_id IN (%s)",
		strings.Join(placeholders, ", "),
	)

	rows, err := dbClient.ExecuteQuery(inQuery, ids...)
	if err != nil {
		errorMsg := "Failed to fetch consent category attributes"
		logger.Debug(errorMsg, log.Error(err))
		return nil, errors2.NewServerError(errors2.ErrorMessage{
			Code:        errors2.FETCH_CONSENT_CATEGORIES.Code,
			Message:     errors2.FETCH_CONSENT_CATEGORIES.Message,
			Description: errorMsg,
		}, err)
	}

	for _, row := range rows {
		catId := row["category_id"].(string)
		attr := model.ConsentAttribute{
			Scope:       row["scope"].(string),
			AttributeId: row["attribute_id"].(string),
			AppId:       row["app_id"].(string),
		}
		result[catId] = append(result[catId], attr)
	}
	return result, nil
}

func parseStringArray(raw interface{}) []string {
	if raw == nil {
		return nil
	}

	var rawStr string
	switch v := raw.(type) {
	case []byte:
		rawStr = string(v)
	case string:
		rawStr = v
	default:
		return nil
	}

	rawStr = strings.Trim(rawStr, "{}")
	if rawStr == "" {
		return nil
	}

	items := strings.Split(rawStr, ",")
	var result []string
	for _, item := range items {
		// Trim spaces and surrounding double quotes
		clean := strings.TrimSpace(item)
		clean = strings.Trim(clean, `"`)
		result = append(result, clean)
	}

	return result
}
