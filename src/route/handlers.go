package route

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/gorilla/mux"
	"github.com/kafkaesque-io/pubsub-function/src/db"
	"github.com/kafkaesque-io/pubsub-function/src/lambda"
	"github.com/kafkaesque-io/pubsub-function/src/model"
	"github.com/kafkaesque-io/pubsub-function/src/pulsardriver"
	"github.com/kafkaesque-io/pubsub-function/src/util"

	log "github.com/sirupsen/logrus"
)

var singleDb db.Db

const subDelimiter = "-"

// Init initializes database
func Init() {
	singleDb = db.NewDbWithPanic(util.GetConfig().PbDbType)
}

// TokenServerResponse is the json object for token server response
type TokenServerResponse struct {
	Subject string `json:"subject"`
	Token   string `json:"token"`
}

// TokenSubjectHandler issues new token
func TokenSubjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subject, ok := vars["sub"]
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if util.StrContains(util.SuperRoles, util.AssignString(r.Header.Get("injectedSubs"), "BOGUSROLE")) {
		tokenString, err := util.JWTAuth.GenerateToken(subject)
		if err != nil {
			util.ResponseErrorJSON(errors.New("failed to generate token"), w, http.StatusInternalServerError)
		} else {
			respJSON, err := json.Marshal(&TokenServerResponse{
				Subject: subject,
				Token:   tokenString,
			})
			if err != nil {
				util.ResponseErrorJSON(errors.New("failed to marshal token response json object"), w, http.StatusInternalServerError)
				return
			}
			w.Write(respJSON)
			w.WriteHeader(http.StatusOK)
		}
		return
	}
	util.ResponseErrorJSON(errors.New("incorrect subject"), w, http.StatusUnauthorized)
	return
}

// StatusPage replies with basic status code
func StatusPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}

