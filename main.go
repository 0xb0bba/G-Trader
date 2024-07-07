package main

import (
	"strings"
	"sync"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/inventory"
	"xabbo.b7c.io/goearth/shockwave/out"
	"xabbo.b7c.io/goearth/shockwave/profile"
	"xabbo.b7c.io/goearth/shockwave/room"
	"xabbo.b7c.io/goearth/shockwave/trade"
)

var ext = g.NewExt(g.ExtInfo{
	Title:       "G-Trader",
	Description: "Quickly add lots of an item to the trade",
	Version:     "0.4.0",
	Author:      "0xb0bba",
})

var lock sync.Mutex
var inventoryMgr = inventory.NewManager(ext)
var profileMgr = profile.NewManager(ext)
var roomMgr = room.NewManager(ext)
var tradeMgr = trade.NewManager(ext)

func main() {
	ext.Intercept(out.CHAT, out.SHOUT).With(interceptChat)
	ext.Intercept(out.TRADE_ADDITEM).With(interceptTradeAddItem)
	ext.Intercept(out.TRADE_CLOSE).With(interceptTradeClose)
	ext.Connected(func(e g.ConnectArgs) {
		loadExternalTexts(e.Host)
	})

	go loopTrader()
	go loopCounter()
	ext.Run()
}

func interceptChat(e *g.Intercept) {
	msg := e.Packet.ReadString()
	args := strings.Split(msg, " ")
	if args[0] == ":trade" {
		e.Block()
		handleTradeCommand(args)
	}
	if args[0] == ":viewtrade" {
		e.Block()
		handleViewTradeCommand()
	}
	if args[0] == ":count" {
		e.Block()
		handleCountCommand()
	}
	if args[0] == ":countroom" {
		e.Block()
		handleCountRoom()
	}
}
