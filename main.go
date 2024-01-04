package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type AttributeOrTraits struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type NewData struct {
	Event           interface{}                  `json:"event"`
	EventType       interface{}                  `json:"event_type"`
	AppID           interface{}                  `json:"app_id"`
	UserID          interface{}                  `json:"user_id"`
	MessageID       interface{}                  `json:"message_id"`
	PageTitle       interface{}                  `json:"page_title"`
	PageURL         interface{}                  `json:"page_url"`
	BrowserLanguage interface{}                  `json:"browser_language"`
	ScreenSize      interface{}                  `json:"screen_size"`
	Attributes      map[string]AttributeOrTraits `json:"attributes"`
	Traits          map[string]AttributeOrTraits `json:"traits"`
}

var requestChannel chan NewData

func main() {
	requestChannel = make(chan NewData)

	go worker()

	http.HandleFunc("/data-convertion", data_convertion)

	fmt.Println("Server listening on :8080...")
	http.ListenAndServe(":8080", nil)
}

func worker() {
	for {
		newData := <-requestChannel

		convertedJSON, err := json.Marshal(newData)
		if err != nil {
			fmt.Println("Error converting data to webhook format:", err)
			continue
		}

		webhookURL := "https://webhook.site/466fd427-b858-46bb-98ef-03a87cc8fcdb"
		resp, err := http.Post(webhookURL, "application/json", strings.NewReader(string(convertedJSON)))
		if err != nil {
			fmt.Println("Error sending webhook:", err)
			continue
		}
		defer resp.Body.Close()

		fmt.Println("Webhook response:", resp.Status)
	}
}

func extractIndex(key, prefix string) (int, bool) {
	if indexStr := strings.TrimPrefix(key, prefix); len(indexStr) < len(key) {
		index, err := strconv.Atoi(indexStr)
		return index, err == nil
	}
	return 0, false
}

func data_convertion(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		var requestData map[string]interface{}
		if err := json.Unmarshal(body, &requestData); err != nil {
			http.Error(w, "Error parsing JSON data", http.StatusBadRequest)
			return
		}

		newData := NewData{
			Event:           requestData["ev"],
			EventType:       requestData["et"],
			AppID:           requestData["id"],
			UserID:          requestData["uid"],
			MessageID:       requestData["mid"],
			PageTitle:       requestData["t"],
			PageURL:         requestData["p"],
			BrowserLanguage: requestData["l"],
			ScreenSize:      requestData["sc"],
			Attributes:      make(map[string]AttributeOrTraits),
			Traits:          make(map[string]AttributeOrTraits),
		}

		for key, value := range requestData {
			var prefix string
			var index int

			if atrkIndex, atrkOk := extractIndex(key, "atrk"); atrkOk {
				prefix, index = "atrv", atrkIndex
			} else if uatrkIndex, uatrkOk := extractIndex(key, "uatrk"); uatrkOk {
				prefix, index = "uatrv", uatrkIndex
			} else {
				continue
			}

			atrvKey := fmt.Sprintf("%s%d", prefix, index)
			atrtKey := fmt.Sprintf("%st%d", prefix[:len(prefix)-1], index)

			if atrv, atrvOk := requestData[atrvKey].(string); atrvOk {
				newKey := value.(string)
				entry := AttributeOrTraits{
					Value: atrv,
					Type:  requestData[atrtKey].(string),
				}

				if prefix == "atrv" {
					newData.Attributes[newKey] = entry
				} else if prefix == "uatrv" {
					newData.Traits[newKey] = entry
				}
			}
		}

		requestChannel <- newData

		newDataJSON, err := json.Marshal(newData)
		if err != nil {
			http.Error(w, "Error encoding JSON data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(newDataJSON)
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
