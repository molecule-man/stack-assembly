package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/molecule-man/stack-assembly/cmd/conf"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/cobra"
)

func infoCmd() *cobra.Command {
	infoCmd := &cobra.Command{
		Use:   "info [stack name]",
		Short: "Show info about the stack",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfgFiles, err := cmd.Parent().PersistentFlags().GetStringSlice("configs")
			handleError(err)

			cfg := conf.LoadConfig(cfgFiles)

			serv := conf.InitStasService(cfg)

			for k, template := range cfg.Templates {

				if len(args) > 0 && args[0] != template.Name && args[0] != k {
					continue
				}

				tpl := stackassembly.StackTemplate{Name: template.Name, Params: template.Parameters}

				info, err := serv.Info(tpl, cfg.Parameters)
				handleError(err)

				displayStackInfo(info)
			}
		},
	}

	return infoCmd
}

func displayStackInfo(info stackassembly.StackInfo) {

	fmt.Println("===================")
	fmt.Printf("STACK: %s\n", info.Name)
	fmt.Println("===================")
	fmt.Println("")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Println("==== RESOURCES ====")
	for _, res := range info.Resources {
		fmt.Fprintf(w, "%s\t%s\t%s\n", res.LogicalID, res.PhysicalID, res.Status)
	}
	w.Flush()
	fmt.Println("")

	fmt.Println("==== OUTPUTS ====")
	for _, out := range info.Outputs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", out.Key, out.Value, out.ExportName)
	}
	w.Flush()
	fmt.Println("")

	fmt.Println("==== EVENTS ====")
	for _, e := range info.Events[:10] {
		fmt.Fprintf(w, "[%v]\t%s\t%s\t%s\t%s\n", e.Timestamp, e.ResourceType, e.LogicalResourceID, e.Status, e.StatusReason)
	}
	w.Flush()
	fmt.Print("\n\n")
}
