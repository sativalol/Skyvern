package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"

	"skyvern/internal/commands"
	"skyvern/internal/manager"
	"skyvern/internal/plugins"
)

type helpPageInfo struct {
	Command     string `json:"command"`
	Syntax      string `json:"syntax"`
	Description string `json:"description"`
}

type cmdInfo struct {
	Trigger     string         `json:"trigger"`
	Aliases     []string       `json:"aliases,omitempty"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	HelpPages   []helpPageInfo `json:"help_pages,omitempty"`
}

type dumpOutput struct {
	Count    int       `json:"count"`
	Commands []cmdInfo `json:"commands"`
}

func HandleDumpCmds() {
	if !(len(os.Args) > 1 && (os.Args[1] == "--dump-cmds" || os.Args[1] == "dump")) {
		return
	}

	var list []cmdInfo
	for _, cmd := range commands.Registry {
		var hPages []helpPageInfo
		if pages, ok := manager.GetHelp(cmd.Trigger); ok {
			for _, p := range pages {
				hPages = append(hPages, helpPageInfo{
					Command:     p.Command,
					Syntax:      p.Syntax,
					Description: p.Description,
				})
			}
		}
		list = append(list, cmdInfo{
			Trigger:     cmd.Trigger,
			Aliases:     cmd.Aliases,
			Name:        cmd.Name,
			Description: cmd.Description,
			Category:    cmd.Category,
			HelpPages:   hPages,
		})
	}
	for _, p := range plugins.Loaded() {
		for _, cmd := range p.Commands() {
			var hPages []helpPageInfo
			if pages, ok := manager.GetHelp(cmd.Trigger); ok {
				for _, hp := range pages {
					hPages = append(hPages, helpPageInfo{
						Command:     hp.Command,
						Syntax:      hp.Syntax,
						Description: hp.Description,
					})
				}
			}
			list = append(list, cmdInfo{
				Trigger:     cmd.Trigger,
				Aliases:     cmd.Aliases,
				Name:        cmd.Name,
				Description: cmd.Description,
				Category:    cmd.Category,
				HelpPages:   hPages,
			})
		}
	}
	out := dumpOutput{
		Count:    len(list),
		Commands: list,
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("cmds.json", b, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Dumped all %d commands to cmds.json\n", len(list))
	os.Exit(0)
}
