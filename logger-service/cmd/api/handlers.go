package main

import (
	"log"
	"net/http"

	goweb "github.com/danilobml/go-webtoolkit"
	"github.com/danilobml/logger-service/data"
)

var tools goweb.Tools

type JSONPayload struct {
	Name string
	Data string
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {
	var requestPayload JSONPayload

	tools.ReadJSON(w, r, &requestPayload)

	event := data.LogEntry{
		Name: requestPayload.Name,
		Data: requestPayload.Data,
	}

	err := app.Models.LogEntry.Insert(event)
	if err != nil {
		log.Println(err)
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: "log created",
	}
	
	tools.WriteJSON(w, http.StatusCreated, payload)
}

func (app *Config) GetAllEntries(w http.ResponseWriter, r *http.Request) {
	logEntry := data.LogEntry{}
	entries, err := logEntry.All()
	if err != nil {
		log.Println(err)
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Data:    entries,
	}

	tools.WriteJSON(w, http.StatusOK, payload)
}
