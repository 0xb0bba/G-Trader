package main

import (
	"fmt"
	"strconv"
	"time"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/in"
	"xabbo.b7c.io/goearth/shockwave/inventory"
	"xabbo.b7c.io/goearth/shockwave/out"
)

var isInTrade = make(map[int]bool)
var tradingItem = ""
var tradingItemProps = ""
var tradingQty = 0
var targetQty = 0
var didLoop = false
var didBrowse = false
var startedAt = 0
var warnTradeDeclined = false

func init() {
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
}

func handleTradeCommand(args []string) {
	lock.Lock()
	defer lock.Unlock()
	if len(args) < 2 {
		return
	}
	qty, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	targetQty = qty
}

func loopTrader() {
	// Server doesn't allow adding items faster than every ~500ms
	for range time.Tick(time.Millisecond * 550) {
		tickTrader()
	}
}

func tickTrader() {
	lock.Lock()
	defer lock.Unlock()
	if tradingItem == "" || tradingQty >= targetQty {
		return
	}
	found := 0
	for _, item := range inventoryMgr.Items() {
		if _, ok := isInTrade[item.ItemId]; ok {
			continue
		}
		if item.Class != tradingItem || item.Props != tradingItemProps {
			continue
		}
		if found == 0 {
			ext.Send(out.TRADE_ADDITEM, []byte(fmt.Sprintf("%v", item.ItemId)))
		}
		found++
	}
	if found <= 1 {
		if !didLoop {
			ext.Send(out.GETSTRIP, []byte("next"))
			didBrowse = true
		} else {
			tradingItem = ""
			tradingQty = 0
			targetQty = 0
		}
	} else {
		// Update hand so traded items go invisible
		ext.Send(out.GETSTRIP, []byte("update"))
	}
}

func interceptTradeClose(e *g.Intercept) {
	// No need to inform when we are the ones canceling trade
	warnTradeDeclined = false
}

func handleTradeComplete(e *g.Intercept) {
	// This gets sent received before TRADE_CLOSE when both players have accepted
	warnTradeDeclined = false
}

func handleTradeClose(e *g.Intercept) {
	lock.Lock()
	defer lock.Unlock()
	tradingItem = ""
	tradingItemProps = ""
	targetQty = 0
	if warnTradeDeclined {
		warnTradeDeclined = false
		ext.Send(in.SYSTEM_BROADCAST, []byte("Other user cancelled the trade!"))
	}
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
	tradingItemProps = item.Props // most posters share class, id in props
	didLoop = false
	didBrowse = false
	startedAt = item.Pos
}

func handleTradeItems(e *g.Intercept) {
	lock.Lock()
	defer lock.Unlock()
	clear(isInTrade)
	tradingQty = 0
	warnTradeDeclined = false
	// Player who initiated trade comes first
	for i := 0; i < 2; i++ {
		user := e.Packet.ReadString()
		e.Packet.ReadInt() // Status ?
		inv := inventory.Inventory{}
		inv.Parse(e.Packet, &e.Packet.Pos) // List of items in trade
		if profileMgr.Name == user {
			for _, item := range inv.Items {
				isInTrade[item.ItemId] = true
				if item.Class == tradingItem {
					tradingQty++
				}
			}
		} else {
			warnTradeDeclined = len(inv.Items) > 0
		}
	}
	if tradingQty >= targetQty {
		targetQty = 0
		tradingItem = ""
		tradingItemProps = ""
	}
}
