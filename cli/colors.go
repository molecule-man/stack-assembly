package cli

import "github.com/fatih/color"

var NoColor = color.New()
var SuccessColor = color.New(color.FgGreen)
var NeutralColor = color.New(color.FgCyan)
var FailureColor = color.New(color.FgRed, color.Bold)
var WarnColor = color.New(color.FgYellow)
