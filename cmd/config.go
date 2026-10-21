package cmd

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/godot-services/gd/internal/godot"
	"github.com/spf13/cobra"
)

var errSettingNotSet = errors.New("is not set in project.godot (Godot uses its default)")

var (
	numberRe      = regexp.MustCompile(`^-?\d+(\.\d+)?([eE][+-]?\d+)?$`)
	constructorRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\(`)
)

// inferValue turns a command line argument into raw project.godot value text.
func inferValue(current string, hasCurrent bool, arg string) string {
	if hasCurrent && isDecodableString(current) {
		if strings.HasPrefix(arg, `"`) {
			return arg
		}

		return godot.EncodeString(arg)
	}

	switch arg {
	case "true", "false", "null":
		return arg
	}

	if numberRe.MatchString(arg) || constructorRe.MatchString(arg) {
		return arg
	}

	for _, p := range []string{`"`, `&"`, `^"`, "[", "{"} {
		if strings.HasPrefix(arg, p) {
			return arg
		}
	}

	return godot.EncodeString(arg)
}

func isDecodableString(raw string) bool {
	_, ok := godot.DecodeString(raw)

	return ok
}

// loadSettings reads project.godot from the current project.
func loadSettings() (*godot.ProjectSettings, error) {
	p, err := godot.NewProject(".")
	if err != nil {
		return nil, err
	}

	return godot.LoadProjectSettings(p.Location)
}

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{Use: "config", Short: "Read and change project.godot settings"}

	c.AddCommand(
		&cobra.Command{
			Use:   "get <path>",
			Short: "Print a setting",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				s, err := loadSettings()
				if err != nil {
					return err
				}

				raw, ok := s.Get(args[0])
				if !ok {
					return fmt.Errorf("%s %w", args[0], errSettingNotSet)
				}
				if str, isString := godot.DecodeString(raw); isString {
					raw = str
				}

				fmt.Fprintln(cmd.OutOrStdout(), raw)

				return nil
			},
		},
		&cobra.Command{
			Use:   "set <path> <value>",
			Short: "Set a setting",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				s, err := loadSettings()
				if err != nil {
					return err
				}

				current, has := s.Get(args[0])
				if err := s.Set(args[0], inferValue(current, has, args[1])); err != nil {
					return err
				}
				if err := s.Save(); err != nil {
					return err
				}

				cmd.Printf("set %s\n", args[0])

				return nil
			},
		},
		&cobra.Command{
			Use:   "unset <path>",
			Short: "Remove a setting",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				s, err := loadSettings()
				if err != nil {
					return err
				}
				if !s.Unset(args[0]) {
					return fmt.Errorf("%s %w", args[0], errSettingNotSet)
				}
				if err := s.Save(); err != nil {
					return err
				}

				cmd.Printf("unset %s\n", args[0])

				return nil
			},
		},
		&cobra.Command{
			Use:   "list [prefix]",
			Short: "List settings",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				s, err := loadSettings()
				if err != nil {
					return err
				}

				prefix := ""
				if len(args) > 0 {
					prefix = args[0]
				}

				for _, e := range s.List() {
					if strings.HasPrefix(e.Path, prefix) {
						fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", e.Path, e.Raw)
					}
				}

				return nil
			},
		},
	)

	return c
}
