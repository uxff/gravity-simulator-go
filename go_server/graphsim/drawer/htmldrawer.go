package drawer

import (
	"html/template"
	"log"
	"net/http"
)

var routeHome = "/"

var htmlTpl = `
{{.Header}}
<p>width={{.Width}} height={{.Height}}</p>
<table >
<tr>
<td></td>
</tr>
</table>
{{.Footer}}
`

var homeDrawTemplateFunc func(t *template.Template, w http.ResponseWriter)
var homeDrawFunc func(w http.ResponseWriter)

func StartHtmlDrawer(addr string) {
	http.HandleFunc(routeHome, HomeHandler)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Printf("http server error:%v", err)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	//	t, err := template.New("home").Parse(htmlTpl)
	//	if err != nil {
	//		log.Printf("create template error:%v", err)
	//	}

	//t.Execute(w, data)
	if homeDrawFunc != nil {
		homeDrawFunc(w)
	} else {
		w.Write([]byte("template func not exist"))
	}

}

/*
   handleFunc = func(t *template.Template, w http.ResponseWriter) {
       t.Execute(w, struct {UserName string, UserId int, UserRegisterTime time}{"a",2,time.Now()})
   }
*/
func SetHomeDrawTemplateHandler(handleFunc func(t *template.Template, w http.ResponseWriter)) {
	homeDrawTemplateFunc = handleFunc
}

func SetHomeDrawHandler(handleFunc func(w http.ResponseWriter)) {
	homeDrawFunc = handleFunc
}
