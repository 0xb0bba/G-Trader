package main

import (
	"strconv"
	"time"
	g "xabbo.b7c.io/goearth"
	"xabbo.b7c.io/goearth/shockwave/in"
	"xabbo.b7c.io/goearth/shockwave/out"
	"xabbo.b7c.io/goearth/shockwave/trade"
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
var lastTrade trade.Offers

func init() {
	inventoryMgr.Updated(handleInventoryUpdated)
	tradeMgr.Updated(handleTradeItems)
	tradeMgr.Closed(handleTradeClose)
	tradeMgr.Completed(handleTradeComplete)
}

func handleInventoryUpdated() {
	lock.Lock()
	defer lock.Unlock()
	// In case we don't have enough furni we need to detect when we run out so we don't just keep looping around
	if didBrowse {
		for _, item := range inventoryMgr.Items() {
			if item.Pos == startedAt {
				// Looped around the whole hand
				didLoop = true
			}
		}
	}
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

func handleViewTradeCommand() {
	var offers trade.Offers
	if tradeMgr.Trading {
		offers = tradeMgr.Offers
	} else {
		offers = lastTrade
	}
	for _, offer := range offers {
		if offer.Name != profileMgr.Name {
			counts := make(map[string]int)
			for _, item := range offer.Items {
				name := getFullName(item)
				counts[name] = counts[name] + 1
			}
			printCountResults(counts)
		}
	}
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
			tradeMgr.OfferItem(item)
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
		inventoryMgr.Update()
	}
}

func interceptTradeClose(e *g.Intercept) {
	// No need to inform when we are the ones canceling trade
	warnTradeDeclined = false
}

func handleTradeComplete(args trade.Args) {
	// This gets sent received before TRADE_CLOSE when both players have accepted
	warnTradeDeclined = false
	lastTrade = args.Offers
}

func handleTradeClose(args trade.Args) {
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

func handleTradeItems(args trade.Args) {
	lock.Lock()
	defer lock.Unlock()
	clear(isInTrade)
	tradingQty = 0
	warnTradeDeclined = false
	// Player who initiated trade comes first
	for i := 0; i < 2; i++ {
		inv := args.Offers[i]
		if profileMgr.Name == inv.Name {
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
