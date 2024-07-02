package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/in"
	"xabbo.b7c.io/goearth/shockwave/inventory"
	"xabbo.b7c.io/goearth/shockwave/out"
)

var ext = g.NewExt(g.ExtInfo{
	Title:       "G-Trader",
	Description: "Quickly add lots of an item to the trade",
	Version:     "0.1.0",
	Author:      "0xb0bba",
})

var lock sync.Mutex
var inventoryMgr = inventory.NewManager(ext)
var userName = ""
var isInTrade = make(map[int]bool)
var tradingItem = ""
var tradingQty = 0
var targetQty = 0
var didLoop = false
var didBrowse = false
var startedAt = 0

func main() {
	ext.Intercept(out.CHAT, out.SHOUT).With(interceptChat)
	ext.Intercept(out.TRADE_ADDITEM).With(interceptTradeAddItem)
	ext.Intercept(in.USER_OBJ).With(handleUserObj)
	ext.Intercept(in.TRADE_CLOSE).With(handleTradeClose)
	ext.Intercept(in.TRADE_ITEMS).With(handleTradeItems)
	go offerItems()
	ext.Run()
}

func interceptChat(e *g.Intercept) {
	lock.Lock()
	defer lock.Unlock()
	msg := e.Packet.ReadString()
	parts := strings.Split(msg, " ")
	if parts[0] == ":trade" {
		e.Block()
		if len(parts) < 2 {
			return
		}
		qty, err := strconv.Atoi(parts[1])
		if err != nil {
			return
		}
		targetQty = qty
	}
}

func handleUserObj(e *g.Intercept) {
	lock.Lock()
	defer lock.Unlock()
	// We need to capture our own userName to know which items are ours in the trade screen
	user := e.Packet.ReadString()
	params := strings.Split(user, "\r")
	for _, param := range params {
		parts := strings.Split(param, "=")
		if parts[0] == "name" {
			userName = parts[1]
		}
	}
}

func offerItems() {
	// In case we don't have enough furni we need to detect when we run out so we don't just keep looping around
	inventoryMgr.Updated(func() {
		lock.Lock()
		defer lock.Unlock()
		if didBrowse {
			for _, item := range inventoryMgr.Items() {
				if item.Pos == startedAt {
					// Looped around the whole hand
					didLoop = true
				}
			}
		}
	})
	// Server doesn't allow adding items faster than every ~500ms
	for range time.Tick(time.Millisecond * 600) {
		tick()
	}
}

func tick() {
	lock.Lock()
	defer lock.Unlock()
	if tradingItem == "" || tradingQty >= targetQty {
		return
	}
	found := false
	for _, item := range inventoryMgr.Items() {
		if _, ok := isInTrade[item.ItemId]; ok {
			continue
		}
		if item.Class != tradingItem {
			continue
		}
		ext.Send(out.TRADE_ADDITEM, []byte(fmt.Sprintf("%v", item.ItemId)))
		ext.Send(out.GETSTRIP, []byte("update")) // Not strictly needed but hides items in hand, looks better for user
		found = true
	}
	if !found && !didLoop {
		ext.Send(out.GETSTRIP, []byte("next"))
		didBrowse = true
	}
}

func handleTradeClose(e *g.Intercept) {
	lock.Lock()
	defer lock.Unlock()
	tradingItem = ""
	targetQty = 0
}

func interceptTradeAddItem(e *g.Intercept) {
	lock.Lock()
	defer lock.Unlock()
	stripId, err := strconv.Atoi(string(e.Packet.ReadBytesAt(0, e.Packet.Length())))
	if err != nil {
		return
	}
	inv := inventoryMgr.Items()
	item, ok := inv[stripId]
	if !ok {
		return
	}
	tradingItem = item.Class
	didLoop = false
	didBrowse = false
	startedAt = item.Pos
}

func handleTradeItems(e *g.Intercept) {
	lock.Lock()
	defer lock.Unlock()
	if userName == "" {
		// Extension wasn't loaded during login
		ext.Send(out.INFORETRIEVE)
	}
	clear(isInTrade)
	tradingQty = 0
	// Player who initiated trade comes first
	for i := 0; i < 2; i++ {
		user := e.Packet.ReadString()
		e.Packet.ReadInt() // Status ?
		inv := inventory.Inventory{}
		inv.Parse(e.Packet, &e.Packet.Pos) // List of items in trade
		if userName == user {
			for _, item := range inv.Items {
				isInTrade[item.ItemId] = true
				if item.Class == tradingItem {
					tradingQty++
				}
			}
		}
	}
	if tradingQty >= targetQty {
		targetQty = 0
		tradingItem = ""
	}
}
