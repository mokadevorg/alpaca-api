package rest

import (
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"net/http"
	"log"
	"gopkg.in/mgo.v2/bson"
	"encoding/json"
	"fmt"
)

type MongoDoc interface {
	DocId() bson.ObjectId
	IsValid() bool
	BuildForInsertion()
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func makePath(prefix, endpoint string, more ...string) string {
	path := "/" + prefix + "/" + endpoint
	for _, add := range more {
		path += "/" + add
	}
	return path
}

func contentTypeJson(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}


type RecordEndpointMaker struct {
	Prefix string
	Router *mux.Router
	DB     *mgo.Database
}


func (rem *RecordEndpointMaker) MakeGetEndpoint(record string, target interface{}) {
	path := makePath(rem.Prefix, record, "{id:[0-9a-f]{24}}")

	rem.Router.NewRoute().Methods("GET").Path(path).HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("GET %s", r.RequestURI)
			// Set Content-Type header
			contentTypeJson(w)

			// Get URI parameters (id)
			vars := mux.Vars(r)
			// Get collection from bound DB with name=record
			resources := rem.DB.C(record);

			if err := resources.FindId(bson.ObjectIdHex(vars["id"])).One(target); err != nil {
				if err == mgo.ErrNotFound {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Printf("Error %s: %s", r.RequestURI, err.Error())
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(target)
		})
}

func (rem *RecordEndpointMaker) MakeListEndpoint(record string, targetList interface{}) {
	path := makePath(rem.Prefix, record)

	rem.Router.NewRoute().Methods("GET").Path(path).HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			contentTypeJson(w)

			log.Printf("GET %s", r.RequestURI)

			resources := rem.DB.C(record)

			if err := resources.Find(nil).All(targetList); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Printf("Error %s: %s", r.RequestURI, err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(targetList)
		})
}

func (rem *RecordEndpointMaker) MakeCreateEndpoint(record string, target MongoDoc) {
	path := makePath(rem.Prefix, record)

	rem.Router.NewRoute().Methods("POST").Path(path).HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			contentTypeJson(w)

			log.Printf("POST %s", r.RequestURI)

			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Printf("Error %s: %s", r.RequestURI, err.Error())
				return
			}

			if target.IsValid() {
				target.BuildForInsertion()

				resources := rem.DB.C(record)
				if err := resources.Insert(target); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
					log.Printf("Error %s: %s", r.RequestURI, err.Error())
					return
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(target)
			} else {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(&ErrorResponse{
					"Required fields missing",
				})
			}

		})
}

func (rem *RecordEndpointMaker) MakeRemoveEndpoint(record string) {
	path := makePath(rem.Prefix, record, "{id:[0-9a-f]{24}}")

	rem.Router.NewRoute().Methods("DELETE").Path(path).HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			contentTypeJson(w)

			log.Printf("DELETE %s", r.RequestURI)

			vars := mux.Vars(r)
			resources := rem.DB.C(record)

			if err := resources.RemoveId(bson.ObjectIdHex(vars["id"])); err != nil {
				if err == mgo.ErrNotFound {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Printf("Error %s: %s", r.RequestURI, err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
		})
}

func (rem *RecordEndpointMaker) MakeUpdateEndpoint(record string, target MongoDoc) {
	path := makePath(rem.Prefix, record, "{id:[0-9a-f]{24}}")

	rem.Router.NewRoute().Methods("PUT").Path(path).HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			contentTypeJson(w)

			log.Printf("PUT %s", r.RequestURI)

			vars := mux.Vars(r)
			resources := rem.DB.C(record)

			if err := resources.FindId(bson.ObjectIdHex(vars["id"])).One(target); err != nil {
				if err == mgo.ErrNotFound {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Printf("Error %s: %s", r.RequestURI, err.Error())
				return
			}

			json.NewDecoder(r.Body).Decode(target)
			defer r.Body.Close()

			if err := resources.UpdateId(target.DocId(), target); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				errMsg := fmt.Sprintf("Unexpected error %s: %s",
					r.RequestURI, err.Error())
				json.NewEncoder(w).Encode(&ErrorResponse{ errMsg })
				log.Printf("%s", errMsg)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(target)
		})
}

func (rem *RecordEndpointMaker) MakeSearchEndpoint(record string, targetList interface{}) {
	path := makePath(rem.Prefix, record, "_search")

	rem.Router.NewRoute().Methods("POST").Path(path).HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			contentTypeJson(w)

			log.Printf("POST %s", r.RequestURI)

			searchRequest := make(map[string]string)
			searchQuery := make(bson.M)

			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Printf("Error %s: %s", r.RequestURI, err.Error())
				return
			}

			for k, v := range searchRequest {
				searchQuery[k] = bson.M{"$regex": "^"+v}
			}

			resources := rem.DB.C(record)
			if err := resources.Find(&searchQuery).All(targetList); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Printf("Error %s: %s", r.RequestURI, err.Error())
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(targetList)
		})
}