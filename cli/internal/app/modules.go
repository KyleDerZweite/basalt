// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"github.com/KyleDerZweite/basalt/internal/config"
	"github.com/KyleDerZweite/basalt/internal/modules"
	"github.com/KyleDerZweite/basalt/internal/modules/beacons"
	"github.com/KyleDerZweite/basalt/internal/modules/bento"
	"github.com/KyleDerZweite/basalt/internal/modules/carrd"
	"github.com/KyleDerZweite/basalt/internal/modules/chesscom"
	"github.com/KyleDerZweite/basalt/internal/modules/codeberg"
	"github.com/KyleDerZweite/basalt/internal/modules/codeforces"
	"github.com/KyleDerZweite/basalt/internal/modules/devto"
	"github.com/KyleDerZweite/basalt/internal/modules/discord"
	"github.com/KyleDerZweite/basalt/internal/modules/dnsct"
	"github.com/KyleDerZweite/basalt/internal/modules/dockerhub"
	"github.com/KyleDerZweite/basalt/internal/modules/github"
	"github.com/KyleDerZweite/basalt/internal/modules/gitlab"
	"github.com/KyleDerZweite/basalt/internal/modules/gravatar"
	"github.com/KyleDerZweite/basalt/internal/modules/hackernews"
	"github.com/KyleDerZweite/basalt/internal/modules/instagram"
	"github.com/KyleDerZweite/basalt/internal/modules/ipinfo"
	"github.com/KyleDerZweite/basalt/internal/modules/keybase"
	"github.com/KyleDerZweite/basalt/internal/modules/lichess"
	"github.com/KyleDerZweite/basalt/internal/modules/linktree"
	"github.com/KyleDerZweite/basalt/internal/modules/matrix"
	"github.com/KyleDerZweite/basalt/internal/modules/medium"
	"github.com/KyleDerZweite/basalt/internal/modules/myanimelist"
	"github.com/KyleDerZweite/basalt/internal/modules/opgg"
	"github.com/KyleDerZweite/basalt/internal/modules/reddit"
	"github.com/KyleDerZweite/basalt/internal/modules/roblox"
	"github.com/KyleDerZweite/basalt/internal/modules/securitytxt"
	"github.com/KyleDerZweite/basalt/internal/modules/shodan"
	"github.com/KyleDerZweite/basalt/internal/modules/spotify"
	"github.com/KyleDerZweite/basalt/internal/modules/stackexchange"
	"github.com/KyleDerZweite/basalt/internal/modules/steam"
	"github.com/KyleDerZweite/basalt/internal/modules/telegram"
	"github.com/KyleDerZweite/basalt/internal/modules/tiktok"
	"github.com/KyleDerZweite/basalt/internal/modules/trello"
	"github.com/KyleDerZweite/basalt/internal/modules/twitch"
	"github.com/KyleDerZweite/basalt/internal/modules/wattpad"
	"github.com/KyleDerZweite/basalt/internal/modules/wayback"
	"github.com/KyleDerZweite/basalt/internal/modules/whois"
	"github.com/KyleDerZweite/basalt/internal/modules/youtube"
)

type moduleFactory func(*config.Config) modules.Module

var moduleFactories = []moduleFactory{
	func(*config.Config) modules.Module { return gravatar.New() },
	func(*config.Config) modules.Module { return linktree.New() },
	func(*config.Config) modules.Module { return beacons.New() },
	func(*config.Config) modules.Module { return carrd.New() },
	func(*config.Config) modules.Module { return bento.New() },
	func(cfg *config.Config) modules.Module { return github.New(cfg.Get("GITHUB_TOKEN")) },
	func(*config.Config) modules.Module { return gitlab.New() },
	func(*config.Config) modules.Module { return codeberg.New() },
	func(*config.Config) modules.Module { return codeforces.New() },
	func(*config.Config) modules.Module { return stackexchange.New() },
	func(*config.Config) modules.Module { return reddit.New() },
	func(*config.Config) modules.Module { return roblox.New() },
	func(*config.Config) modules.Module { return youtube.New() },
	func(*config.Config) modules.Module { return twitch.New() },
	func(*config.Config) modules.Module { return discord.New() },
	func(*config.Config) modules.Module { return instagram.New() },
	func(*config.Config) modules.Module { return tiktok.New() },
	func(*config.Config) modules.Module { return matrix.New() },
	func(*config.Config) modules.Module { return steam.New() },
	func(*config.Config) modules.Module { return spotify.New() },
	func(*config.Config) modules.Module { return telegram.New() },
	func(*config.Config) modules.Module { return medium.New() },
	func(*config.Config) modules.Module { return myanimelist.New() },
	func(*config.Config) modules.Module { return opgg.New() },
	func(*config.Config) modules.Module { return keybase.New() },
	func(*config.Config) modules.Module { return hackernews.New() },
	func(*config.Config) modules.Module { return dockerhub.New() },
	func(*config.Config) modules.Module { return devto.New() },
	func(*config.Config) modules.Module { return chesscom.New() },
	func(*config.Config) modules.Module { return lichess.New() },
	func(*config.Config) modules.Module { return trello.New() },
	func(*config.Config) modules.Module { return wattpad.New() },
	func(*config.Config) modules.Module { return whois.New() },
	func(*config.Config) modules.Module { return dnsct.New() },
	func(*config.Config) modules.Module { return securitytxt.New() },
	func(*config.Config) modules.Module { return shodan.New() },
	func(*config.Config) modules.Module { return wayback.New() },
	func(*config.Config) modules.Module { return ipinfo.New() },
}

func buildRegistry(cfg *config.Config, disabled map[string]struct{}) *modules.Registry {
	reg := modules.NewRegistry()
	for _, factory := range moduleFactories {
		module := factory(cfg)
		if _, blocked := disabled[module.Name()]; blocked {
			continue
		}
		reg.Register(module)
	}
	return reg
}
