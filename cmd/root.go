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
	"embed"
	"fmt"
	"io"
	"os"

	"github.com/scs/sbom-utility/log"
	"github.com/scs/sbom-utility/schema"
	"github.com/scs/sbom-utility/utils"
	"github.com/spf13/cobra"
)

// Globals
var SchemaFiles embed.FS
var ProjectLogger *log.MiniLogger
var licensePolicyConfig *LicenseComplianceConfig

// top-level commands
const (
	CMD_VERSION  = "version"
	CMD_VALIDATE = "validate"
	CMD_LICENSE  = "license"
	CMD_QUERY    = "query"
)

const (
	CMD_USAGE_VALIDATE     = CMD_VALIDATE + " -i input_file" + "[--force schema_file]"
	CMD_USAGE_QUERY        = CMD_QUERY + " -i input_filename [--select * | field1[,fieldN]] [--from [key1[.keyN]] [--where key=regex[,...]]"
	CMD_USAGE_LICENSE_LIST = SUBCOMMAND_LICENSE_LIST + "  -i input_file [[--summary] [--policy]] [--format json|txt|csv|md]"
)

const (
	FLAG_TRACE                 = "trace"
	FLAG_TRACE_SHORT           = "t"
	FLAG_DEBUG                 = "debug"
	FLAG_DEBUG_SHORT           = "d"
	FLAG_FILENAME_INPUT        = "input-file"
	FLAG_FILENAME_INPUT_SHORT  = "i"
	FLAG_FILENAME_OUTPUT       = "output-file"
	FLAG_FILENAME_OUTPUT_SHORT = "o"
	FLAG_QUIET_MODE            = "quiet"
	FLAG_QUIET_MODE_SHORT      = "q"
	FLAG_LOG_OUTPUT_INDENT     = "indent"
	FLAG_FILE_OUTPUT_FORMAT    = "format"
)

const (
	MSG_APP_NAME        = "Software Bill-of-Materials (SBOM) utility."
	MSG_APP_DESCRIPTION = "This utility serves as centralized command line interface into various Software Bill-of-Materials (SBOM) helper utilities."
	MSG_FLAG_TRACE      = "enable trace logging"
	MSG_FLAG_DEBUG      = "enable debug logging"
	MSG_FLAG_INPUT      = "input filename (e.g., \"path/sbom.json\")"
	MSG_FLAG_OUTPUT     = "output filename"
	MSG_FLAG_LOG_QUIET  = "enable quiet logging mode (e.g., removes all [INFO] messages from output). Overrides other logging commands."
	MSG_FLAG_LOG_INDENT = "enable log indentation of functional callstack."
)

const (
	DEFAULT_SCHEMA_CONFIG            = "config.json"
	DEFAULT_CUSTOM_VALIDATION_CONFIG = "custom.json"
	DEFAULT_LICENSE_POLICIES         = "license.json"
)

// Supported output formats
const (
	OUTPUT_DEFAULT  = ""
	OUTPUT_TEXT     = "txt"
	OUTPUT_JSON     = "json"
	OUTPUT_CSV      = "csv"
	OUTPUT_MARKDOWN = "md"
)

var rootCmd = &cobra.Command{
	Use:           fmt.Sprintf("%s [command] [flags]", utils.GlobalFlags.Project),
	SilenceErrors: false, // TODO: investigate if we should use
	SilenceUsage:  false, // TODO: investigate if we should use
	Short:         MSG_APP_NAME,
	Long:          MSG_APP_DESCRIPTION,
	RunE:          RootCmdImpl,
}

func getLogger() *log.MiniLogger {
	if ProjectLogger == nil {
		// TODO: use LDFLAGS to turn on "TRACE" (and require creation of a Logger)
		// ONLY if needed to debug init() methods in the "cmd" package
		ProjectLogger = log.NewLogger(log.ERROR)

		// Attempt to read in `--args` values such as `--trace`
		// Note: if they exist, quiet mode will be overridden
		// Default to ERROR level and, turn on "Quiet mode" for tests
		// This simplifies the test output to simply RUN/PASS|FAIL messages.
		ProjectLogger.InitLogLevelAndModeFromFlags()
	}
	return ProjectLogger
}

