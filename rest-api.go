package spudo

import (
	"net/http"
)

func (sp *Spudo) startRESTApi() {
	http.HandleFunc("/", http.NotFound)
	err := http.ListenAndServe(":"+sp.Config.RESTPort, nil)
	if err != nil {
		sp.logger.info("Error on creating listener: ", err)
	}
}
