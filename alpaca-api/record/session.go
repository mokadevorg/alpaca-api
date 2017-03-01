package record

import (
	"gopkg.in/mgo.v2"
	"log"
)

var (
	alpacaRecordSession *mgo.Session
)

func InitAlpacaRecord() {
	if alpacaRecordSession != nil {
		log.Fatal("InitAlpacaRecord was already called")
	}

	var err error
	alpacaRecordSession, err = mgo.Dial("mongodb://localhost:27017")
	if err != nil {
		log.Fatal("Database dial error: ", err.Error())
	}
}

func AlpacaRecord() *mgo.Session {
	if alpacaRecordSession == nil {
		InitAlpacaRecord()
	}
	return alpacaRecordSession
}

func AlpacaRecordCollection(coll string) *mgo.Collection {
	return AlpacaRecord().DB("alpaca").C(coll)
}