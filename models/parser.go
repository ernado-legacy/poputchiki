package models

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	ContentTypeHeader = "Content-Type"
)

func mapToStruct(q url.Values, val interface{}) error {
	nQ := make(map[string]interface{})
	for key, value := range q {
		log.Printf("%s:%s", key, value)
		if len(value) == 1 {
			v := value[0]
			vInt, err := strconv.Atoi(v)
			if err != nil {
				nQ[key] = v
			} else {
				nQ[key] = vInt
			}
		} else {
			nQ[key] = value
		}
	}
	j, err := json.Marshal(nQ)
	if err != nil {
		return err
	}
	log.Println("json", string(j))
	return json.Unmarshal(j, val)

}

func Parse(r *http.Request, v interface{}) error {
	contentType := r.Header.Get(ContentTypeHeader)
	log.Println(contentType, r.Form, r.PostForm)
	if strings.Index(contentType, "json") != -1 {
		decoder := json.NewDecoder(r.Body)
		return decoder.Decode(v)
	}
	if strings.Index(contentType, "form") != -1 {
		err := r.ParseForm()
		if err != nil {
			log.Println("parse frm", err)
		}
		if r.Method == "GET" {
			log.Printf("GET FORM %+v", r.Form)
			return mapToStruct(r.Form, v)
		}
		log.Printf("POST FORM %+v", r.PostForm)
		return mapToStruct(r.PostForm, v)
	}

	return mapToStruct(r.URL.Query(), v)
}

type Parser struct {
	r *http.Request
}

func (p Parser) Parse(v interface{}) error {
	return Parse(p.r, v)
}

func NewParser(r *http.Request) Parser {
	return Parser{r}
}
