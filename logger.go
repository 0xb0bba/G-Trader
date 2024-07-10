package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"xabbo.b7c.io/goearth/shockwave/trade"
)

func formatTrade(trade trade.Offers) bytes.Buffer {
	if trade.Trader().Name != profileMgr.Name {
		trade[0], trade[1] = trade[1], trade[0]
	}
	var buffer bytes.Buffer
	buffer.WriteString("--------------------\n")
	for _, offer := range trade {
		buffer.WriteString(fmt.Sprintf("%v traded:\n", offer.Name))
		counts := make(map[string]int)
		for _, item := range offer.Items {
			name := getFullName(item)
			counts[name] = counts[name] + 1
		}
		for name, count := range counts {
			buffer.WriteString(fmt.Sprintf("%vx %v\n", count, name))
		}
		if len(counts) == 0 {
			buffer.WriteString("Nothing\n")
		}
	}
	return buffer
}

func logTrade(content bytes.Buffer) {
	webhookURL := config["WEBHOOK_URL"]
	if webhookURL == "" {
		return
	}
	field := config["WEBHOOK_FIELD"]
	if field == "" {
		field = "content"
	}

	payloadBytes, _ := json.Marshal(map[string]string{field: content.String()})
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error building webhook request", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error posting to webhook", err)
		return
	}
}