// ReceiveHandler - the message receiver handler
func ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		util.ResponseErrorJSON(err, w, http.StatusInternalServerError)
		return
	}
	token, topic, pulsarURL, err := util.ReceiverHeader(util.AllowedPulsarURLs, &r.Header)
	if err != nil {
		util.ResponseErrorJSON(err, w, http.StatusUnauthorized)
		return
	}

	topicFN, err2 := GetTopicFnFromRoute(mux.Vars(r))
	if topic == "" && err2 != nil {
		// only read topic from routes
		util.ResponseErrorJSON(err2, w, http.StatusUnprocessableEntity)
		return
	}
	topicFN = util.AssignString(topic, topicFN) // header topicFn overwrites topic specified in the routes
	log.Infof("topicFN %s pulsarURL %s", topicFN, pulsarURL)

	pulsarAsync := r.URL.Query().Get("mode") == "async"
	err = pulsardriver.SendToPulsar(pulsarURL, token, topicFN, b, pulsarAsync)
	if err != nil {
		util.ResponseErrorJSON(err, w, http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// GetTopicKey gets the topic key from the request body or url sub route
func GetTopicKey(r *http.Request) (string, error) {
	var err error
	vars := mux.Vars(r)
	topicKey, ok := vars["topicKey"]
	if !ok {
		var topic model.TopicKey
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()

		err := decoder.Decode(&topic)
		switch {
		case err == io.EOF:
			return "", errors.New("missing topic key or topic names in body")
		case err != nil:
			return "", err
		}
		topicKey, err = model.GetKeyFromNames(topic.TopicFullName, topic.PulsarURL)
		if err != nil {
			return "", err
		}
	}
	return topicKey, err
}

// VerifySubjectBasedOnTopic verifies the subject can meet the requirement.
func VerifySubjectBasedOnTopic(topicFN, tokenSub string, evalTenant func(tenant, subjects string) bool) bool {
	parts := strings.Split(topicFN, "/")
	if len(parts) < 4 {
		return false
	}
	tenant := parts[2]
	if len(tenant) < 1 {
		log.Infof(" auth verify tenant %s token sub %s", tenant, tokenSub)
		return false
	}
	return VerifySubject(tenant, tokenSub, evalTenant)
}

// VerifySubject verifies the subject can meet the requirement.
// Subject verification requires role or tenant name in the jwt subject
func VerifySubject(requiredSubject, tokenSubjects string, evalTenant func(tenant, subjects string) bool) bool {
	for _, v := range strings.Split(tokenSubjects, ",") {
		if util.StrContains(util.SuperRoles, v) {
			return true
		}
		if requiredSubject == v {
			return true
		}
		if evalTenant(requiredSubject, v) {
			return true
		}
	}

	return false
}

// ExtractEvalTenant is a customized function to evaluate subject against tenant
func ExtractEvalTenant(requiredSubject, tokenSub string) bool {
	// expect - in subject unless it is superuser
	var sub string
	parts := strings.Split(tokenSub, subDelimiter)
	if len(parts) < 2 {
		sub = parts[0]
	}

	validLength := len(parts) - 1
	sub = strings.Join(parts[:validLength], subDelimiter)
	if sub != "" && requiredSubject == sub {
		return true
	}
	return false
}

// GetTopicFnFromRoute builds a valida topic fullname from the http route
func GetTopicFnFromRoute(vars map[string]string) (string, error) {
	tenant, ok := vars["tenant"]
	namespace, ok2 := vars["namespace"]
	topic, ok3 := vars["topic"]
	persistent, ok4 := vars["persistent"]
	if !(ok && ok2 && ok3 && ok4) {
		return "", fmt.Errorf("missing topic parts")
	}
	topicFn, err := util.BuildTopicFn(persistent, tenant, namespace, topic)
	if err != nil {
		return "", err
	}
	return topicFn, nil
}

// ConsumerParams returns a configuration parameters for Pulsar consumer
func ConsumerParams(params url.Values) (subName string, subInitPos pulsar.SubscriptionInitialPosition, subType pulsar.SubscriptionType, err error) {
	subType, err = model.GetSubscriptionType(util.QueryParamString(params, "SubscriptionType", "exclusive"))
	if err != nil {
		return "", -1, -1, err
	}
	subInitPos, err = model.GetInitialPosition(util.QueryParamString(params, "SubscriptionInitialPosition", "latest"))
	if err != nil {
		return "", -1, -1, err
	}

	subName = util.QueryParamString(params, "SubscriptionName", "")
	if len(subName) == 0 {
		name, err := util.NewUUID()
		if err != nil {
			return "", -1, -1, fmt.Errorf("failed to generate uuid error %v", err)
		}
		return model.NonResumable + name, subInitPos, subType, nil
	} else if len(subName) < 5 {
		return "", -1, -1, fmt.Errorf("subscription name must be more than 4 characters")
	}
	return subName, subInitPos, subType, nil
}

// ConsumerConfigFromHTTPParts returns configuration parameters required to generate Pulsar Client and Consumer
func ConsumerConfigFromHTTPParts(allowedClusters []string, h *http.Header, vars map[string]string, params url.Values) (token, topicFN, pulsarURL, subName string, subInitPos pulsar.SubscriptionInitialPosition, subType pulsar.SubscriptionType, err error) {
	token, _, pulsarURL, err = util.ReceiverHeader(allowedClusters, h)
	if err != nil {
		return "", "", "", "", -1, -1, err
	}

	topicFN, err = GetTopicFnFromRoute(vars)
	if err != nil {
		return "", "", "", "", -1, -1, err
	}

	subName, subInitPos, subType, err = ConsumerParams(params)
	if err != nil {
		return "", "", "", "", -1, -1, err
	}

	return token, topicFN, pulsarURL, subName, subInitPos, subType, nil
}

// Pulsar function CRUD and trigger

// GetFunctionHandler gets a function
func GetFunctionHandler(w http.ResponseWriter, r *http.Request) {
}

// UpdateFunctionHandler creates or updates a function
func UpdateFunctionHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 10); err != nil {
		util.ResponseErrorJSON(err, w, http.StatusUnprocessableEntity)
		return
	}
	tenant, functionName, err := tenantFunctionName(mux.Vars(r))
	if tenant == "" || functionName == "" || err != nil {
		util.ResponseErrorJSON(err, w, http.StatusUnprocessableEntity)
		return
	}
	tokenStr, _, pulsarURL, err := util.ReceiverHeader(util.AllowedPulsarURLs, &r.Header)
	if err != nil {
		util.ResponseErrorJSON(err, w, http.StatusUnauthorized)
		return
	}

	now := time.Now()
	doc := model.FunctionConfig{
		Name:           functionName,
		Tenant:         tenant,
		ID:             tenant + functionName,
		LanguagePack:   util.AssignString(r.FormValue("language-pack"), "javascript"),
		Parallelism:    util.GetEnvInt(r.FormValue("parallelism"), 1),
		TriggerType:    util.AssignString(r.FormValue("trigger-type"), "pulsar-topic"),
		FunctionStatus: model.StringToStatus(r.FormValue("function-status")),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	file, fileReader, err := r.FormFile("source")
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		util.ResponseErrorJSON(err, w, http.StatusUnprocessableEntity)
		return
	}

	log.Infof("MIME Header: %+v\nUploaded File: %+v\nFile Size: %+v\n, languagePack %s, parallel instance %d, triggerType %s",
		fileReader.Header, fileReader.Filename, fileReader.Size, doc.LanguagePack, doc.Parallelism, doc.TriggerType)
	if doc.TriggerType == lambda.PulsarTrigger {
		doc.InputTopic = model.FunctionTopic{
			PulsarURL:        pulsarURL,
			TopicFullName:    r.FormValue("input-topic"),
			Token:            tokenStr,
			Tenant:           tenant,
			Subscription:     r.FormValue("subscription-name"),
			SubscriptionType: r.FormValue("subscription-type"),
			InitialPosition:  r.FormValue("subscription-initial-position"),
			KeySharedPolicy:  r.FormValue("key-shared-policy"),
		}
	}
	if r.FormValue("output-topic") != "" {
		doc.OutputTopic = model.FunctionTopic{
			PulsarURL:     pulsarURL,
			Token:         tokenStr,
			TopicFullName: r.FormValue("output-topic"),
			Tenant:        tenant,
		}
	}

	// read all of the contents of our uploaded file into a byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		util.ResponseErrorJSON(err, w, http.StatusInternalServerError)
		return
	}
	doc.FunctionFilePath = lambda.GetSourceFilePath(doc.Tenant) + "/" + functionName + ".js"
	// write this byte array to our temporary file
	if err = ioutil.WriteFile(doc.FunctionFilePath, fileBytes, 0644); err != nil {
		util.ResponseErrorJSON(err, w, http.StatusInternalServerError)
		return
	}

	functionURLs := []string{}
	for i := 0; i < doc.Parallelism; i++ {
		url, err := lambda.StartNodeInstance(doc)
		if err != nil {
			log.Errorf("start function node failure %v", err)
			util.ResponseErrorJSON(err, w, http.StatusInternalServerError)
			return
		}
		// return that we have successfully uploaded our file!
		// fmt.Fprintf(w, "Successfully Uploaded File server started with url %s\n", url)
		functionURLs = append(functionURLs, url)
	}
	doc.WebhookURLs = functionURLs

	log.Infof("function metadata %v", doc)

	id, err := singleDb.Update(&doc)
	if err != nil {
		util.ResponseErrorJSON(err, w, http.StatusConflict)
		return
	}
	if len(id) > 1 {
		savedDoc, err := singleDb.GetByKey(id)
		if err != nil {
			util.ResponseErrorJSON(err, w, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		savedDoc.InputTopic.Token = "***"
		savedDoc.OutputTopic.Token = "***"
		savedDoc.LogTopic.Token = "***"
		resJSON, err := json.Marshal(savedDoc)
		if err != nil {
			util.ResponseErrorJSON(err, w, http.StatusInternalServerError)
			return
		}
		w.Write(resJSON)
		return
	}
	util.ResponseErrorJSON(fmt.Errorf("failed to update"), w, http.StatusInternalServerError)
}

// DeleteFunctionHandler deletes a function
func DeleteFunctionHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// TriggerFunctionHandler deletes a function
func TriggerFunctionHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func tenantFunctionName(vars map[string]string) (string, string, error) {
	tenant, ok := vars["tenant"]
	name, ok2 := vars["function"]
	if !(ok && ok2) {
		return "", "", fmt.Errorf("missing tenant function names")
	}
	return tenant, name, nil
}