// initialize the module; primarily, initialize cobra
// NOTE: the "cmd" module is problematic as Cobra recommends using init() to configure flags.
func init() {
	// Note: getLogger(): if it is creating the logger, will also
	// initialize the log "level" and set "quiet" mode from command line args.
	getLogger().Enter()
	defer getLogger().Exit()

	// Tell Cobra what our Cobra "init" call back method is
	cobra.OnInitialize(initConfigurations)

	// Declare top-level, persistent flags and where to place the post-parse values
	// TODO: move command help strings to (centralized) constants for better editing/translation across all files
	rootCmd.PersistentFlags().BoolVarP(&utils.GlobalFlags.Trace, FLAG_TRACE, FLAG_TRACE_SHORT, false, MSG_FLAG_TRACE)
	rootCmd.PersistentFlags().BoolVarP(&utils.GlobalFlags.Debug, FLAG_DEBUG, FLAG_DEBUG_SHORT, false, MSG_FLAG_DEBUG)
	rootCmd.PersistentFlags().StringVarP(&utils.GlobalFlags.InputFile, FLAG_FILENAME_INPUT, FLAG_FILENAME_INPUT_SHORT, "", MSG_FLAG_INPUT)
	rootCmd.PersistentFlags().StringVarP(&utils.GlobalFlags.OutputFile, FLAG_FILENAME_OUTPUT, FLAG_FILENAME_OUTPUT_SHORT, "", MSG_FLAG_OUTPUT)

	// NOTE: Although we check for the quiet mode flag in main; we track the flag
	// using Cobra framework in order to enable more comprehensive help
	// and take advantage of other features.
	rootCmd.PersistentFlags().BoolVarP(&utils.GlobalFlags.Quiet, FLAG_QUIET_MODE, FLAG_QUIET_MODE_SHORT, false, MSG_FLAG_LOG_QUIET)

	// Optionally, allow log callstack trace to be indented
	rootCmd.PersistentFlags().BoolVarP(&utils.GlobalFlags.LogOutputIndentCallstack, FLAG_LOG_OUTPUT_INDENT, "", false, MSG_FLAG_LOG_INDENT)

	// Add root commands
	rootCmd.AddCommand(NewCommandVersion())
	rootCmd.AddCommand(NewCommandSchema())
	rootCmd.AddCommand(NewCommandValidate())
	rootCmd.AddCommand(NewCommandQuery())

	// Add license command its subcommands
	licenseCmd := NewCommandLicense()
	licenseCmd.AddCommand(NewCommandList())
	licenseCmd.AddCommand(NewCommandPolicy())
	rootCmd.AddCommand(licenseCmd)
}

// load and process configuration files.  Processing includes JSON unmarshalling and hashing.
// includes JSON files:
// config.json (SBOM format/schema definitions),
// license.json (license policy definitions),
// custom.json (custom validation settings)
func initConfigurations() {
	getLogger().Enter()
	defer getLogger().Exit()

	// Print global flags in debug mode
	flagInfo, err := getLogger().FormatStructE(utils.GlobalFlags)
	if err != nil {
		getLogger().Error(err.Error())
	} else {
		getLogger().Debugf("%s: \n%s", "utils.Flags", flagInfo)
	}

	// NOTE: some commands operate just on JSON SBOM (i.e., no validation)
	// we leave the code below "in place" as we may still want to validate any
	// input file as JSON SBOM document that matches a known format/version (in the future)

	// Load application configuration file (i.e., primarily SBOM supported Formats/Schemas)
	// TODO: page fault "load" of data only when needed
	errCfg := schema.LoadFormatBasedSchemas(DEFAULT_SCHEMA_CONFIG)
	if errCfg != nil {
		getLogger().Error(errCfg.Error())
		os.Exit(ERROR_APPLICATION)
	}

	// Load custom validation file
	// TODO: page fault "load" of data only when needed (sync.Once)
	errCfg = schema.LoadCustomValidationConfig(DEFAULT_CUSTOM_VALIDATION_CONFIG)
	if errCfg != nil {
		getLogger().Error(errCfg.Error())
		os.Exit(ERROR_APPLICATION)
	}

	// i.e., License approval policies
	if utils.GlobalFlags.LicensePolicyConfigFile == "" {
		utils.GlobalFlags.LicensePolicyConfigFile = DEFAULT_LICENSE_POLICIES
	}

	licensePolicyConfig = new(LicenseComplianceConfig)
	errPolicies := licensePolicyConfig.LoadLicensePolicies(utils.GlobalFlags.LicensePolicyConfigFile)
	if errPolicies != nil {
		getLogger().Error(errPolicies.Error())
		os.Exit(ERROR_APPLICATION)
	}
}

func RootCmdImpl(cmd *cobra.Command, args []string) error {
	getLogger().Enter()
	defer getLogger().Exit()

	// no commands (empty) passed; display help
	if len(args) == 0 {
		cmd.Help()
		os.Exit(ERROR_APPLICATION)
	}
	return nil
}

func Execute() {
	// instead of creating a dependency on the "main" module
	getLogger().Enter()
	defer getLogger().Exit()

	if err := rootCmd.Execute(); err != nil {
		if IsInvalidSBOMError(err) {
			os.Exit(ERROR_VALIDATION)
		} else {
			os.Exit(ERROR_APPLICATION)
		}
	}
}

// Command PreRunE helper function to test for input file
func preRunTestForInputFile(cmd *cobra.Command, args []string) error {
	getLogger().Enter()
	defer getLogger().Exit()
	getLogger().Tracef("args: %v", args)

	// Make sure the input filename is present and exists
	file := utils.GlobalFlags.InputFile
	if file == "" {
		return getLogger().Errorf("Missing required argument(s): %s", FLAG_FILENAME_INPUT)
	} else if _, err := os.Stat(file); err != nil {
		return getLogger().Errorf("File not found: `%s`", file)
	}
	return nil
}

func createOutputFile(outputFilename string) (outputFile *os.File, writer io.Writer, err error) {
	// default to Stdout
	writer = os.Stdout

	// If command included an output file, attempt to create it and create a writer
	if outputFilename != "" {
		getLogger().Infof("Creating output file: `%s`...", outputFilename)
		outputFile, err = os.Create(outputFilename)
		if err != nil {
			getLogger().Error(err)
		}
		writer = outputFile
	}
	return
}

// Misc. common funcs
// Note: intend to use for truncation of fields in (text) reports/lists
func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
