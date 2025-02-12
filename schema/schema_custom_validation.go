/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package schema

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/scs/sbom-utility/utils"
)

// Globals
var CustomValidationChecks CustomValidationConfig

// ---------------------------------------------------------------
// Custom Validation
// ---------------------------------------------------------------

func LoadCustomValidationConfig(filename string) error {
	getLogger().Enter()
	defer getLogger().Exit()

	var cfgFilename string

	// validate filename
	if len(filename) == 0 {
		return fmt.Errorf("config: invalid filename: `%s`", filename)
	}

	// Conditionally append working directory if no abs. path detected
	if len(filename) > 0 && filename[0] != '/' {
		cfgFilename = utils.GlobalFlags.WorkingDir + "/" + filename
	} else {
		cfgFilename = filename
	}

	buffer, err := ioutil.ReadFile(cfgFilename)
	if err != nil {
		return fmt.Errorf("config: unable to `ReadFile`: `%s`", cfgFilename)
	}

	err = json.Unmarshal(buffer, &CustomValidationChecks)
	if err != nil {
		return fmt.Errorf("config: cannot `Unmarshal`: `%s`", cfgFilename)
	}
	return nil
}

// TODO: return copies
func (config *CustomValidationConfig) GetCustomValidationConfig() *CustomValidation {
	return &config.Validation
}

func (config *CustomValidationConfig) GetCustomValidationMetadata() *CustomValidationMetadata {

	if cfg := config.GetCustomValidationConfig(); cfg != nil {
		return &cfg.Metadata
	}
	return nil
}

func (config *CustomValidationConfig) GetCustomValidationMetadataProperties() []CustomValidationProperty {

	if metadata := config.GetCustomValidationMetadata(); metadata != nil {
		return metadata.Properties
	}
	return nil
}

type CustomValidationConfig struct {
	Validation CustomValidation `json:"validation"`
}

// Custom Validation config.
type CustomValidation struct {
	Metadata CustomValidationMetadata `json:"metadata"`
}

type CustomValidationMetadata struct {
	Properties []CustomValidationProperty `json:"properties"`
	Tools      []CustomValidationTool     `json:"tools"`
}

// NOTE: Assumes property "key" is the value in the "name" field
type CustomValidationProperty struct {
	CDXProperty
	Description string `json:"_validate_description"`
	Key         string `json:"_validate_key"`
	CheckUnique string `json:"_validate_unique"`
	CheckRegex  string `json:"_validate_regex"`
}

type CustomValidationTool struct {
	CDXTool
	Description string `json:"_validate_description"`
}
