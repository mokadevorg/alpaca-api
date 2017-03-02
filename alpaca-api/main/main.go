package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"encoding/json"
	"log"
	"gopkg.in/mgo.v2/bson"
	"github.com/mokadevorg/alpaca-api/record"
	"errors"
	"github.com/mokadevorg/alpaca-api/rest"
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

type Category struct {
	Id   bson.ObjectId `bson:"_id"  json:"id"`
	Name string 	   `bson:"name" json:"name"`
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

	rem := rest.RecordEndpointMaker{"api", router, record.AlpacaRecord().DB("alpaca")}
	rem.MakeGetEndpoint("projects", &Project{})
	rem.MakeListEndpoint("projects", new([]Project))
	rem.MakeCreateEndpoint("projects", &Project{})
	rem.MakeRemoveEndpoint("projects")
	rem.MakeUpdateEndpoint("projects", &Project{})

	rem.MakeGetEndpoint("categories", &Category{})
	rem.MakeListEndpoint("categories", new([]Category))
	rem.MakeCreateEndpoint("categories", &Category{})
	rem.MakeRemoveEndpoint("categories")
	rem.MakeUpdateEndpoint("categories", &Category{})
	rem.MakeSearchEndpoint("categories", new([]Category))


	// Also (Example)
	//projectList := make([]Project, 0, 10)
	//rem.MakeListEndpoint("projects", &projectList)


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

	http.Handle("/", router)
	http.ListenAndServe("0.0.0.0:8000", nil)
}

func getProject(id string) (project *Project, rerr error) {
	defer func() {
		if err := recover(); err != nil {
			switch t := err.(type) {
			case string:
				rerr = errors.New(t)
			case error:
				rerr = t
			default:
				rerr = errors.New("Unknown Panic")
			}
		}
	}()

	projects := record.AlpacaRecordCollection("projects")
	var projectContainer Project

	if err := projects.FindId(bson.ObjectIdHex(id)).One(&projectContainer); err != nil {
		rerr = err
		return
	}
	project = &projectContainer
	return
}

func getCategory(id string) (category *Category, rerr error) {
	categories := record.AlpacaRecordCollection("categories")
	var categoryContainer Category

	// bson.ObjectIdHex should never panic, if id is well-defined in regex
	// i.e.: [0-9a-f]{24}, since this matches content+length requirements
	if err := categories.FindId(bson.ObjectIdHex(id)).One(&categoryContainer); err != nil {
		rerr = err
		return
	}
	category = &categoryContainer
	return
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
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, PUT");
	w.Header().Set("Access-Control-Max-Age", "3600");
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type");
}

func ContentTypeJson(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (p *Project) DocId() bson.ObjectId {
	return p.Id
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

func (c *Category) DocId() bson.ObjectId {
	return c.Id
}

func (c *Category) IsValid() bool {
	if c.Name == "" {
		return false
	}
	return true
}

func (c *Category) BuildForInsertion() {
	c.Id = bson.NewObjectId()
}