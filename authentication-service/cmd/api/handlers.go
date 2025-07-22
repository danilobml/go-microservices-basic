package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/danilobml/authentication-service/cmd/api/data"

	goweb "github.com/danilobml/go-webtoolkit"
)

var tools goweb.Tools

type User = data.User

func (app *Config) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := app.Models.GetAll()
	if err != nil {
		log.Printf("error getting users: %s", err)
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: "success",
		Data:    users,
	}

	tools.WriteJSON(w, http.StatusOK, payload)
}

func (app *Config) authenticate(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := tools.ReadJSON(w, r, &requestBody)
	if err != nil {
		log.Println(err)
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	user, err := app.Models.GetByEmail(requestBody.Email)
	if err != nil {
		log.Println(err)
		tools.ErrorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}

	valid, err := user.PasswordMatches(requestBody.Password)
	if err != nil || !valid {
		log.Println(err)
		tools.ErrorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}

	err = app.logRequest("auth", fmt.Sprintf("Logged in user: %s", user.Email))
	if err != nil {
		log.Println(err)
		tools.ErrorJSON(w, errors.New("log service error"), http.StatusUnauthorized)
		return
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged user %s in", user.Email),
		Data:    user,
	}

	tools.WriteJSON(w, http.StatusOK, payload)
}

func (app *Config) logRequest(name, data string) error {
	var entry struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}

	entry.Name = name
	entry.Data = data

	jsonData, _ := json.Marshal(entry)

	loggerUrl := "http://logger-service/log"

	request, err := http.NewRequest("POST", loggerUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	client := &http.Client{}
	_, err = client.Do(request)
	if err != nil {
		return err
	}

	return nil
}
