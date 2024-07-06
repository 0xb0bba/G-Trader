package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"xabbo.b7c.io/goearth/shockwave/in"
	"xabbo.b7c.io/goearth/shockwave/inventory"
	"xabbo.b7c.io/goearth/shockwave/out"
)

var isCountingHand = false
var counts = make(map[string]int)
var isCounted = make(map[int]bool)
var refreshed = false

func handleCountCommand() {
	lock.Lock()
	defer lock.Unlock()
	clear(counts)
	clear(isCounted)
	refreshed = false
	isCountingHand = true
}

func handleCountRoom() {
	counts := make(map[string]int)
	for _, item := range roomMgr.Objects {
		name := fmt.Sprintf("%v (%v)", getFurniName(item.Class), item.Class)
		counts[name] = counts[name] + 1
	}
	for _, poster := range roomMgr.Items {
		name, key := getPosterName(poster.Class, poster.Type)
		name = fmt.Sprintf("%v (%v)", name, key)
		counts[name] = counts[name] + 1
	}
	printCountResults(counts)
}

func loopCounter() {
	for range time.Tick(time.Millisecond * 600) {
		tickCounter()
	}
}

func tickCounter() {
	lock.Lock()
	defer lock.Unlock()
	if !isCountingHand {
		return
	}
	if !refreshed {
		refreshed = true
		ext.Send(out.GETSTRIP, "update")
		return
	}

	isDone := len(inventoryMgr.Items()) == 0
	for _, item := range inventoryMgr.Items() {
		if isCounted[item.ItemId] {
			isDone = true
			continue
		}
		if item.Type == inventory.Floor {
			name := fmt.Sprintf("%v (%v)", getFurniName(item.Class), item.Class)
			counts[name] = counts[name] + 1
		} else {
			name, key := getPosterName(item.Class, item.Props)
			name = fmt.Sprintf("%v (%v)", name, key)
			counts[name] = counts[name] + 1
		}
		isCounted[item.ItemId] = true
	}
	if isDone {
		printCountResults(counts)
		isCountingHand = false
	} else {
		ext.Send(out.GETSTRIP, []byte("next"))
	}
}

func printCountResults(counts map[string]int) {
	type kv struct {
		k string
		v int
	}
	var entries []kv
	for k, v := range counts {
		entries = append(entries, kv{k, v})
	}

	// Sort the slice by count in descending order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].v > entries[j].v
	})

	// Take top 30 (more doesn't fit in alert box)
	more := 0
	if len(entries) > 30 {
		more = len(entries) - 29
		entries = entries[:29]
	}
	var alert []string
	for _, entry := range entries {
		alert = append(alert, fmt.Sprintf("%vx %v", entry.v, entry.k))
	}
	if more > 0 {
		alert = append(alert, fmt.Sprintf("... and %v more", more))
	}
	if len(alert) > 0 {
		ext.Send(in.SYSTEM_BROADCAST, []byte(strings.Join(alert, "\r")))
	}
}
