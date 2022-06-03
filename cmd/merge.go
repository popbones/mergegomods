package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
)

// mergeCmd represents the merge command
var mergeCmd = &cobra.Command{
	Use:   "merge [flags] input_files...",
	Short: "performs merging",
	Run: func(cmd *cobra.Command, args []string) {
		t, err := buildMergeTask(cmd, args)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		if err := t.Run(); err != nil {
			log.Println(err)
			os.Exit(2)
		}
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)

	mergeCmd.Flags().StringP("package", "p", "",
		"target package name")
	mergeCmd.Flags().StringP("version", "v", "",
		"target go version")
}

func buildMergeTask(cmd *cobra.Command, args []string) (t *MergeTask, err error) {
	t = &MergeTask{}
	if t.OutputPackageName, err = cmd.Flags().GetString("package"); err != nil {
		return
	}
	if t.OutputGoVersion, err = cmd.Flags().GetString("version"); err != nil {
		return
	}
	t.InputGoModFiles = map[string]bool{}
	for _, filename := range args {
		if _, ok := t.InputGoModFiles[filename]; ok {
			continue
		}
		if s, err := os.Stat(filename); err != nil {
			return nil, err
		} else if s.IsDir() {
			return t, fmt.Errorf("%s is a dir", filename)
		}
		t.InputGoModFiles[filename] = true
	}
	return
}

type MergeTask struct {
	OutputPackageName string
	OutputGoVersion   string
	InputGoModFiles   map[string]bool
}

func (t *MergeTask) Run() error {
	var err error
	// init target file
	modFile, err := modfile.Parse("go.mod", []byte{}, nil)
	if err != nil {
		return err
	}

	if t.OutputPackageName != "" {
		if err := modFile.AddModuleStmt(t.OutputPackageName); err != nil {
			return err
		}
	}

	// copy
	for name := range t.InputGoModFiles {
		inputModFile, err := loadGoModFile(name)
		if err != nil {
			return err
		}

		for _, r := range inputModFile.Require {
			modFile.AddNewRequire(r.Mod.Path, r.Mod.Version, false)
		}
		for _, x := range inputModFile.Exclude {
			if err := modFile.AddExclude(x.Mod.Path, x.Mod.Version); err != nil {
				return err
			}
		}
		for _, r := range inputModFile.Replace {
			if err := modFile.AddReplace(r.Old.Path, r.Old.Version, r.New.Path, r.New.Version); err != nil {
				return err
			}
		}
		for _, r := range inputModFile.Retract {
			if err := modFile.AddRetract(r.VersionInterval, r.Rationale); err != nil {
				return err
			}
		}

		if err := modFile.AddGoStmt(inputModFile.Go.Version); err != nil {
			return err
		}
	}

	if t.OutputGoVersion != "" {
		if err := modFile.AddGoStmt(t.OutputGoVersion); err != nil {
			return err
		}
	}

	modFile.Cleanup()
	modFile.SortBlocks()

	data := modfile.Format(modFile.Syntax)
	fmt.Println(string(data))
	return nil
}

func loadGoModFile(name string) (*modfile.File, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return modfile.Parse(name, data, nil)
}
