package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samber/lo"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/oidc/userinfo", userinfo)
	bindAddr := ":" + getenv("PORT", "8080")
	err := http.ListenAndServe(bindAddr, nil)
	if err != nil {
		panic(err)
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func userinfo(rw http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://graph.microsoft.com/oidc/userinfo", nil)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		http.Error(rw, fmt.Sprintf("userinfo returned status code: %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}
	uinfo := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&uinfo); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	gs, err := groups(auth, "https://graph.microsoft.com/v1.0/me/transitiveMemberOf?$top=999&$select=displayName,id")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	uinfo["groups"] = gs

	output, err := json.Marshal(uinfo)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(output)
}

func groups(auth string, url string) ([]string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("userinfo returned status code: %d", resp.StatusCode))
	}
	graph := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&graph); err != nil {
		return nil, err
	}
	values, ok := graph["value"].([]interface{})
	if !ok {
		return nil, err
	}
	var gs []string
	gs = lo.Map(values, func(value interface{}, _ int) string {
		m, _ := value.(map[string]interface{})
		displayName, _ := m["displayName"].(string)
		return displayName
	})
	if graph["@odata.nextLink"] != nil {
		nextLink, _ := graph["@odata.nextLink"].(string)
		ngs, err := groups(auth, nextLink)
		if err != nil {
			return nil, err
		}
		gs = append(gs, ngs...)
	}
	return gs, nil
}
