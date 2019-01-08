package spudo

import (
	"log"
	"net/http"
)

func (sp *Spudo) startRESTApi() {
	http.HandleFunc("/", http.NotFound)
	err := http.ListenAndServe(":"+sp.Config.RESTPort, nil)
	if err != nil {
		log.Fatal("Error on creating listener: ", err)
	}
}
