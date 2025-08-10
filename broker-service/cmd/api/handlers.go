package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/rpc"
	"time"

	"github.com/danilobml/broker/cmd/api/event"
	"github.com/danilobml/broker/logs"
	goweb "github.com/danilobml/go-webtoolkit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var tools goweb.Tools

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
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
		// // REST:
		// app.logItem(w, requestPayload.Log)
		// // RabbitMQ
		// app.logEventViaRabbit(w, requestPayload.Log)
		// RPC:
		app.logEventViaRpc(w, requestPayload.Log)
	case "mail":
		app.sendMail(w, requestPayload.Mail)
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

func (app *Config) sendMail(w http.ResponseWriter, mail MailPayload) {
	jsonData, _ := json.Marshal(mail)

	request, err := http.NewRequest("POST", "http://mail-service/send", bytes.NewBuffer(jsonData))
	if err != nil {
		tools.ErrorJSON(w, errors.New("failed creating request to mail-service"), http.StatusInternalServerError)
		return
	}
	request.Header.Set("Content-Type", "application-json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		tools.ErrorJSON(w, errors.New("response from mail-service failed"), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	var jsonFromService goweb.JsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		tools.ErrorJSON(w, errors.New("failed parsing response from mail-service"), http.StatusInternalServerError)
		return
	}

	if jsonFromService.Error {
		tools.ErrorJSON(w, errors.New("failed sending mail: "+jsonFromService.Message), http.StatusUnauthorized)
		return
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: "mail sent succesfully",
	}

	tools.WriteJSON(w, http.StatusCreated, payload)
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: "logged entry succesfully via RabbitMq",
	}

	tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}

	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	jsonPayload, _ := json.Marshal(&payload)

	err = emitter.Push(string(jsonPayload), "log.INFO")
	if err != nil {
		return err
	}

	return nil
}

type RPCPayload struct {
	Name string
	Data string
}

func (app *Config) logEventViaRpc(w http.ResponseWriter, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
	}

	rpcPayload := RPCPayload{
		Name: l.Name,
		Data: l.Data,
	}

	var result string
	err = client.Call("RPCServer.LogInfo", rpcPayload, &result)
	if err != nil {
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: result,
	}

	tools.WriteJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logEventViaGrpc(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	err := tools.ReadJSON(w, r, &requestPayload)
	if err != nil {
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	logEntry := logs.Log{
		Name: requestPayload.Log.Name,
		Data: requestPayload.Log.Data,
	}

	log.Printf("Broker sending gRPC log: name=%q, data=%q", logEntry.Name, logEntry.Data)

	logRequest := logs.LogRequest{LogEntry: &logEntry}

	conn, err := grpc.Dial("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	c := logs.NewLoggerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result, err := c.WriteLog(ctx, &logRequest)
	if err != nil {
		tools.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	payload := goweb.JsonResponse{
		Error:   false,
		Message: result.GetResult(),
	}

	tools.WriteJSON(w, http.StatusAccepted, payload)
}
