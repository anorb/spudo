package catgif

import (
	"fmt"
	"net/http"

	"github.com/anorb/spudo"
)

func catgif(args []string) interface{} {
	res, err := http.Get("http://thecatapi.com/api/images/get?format=src&type=gif")
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	return spudo.NewEmbed().SetColor(0x808080).SetImage(res.Request.URL.String())
}

func init() {
	spudo.AddCommandPlugin("catgif", "gets random cat gif", catgif)
}
