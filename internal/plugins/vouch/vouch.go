package vouch

import (
	"skyvern/internal/manager"
	"skyvern/internal/storage"
)

func init() {
	p := &VouchPlugin{}
	SharedPlugin = p
	manager.RegisterHelp("vouch", []manager.HelpPage{
		{
			Command:     "Vouch",
			Syntax:      ".vouch <user> [comment]",
			Description: "Vouch for a user to build their reputation.",
		},
		{
			Command:     "Vouch Remove",
			Syntax:      ".vouch remove <user>",
			Description: "Remove your vouch for a user.",
		},
	})
	manager.RegisterHelp("vouches", []manager.HelpPage{
		{
			Command:     "Vouches",
			Syntax:      ".vouches [@user] [page]",
			Description: "List vouches for a user (scrollable).",
		},
	})
}

var SharedPlugin *VouchPlugin

type VouchPlugin struct{}

func (p *VouchPlugin) Name() string { return "vouch" }

func (p *VouchPlugin) Init(db *storage.DB, mgr *manager.Manager) error {
	return nil
}

func (p *VouchPlugin) Commands() []*manager.Command {
	return nil
}

func (p *VouchPlugin) Stop() {}
