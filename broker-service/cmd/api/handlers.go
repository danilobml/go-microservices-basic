package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	goweb "github.com/danilobml/go-webtoolkit"
)

var tools goweb.Tools

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := goweb.JsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	tools.WriteJSON(w, http.StatusOK, payload)
}

func (app *Config) handleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := tools.ReadJSON(w, r, &requestPayload)
	if err != nil {
		log.Println(err)
		tools.ErrorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
	case "log":
		app.logItem(w, requestPayload.Log)
	default:
		tools.ErrorJSON(w, errors.New("invalid action"), http.StatusBadRequest)
	}
}

func (app *Config) authenticate(w http.ResponseWriter, auth AuthPayload) {
	jsonData, _ := json.Marshal(auth)

	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		tools.ErrorJSON(w, errors.New("failed creating request to auth-service"), http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		tools.ErrorJSON(w, errors.New("response from auth-service failed"), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		tools.ErrorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}
	if response.StatusCode != http.StatusOK {
		tools.ErrorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}

	var jsonFromService goweb.JsonResponse

	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		tools.ErrorJSON(w, errors.New("failed parsing response from auth service"), http.StatusUnauthorized)
		return
	}

	if jsonFromService.Error {
		tools.ErrorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: "logged!",
		Data:    jsonFromService.Data,
	}

	tools.WriteJSON(w, http.StatusOK, payload)
}

func (app *Config) logItem(w http.ResponseWriter, log LogPayload) {
	jsonData, _ := json.Marshal(log)

	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		tools.ErrorJSON(w, errors.New("failed creating request to log-service"), http.StatusInternalServerError)
		return
	}
	request.Header.Set("Content-Type", "application-json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		tools.ErrorJSON(w, errors.New("response from log-service failed"), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var jsonFromService goweb.JsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		tools.ErrorJSON(w, errors.New("failed parsing response from log service"), http.StatusInternalServerError)
		return
	}

	if jsonFromService.Error {
		tools.ErrorJSON(w, errors.New("failed logging: "+jsonFromService.Message), http.StatusUnauthorized)
		return
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: "logged entry succesfully",
	}

	tools.WriteJSON(w, http.StatusCreated, payload)
}
