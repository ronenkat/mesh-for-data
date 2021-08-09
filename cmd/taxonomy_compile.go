// Copyright 2020 IBM Corp.
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fybrik.io/fybrik/pkg/taxonomy/compile"
	taxonomyio "fybrik.io/fybrik/pkg/taxonomy/io"
	"github.com/spf13/cobra"
)

var (
	taxonomyCompileBasePath string
	taxonomyCompileOutPath  string
	taxonomyCompileCodegen  bool
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile --out <outputFile> --base <baseFile> [<layerFile> ...] [--codegen]",
	Short: "Generate a taxonomy.json file from base taxonomy and taxonomy layers",
	Args:  cobra.ArbitraryArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := compile.Files(taxonomyCompileBasePath, args, compile.WithCodeGenerationTarget(taxonomyCompileCodegen))
		if err != nil {
			return err
		}
		return taxonomyio.WriteDocumentToFile(result, taxonomyCompileOutPath)
	},
	DisableFlagsInUseLine: true,
}

func init() {
	taxonomyCmd.AddCommand(compileCmd)

	compileCmd.Flags().StringVarP(&taxonomyCompileBasePath, "base", "b", "", "File with base taxonomy definitions (required)")
	_ = compileCmd.MarkFlagFilename("base", "yaml", "yml", "json")
	_ = compileCmd.MarkFlagRequired("base")

	compileCmd.Flags().StringVarP(&taxonomyCompileOutPath, "out", "o", "taxonomy.json", "Path for output file")
	_ = compileCmd.MarkFlagFilename("out", "yaml", "yml", "json")

	compileCmd.Flags().BoolVar(&taxonomyCompileCodegen, "codegen", false,
		"Best effort to make output suitable for code generation tools")
}
