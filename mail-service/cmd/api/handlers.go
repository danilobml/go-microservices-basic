package main

import (
	"errors"
	"net/http"

	goweb "github.com/danilobml/go-webtoolkit"
)

var tools goweb.Tools

func (app *Config) SendMail(w http.ResponseWriter, r *http.Request) {
	type mailMessage struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}

	var requestPayload mailMessage

	err := tools.ReadJSON(w, r, &requestPayload)
	if err != nil {
		tools.ErrorJSON(w, errors.New("failed sending mail:"+err.Error()), http.StatusInternalServerError)
		return
	}
	
	message := Message{
		From: requestPayload.From,
		To: requestPayload.To,
		Subject: requestPayload.Subject,
		Data: requestPayload.Message,
	}
	

	err = app.Mailer.SendSMTPMessage(message)
	if err != nil {
		tools.ErrorJSON(w, errors.New("failed sending mail:"+err.Error()), http.StatusInternalServerError)
		return
	}

	payload := goweb.JsonResponse{
		Error: false,
		Message: "email sent to: " + requestPayload.To,
	}

	tools.WriteJSON(w, http.StatusAccepted, payload)
}
