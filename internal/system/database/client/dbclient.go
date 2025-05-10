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

package client

import (
	"database/sql"
	"strings"

	_ "github.com/lib/pq"
)

// DBClientInterface defines the interface for database operations.
type DBClientInterface interface {
	ExecuteQuery(query string, args ...interface{}) ([]map[string]interface{}, error)
	BeginTx() (*sql.Tx, error)
	Close() error
}

// DBClient is the implementation of DBClientInterface.
type DBClient struct {
	db *sql.DB
}

// NewDBClient creates a new instance of DBClient with the provided database connection.
func NewDBClient(db *sql.DB) DBClientInterface {

	return &DBClient{
		db: db,
	}
}

// ExecuteQuery executes a SELECT query and returns the result as a slice of maps.
func (client *DBClient) ExecuteQuery(query string, args ...interface{}) ([]map[string]interface{}, error) {

	rows, err := client.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		row := make([]interface{}, len(columns))
		rowPointers := make([]interface{}, len(columns))
		for i := range row {
			rowPointers[i] = &row[i]
		}

		if err := rows.Scan(rowPointers...); err != nil {
			return nil, err
		}

		result := map[string]interface{}{}
		for i, col := range columns {
			// Normalize column names to lowercase for consistency.
			result[strings.ToLower(col)] = row[i]
		}
		results = append(results, result)
	}

	return results, nil
}

// BeginTx starts a new database transaction.
func (client *DBClient) BeginTx() (*sql.Tx, error) {

	return client.db.Begin()
}

// Close closes the database connection.
func (client *DBClient) Close() error {

	return client.db.Close()
}
