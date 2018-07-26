package drawer

import (
	"html/template"
	"log"
	"net/http"
)

var routeHome = "/"

var htmlTpl = `
<table >
<tr>
<td></td>
</tr>
</table>
`

var homeDrawFunc func(t *template.Template, w http.ResponseWriter)

func StartHtmlDrawer(addr string) {
	http.HandleFunc(routeHome, HomeHandler)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Printf("http server error:%v", err)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("home").Parse(htmlTpl)
	if err != nil {
		log.Printf("create template error:%v", err)
	}

	//t.Execute(w, data)
	if homeDrawFunc != nil {
		homeDrawFunc(t, w)
	} else {
		w.Write([]byte("template func not exist"))
	}

}

func SetHomeDrawHandler(handleFunc func(t *template.Template, w http.ResponseWriter)) {
	homeDrawFunc = handleFunc
}
