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

package cmd

import (
	"fmt"

	"github.com/scs/sbom-utility/schema"
	"github.com/scs/sbom-utility/utils"
)

func LoadInputSbomFileAndDetectSchema() (document *schema.Sbom, err error) {
	getLogger().Enter()
	defer getLogger().Exit()

	// check for required fields on command
	getLogger().Tracef("utils.Flags.InputFile: `%s`", utils.GlobalFlags.InputFile)
	if utils.GlobalFlags.InputFile == "" {
		return nil, fmt.Errorf("invalid input file (-%s): `%s` ", FLAG_FILENAME_INPUT_SHORT, utils.GlobalFlags.InputFile)
	}

	// Construct an Sbom object around the input file
	document = schema.NewSbom(utils.GlobalFlags.InputFile)

	// Load the raw, candidate SBOM (file) as JSON data
	getLogger().Infof("Unmarshalling file `%s`...", utils.GlobalFlags.InputFile)
	err = document.UnmarshalSBOMAsJsonMap() // i.e., utils.Flags.InputFile
	if err != nil {
		return
	}

	// Search the document keys/values for known SBOM formats and schema in the config. file
	getLogger().Infof("Determining file's sbom format and version...")
	err = document.FindFormatAndSchema()
	if err != nil {
		return
	}

	return
}
