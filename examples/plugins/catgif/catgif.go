package main

import (
	"fmt"
	"net/http"

	"github.com/anorb/spudo/pluginhandler"
	"github.com/anorb/spudo/utils"
)

func catgif(args []string) interface{} {
	res, err := http.Get("http://thecatapi.com/api/images/get?format=src&type=gif")
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	return utils.NewEmbed().SetColor(0x808080).SetImage(res.Request.URL.String())
}

// Register ...
func Register() interface{} {
	return pluginhandler.NewCommand("catgif", catgif)
}
