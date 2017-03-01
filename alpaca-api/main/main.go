package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"encoding/json"
	"log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/mokadevorg/alpaca-api/record"
)

type ServerInfo struct {
	Version string `json:"version"`
	Name    string `json:"name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Project struct {
	Id	    bson.ObjectId `bson:"_id"         json:"id,omitempty"`
	Name	    string 	  `bson:"name"        json:"name"`
	Category    string 	  `bson:"category"    json:"category"`
	Description string 	  `bson:"description" json:"description"`
}

func main() {
	record.InitAlpacaRecord()

	router := mux.NewRouter()

	router.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		ContentTypeJson(w)

		err := json.NewEncoder(w).Encode(&ServerInfo{"0.1", "Project Alpaca"})
		if err != nil {
			log.Fatal("Could not encode server info")
		}
	})

	router.NewRoute().Methods("GET").Path("/api/projects/{id:[0-9a-f]+}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer manageObjectIdPanic(r.RequestURI, w)

			// Set content-type
			ContentTypeJson(w)

			log.Println("GET", r.RequestURI)

			// Get uri vars
			vars := mux.Vars(r)
			// Bind collection
			projects := record.AlpacaRecordCollection("projects")
			// Get by id
			var match Project
			err := projects.FindId(bson.ObjectIdHex(vars["id"])).One(&match) // This panics if Id is not valid
			// Handle errors
			if err == mgo.ErrNotFound {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
			} else if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Fatal(err.Error())
			} else {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(&match)
			}
		})

	router.NewRoute().Methods("GET").Path("/api/projects").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ContentTypeJson(w)

			log.Println("GET", r.RequestURI)

			projects := record.AlpacaRecordCollection("projects")
			projectList := make([]Project, 0, 10)

			if err := projects.Find(nil).All(&projectList); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Println("Error listing /api/projects: ", err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(projectList)
		})

	router.NewRoute().Methods("POST").Path("/api/projects").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ContentTypeJson(w)

			log.Println("POST", r.RequestURI)

			var project Project
			if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				log.Println("Error decoding POST /api/projects: ", err.Error())
				return
			}

			if project.IsValid() {
				project.BuildForInsertion() // Set id

				projects := record.AlpacaRecordCollection("projects")
				if err := projects.Insert(&project); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
					log.Println("Error inserting /api/projects/create:", err.Error())
					return
				}

				var insertedProject Project
				projects.FindId(project.Id).One(&insertedProject)

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(&insertedProject)
			} else {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(&ErrorResponse{ "Name, Description and Category are required" })
			}
		})


	router.NewRoute().Methods("OPTIONS").Path("/api/projects").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Println("OPTIONS", r.RequestURI)
			AllowExternal(w)
		});

	router.NewRoute().Methods("OPTIONS").Path("/api/projects/{id:[0-9a-f]+}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Println("OPTIONS", r.RequestURI)
			AllowExternal(w)
		});

	router.NewRoute().Methods("DELETE").Path("/api/projects/{id:[0-9a-f]+}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer manageObjectIdPanic(r.RequestURI, w)

			ContentTypeJson(w)

			log.Println("DELETE", r.RequestURI)

			vars := mux.Vars(r)
			projects := record.AlpacaRecordCollection("projects")

			if err := projects.RemoveId(bson.ObjectIdHex(vars["id"])); err != nil {
				if err == mgo.ErrNotFound {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					log.Println("Unknown error /api/projects/create/", vars["id"], ":", err.Error())
				}
				json.NewEncoder(w).Encode(&ErrorResponse{ err.Error() })
				return
			}
			w.WriteHeader(http.StatusOK)
		})

	http.Handle("/", router)
	http.ListenAndServe("0.0.0.0:8000", nil)
}



func manageObjectIdPanic(endpoint string, w http.ResponseWriter) {
	if err := recover(); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		switch t := err.(type) {
		case string:
			json.NewEncoder(w).Encode(&ErrorResponse{ t })
			log.Println("Error", endpoint, ":", t)
		case error:
			json.NewEncoder(w).Encode(&ErrorResponse{ t.Error() })
			log.Println("Error", endpoint, ":", t.Error())
		default:
			json.NewEncoder(w).Encode(&ErrorResponse{ "Unknown error" })
			log.Println("Error", endpoint, ": Unknown error")
		}
	}
}

func AllowExternal(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*");
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE");
	w.Header().Set("Access-Control-Max-Age", "3600");
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type");
}

func ContentTypeJson(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (p *Project) IsValid() bool {
	if p.Name == "" || p.Description == "" || p.Category == "" {
		return false
	}
	return true
}

func (p *Project) BuildForInsertion() {
	p.Id = bson.NewObjectId()
}