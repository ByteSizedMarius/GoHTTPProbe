package app

import "github.com/spf13/cobra"

// Flag interface defines methods all flag types must implement
type Flag interface {
	Register(cmd *cobra.Command)
	GetLongName() string
	GetShortName() string
	GetDescription() string
}

// BaseFlag contains common flag properties
type BaseFlag struct {
	LongName    string
	ShortName   string
	Description string
}

func (f BaseFlag) GetLongName() string    { return f.LongName }
func (f BaseFlag) GetShortName() string   { return f.ShortName }
func (f BaseFlag) GetDescription() string { return f.Description }

// StringFlag for string values
type StringFlag struct {
	BaseFlag
	Default string
	Target  *string
}

func (f StringFlag) Register(cmd *cobra.Command) {
	cmd.Flags().StringVarP(f.Target, f.LongName, f.ShortName, f.Default, f.Description)
}

// BoolFlag for boolean values
type BoolFlag struct {
	BaseFlag
	Default bool
	Target  *bool
}

func (f BoolFlag) Register(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(f.Target, f.LongName, f.ShortName, f.Default, f.Description)
}

// IntFlag for integer values
type IntFlag struct {
	BaseFlag
	Default int
	Target  *int
}

func (f IntFlag) Register(cmd *cobra.Command) {
	cmd.Flags().IntVarP(f.Target, f.LongName, f.ShortName, f.Default, f.Description)
}

// StringArrayFlag for string array values
type StringArrayFlag struct {
	BaseFlag
	Default []string
	Target  *[]string
}

func (f StringArrayFlag) Register(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(f.Target, f.LongName, f.ShortName, f.Default, f.Description)
}

// StringPFlag for string values without a target variable
type StringPFlag struct {
	BaseFlag
	Default string
}

func (f StringPFlag) Register(cmd *cobra.Command) {
	cmd.Flags().StringP(f.LongName, f.ShortName, f.Default, f.Description)
}
