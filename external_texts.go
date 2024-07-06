package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
)

var externalTexts = make(map[string]string)

func loadExternalTexts(gameHost string) {
	url := "https://origins-gamedata.habbo.com/external_texts/1"
	switch gameHost {
	case "game-obr.habbo.com":
		url = "https://origins-gamedata.habbo.com.br/external_texts/1"
	case "game-oes.habbo.com":
		url = "https://origins-gamedata.habbo.es/external_texts/1"
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching external texts:", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			externalTexts[key] = value
		}
	}
}

func getFurniName(class string) string {
	name, ok := externalTexts[fmt.Sprintf("furni_%v_name", class)]
	if ok {
		return name
	}
	return class
}

func getPosterName(class string, props string) (string, string) {
	name, ok := externalTexts[fmt.Sprintf("wallitem_%v_name", class)]
	if ok {
		return name, class
	}
	posterId := fmt.Sprintf("%v_%v", class, props)
	name, ok = externalTexts[fmt.Sprintf("%v_name", posterId)]
	if ok {
		return name, posterId
	}
	return posterId, posterId
}
